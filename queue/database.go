package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/ortupik/wifigo/server/database/model"
	job "github.com/ortupik/wifigo/server/job"
	"github.com/ortupik/wifigo/websocket"
)


// DatabaseHandler handles database-related tasks.
type DatabaseHandler struct {
	actionHandlers map[string]func(ctx context.Context, raw json.RawMessage) error
	wsHub          *websocket.Hub
}

// NewDatabaseHandler creates a new DatabaseHandler.
func NewDatabaseHandler(wsHub *websocket.Hub) *DatabaseHandler {
	h := &DatabaseHandler{
		actionHandlers: make(map[string]func(ctx context.Context, raw json.RawMessage) error),
		wsHub:          wsHub,
	}
	h.registerHandlers()
	return h
}

func (h *DatabaseHandler) registerHandlers() {
	 h.actionHandlers[ActionSaveMpesaCallback] = h.handleSaveMpesaPayment
	// Add more handlers here as needed
}

// HandleTask processes database-related tasks.
func (h *DatabaseHandler) HandleTask(ctx context.Context, task *asynq.Task) error {
	var payload GenericTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	if payload.System != "mysql" {
		return fmt.Errorf("invalid system for database handler: %s", payload.System)
	}

	log.Printf("Processing MySQL operation: %s", payload.Action)

	handlerFunc, ok := h.actionHandlers[payload.Action]
	if !ok {
		return fmt.Errorf("unknown MySQL action: %s", payload.Action)
	}
	return handlerFunc(ctx, payload.Payload)
}

func (h *DatabaseHandler) handleSaveMpesaPayment(ctx context.Context, raw json.RawMessage) error {
	var data model.MpesaCallbackPayload
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	resp, err := job.SaveMpesaPayment(&data)
	if err != nil {
		return fmt.Errorf("failed to save payment ID: %w", err)
	}

	ip := resp["ip"].(string)
	paymentId := resp["paymentID"].(int)

	log.Printf("Payment saved to database: %v", paymentId)

	h.wsHub.SendToIP(ip, []byte(`{"type":"payment","status":"success","message":"Payment received"}`))
	return nil
}

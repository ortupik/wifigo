package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/ortupik/wifigo/server/database/model"
	service "github.com/ortupik/wifigo/server/service"
	"github.com/ortupik/wifigo/websocket"
)


// DatabaseQueueHandler handles database-related tasks.
type DatabaseQueueHandler struct {
	actionHandlers map[string]func(ctx context.Context, raw json.RawMessage) error
	wsHub          *websocket.Hub
}

// NewDatabaseQueueHandler creates a new DatabaseQueueHandler.
func NewDatabaseQueueHandler(wsHub *websocket.Hub) *DatabaseQueueHandler {
	h := &DatabaseQueueHandler{
		actionHandlers: make(map[string]func(ctx context.Context, raw json.RawMessage) error),
		wsHub:          wsHub,
	}
	h.registerHandlers()
	return h
}

func (h *DatabaseQueueHandler) registerHandlers() {
	 h.actionHandlers[ActionSaveMpesaCallback] = h.handleSaveMpesaPayment
	// Add more handlers here as needed
}

// HandleTask processes database-related tasks.
func (h *DatabaseQueueHandler) HandleTask(ctx context.Context, task *asynq.Task) error {
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

func (h *DatabaseQueueHandler) handleSaveMpesaPayment(ctx context.Context, raw json.RawMessage) error {
    var data model.MpesaCallbackPayload
    if err := json.Unmarshal(raw, &data); err != nil {
        return fmt.Errorf("failed to decode payload: %w", err)
    }

    // Save payment data regardless of success or failure for record keeping
    resp, err := service.SaveMpesaPayment(&data)
    if err != nil {
        return fmt.Errorf("failed to save payment: %w", err)
    }

    paymentID := resp["paymentID"].(int)
    ip, ok := resp["ip"].(string)
    if !ok {
        return fmt.Errorf("failed to get IP for notification")
    }
    
    status := resp["status"].(string)
    message := resp["message"].(string)

    log.Printf("Payment record saved to database (ID: %v, Result: %d - %s)", 
        paymentID, data.ResultCode, data.ResultDesc)

    // Prepare WebSocket notification payload
    wsPayload := map[string]interface{}{
        "type":          "payment",
        "status":        status,
        "message":       message,
        "resultCode":    data.ResultCode,
        "transactionID": data.MpesaReceiptNumber,
        "paymentID":     paymentID,
    }

    // Only include receipt number if payment was successful
    if data.ResultCode == 0 && data.MpesaReceiptNumber != "" {
        wsPayload["receiptNumber"] = data.MpesaReceiptNumber
    }

    // Convert payload to JSON
    wsData, err := json.Marshal(wsPayload)
    if err != nil {
        return fmt.Errorf("failed to marshal WebSocket payload: %w", err)
    }

    // Send notification to client
    h.wsHub.SendToIP(ip, wsData)
    return nil
}

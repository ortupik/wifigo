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

    // Define message based on ResultCode
    status := "error"
    message := "An unknown error occurred"

    // Map common M-Pesa result codes to user-friendly messages
    switch data.ResultCode {
    case 0:
        status = "success"
        message = "Payment received successfully"
    case 1:
        message = "Insufficient balance"
    case 1001:
        message = "Payment is being processed"
    case 1002:
        message = "Payment request is being processed"
    case 1031:
        message = "Request cancelled by user"
    case 1032:
        message = "The request was canceled by the user"
    case 1037:
        message = "Phone cannot be reached!"
    case 2001:
        message = "Wrong PIN provided"
    case 17:
        message = "User account does not exist"
    case 20:
        message = "User account is inactive"
    case 26:
        message = "Payment request timed out"
    default:
        message = fmt.Sprintf("Payment failed: %s", data.ResultDesc)
    }

    // Save payment data regardless of success or failure for record keeping
    resp, err := job.SaveMpesaPayment(&data)
    if err != nil {
        return fmt.Errorf("failed to save payment: %w", err)
    }

    paymentID := resp["paymentID"].(int)
    ip, ok := resp["ip"].(string)
    if !ok {
        return fmt.Errorf("failed to get IP for notification")
    }

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

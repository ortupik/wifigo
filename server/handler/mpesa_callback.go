package handler

import (
	"context" // Import context for goroutine cancellation/timeout
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync" // Import sync for WaitGroup
	"time" // For parsing TransactionDate

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/queue"
	"github.com/ortupik/wifigo/server/database/model"
	dto "github.com/ortupik/wifigo/server/dto"
	"github.com/ortupik/wifigo/websocket"
)

// MpesaCallbackHandler manages M-Pesa callbacks and related operations.
type MpesaCallbackHandler struct {
	queue *queue.Client
	wsHub *websocket.Hub
}

// NewMpesaCallbackHandler creates a new instance of MpesaCallbackHandler.
func NewMpesaCallbackHandler(queueClient *queue.Client, wsHub *websocket.Hub) *MpesaCallbackHandler {
	return &MpesaCallbackHandler{
		queue: queueClient,
		wsHub: wsHub,
	}
}

// MpesaStkHandlerCallback processes incoming M-Pesa STK push callbacks.
func (h *MpesaCallbackHandler) MpesaStkHandlerCallback(c *gin.Context) {
	// 1. Parse raw Safaricom callback
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid M-Pesa callback payload."})
		return
	}

	data, err := json.Marshal(raw) // Marshal back to bytes for ParseCallback
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize callback data."})
		return
	}

	payload, err := ParseCallback(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse M-Pesa callback details.", "details": err.Error()})
		return
	}

	// 2. Look up matching order
	db := gdatabase.GetDB(config.AppDB)
	var order model.Order
	if err := db.Preload("ServicePlan").Where("CheckoutRequestID = ?", payload.CheckoutRequestID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated order not found for this M-Pesa callback."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order from database.", "details": err.Error()})
		return
	}

	// 3. Handle failed M-Pesa payments (ResultCode != 0)
	if payload.ResultCode != 0 {
		// Enqueue for reporting/audit. This is independent and non-critical for the HTTP response.
		go func() {
			_, err := h.queue.EnqueueDatabaseOperation(context.Background(), queue.ActionSaveMpesaCallback, *payload, queue.QueueReporting)
			if err != nil {
				fmt.Printf("WARNING: Failed to enqueue failed M-Pesa callback for reporting: %v\n", err)
			}
		}()
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"payment", "status": "failed", "message": %q}`, payload.ResultDesc)))
		c.JSON(http.StatusOK, gin.H{"status": "Failed payment received and processed.", "ResultDesc": payload.ResultDesc})
		return
	}

	// 4. Prepare subscription and manage hotspot user (synchronous RADIUS operation)
	subscription := dto.HotspotSubscriptionRequest{
		Phone:       payload.PhoneNumber,
		Username:    order.Username,
		IsHomeUser:  order.IsHomeUser,
		ISP:         order.ISP,
		ServiceName: order.ServicePlan.Name,
		Duration:    order.ServicePlan.Duration,
		Devices:     order.Devices,
	}

	// ManageHotspotUser is assumed to be a blocking call to a RADIUS management API
	resp, manageStatus := ManageHotspotUser(subscription, true) // Renamed 'status' to 'manageStatus' to avoid conflict
	if manageStatus == http.StatusInternalServerError {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create/manage RADIUS user."})
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "failed", "message": %q}`, "Failed to create Account")))
		return
	} else if manageStatus == http.StatusConflict {
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "failed", "message": %q}`, "User already subscribed")))
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"payment", "status": "success", "message": %q}`, "Payment already done")))
	} else { // http.StatusOK or other success codes from ManageHotspotUser
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "success", "message": %q}`, "Account created/updated successfully")))
	}

	// Extract password safely
	password, _ := resp["password"].(string) // Assumes "" if not present or not string

	loginPayload := dto.MikrotikLogin{
		DeviceID: order.DeviceID,
		Address:  order.Ip,
		Username: order.Username, // 'username' already defined from order.Username
		Password: password,
	}

	// 5. Enqueue Mikrotik Command and Database Operation Independently and Concurrently
	var wg sync.WaitGroup
	wg.Add(2) // We have two operations to wait for

	// Use channels to capture errors from goroutines
	mikrotikErrCh := make(chan error, 1) // Buffered channel to avoid deadlock
	dbErrCh := make(chan error, 1)

	// Goroutine for Mikrotik Login Command
	go func() {
		defer wg.Done()
		if _, err := h.queue.EnqueueMikrotikCommand(c.Request.Context(), queue.ActionMikrotikLoginUser, loginPayload, queue.QueueCritical); err != nil {
			mikrotikErrCh <- fmt.Errorf("failed to enqueue Mikrotik login command: %w", err)
		} else {
			mikrotikErrCh <- nil // Send nil on success
		}
	}()

	// Goroutine for Database Operation (Save Mpesa Callback)
	go func() {
		defer wg.Done()
		// Only enqueue DB operation if it's not a conflict (i.e., not already paid)
		if manageStatus != http.StatusConflict {
			if _, err := h.queue.EnqueueDatabaseOperation(c.Request.Context(), queue.ActionSaveMpesaCallback, *payload, queue.QueueCritical); err != nil {
				dbErrCh <- fmt.Errorf("failed to enqueue DB save operation for Mpesa callback: %w", err)
			} else {
				dbErrCh <- nil // Send nil on success
			}
		} else {
			dbErrCh <- nil // If skipped due to conflict, treat as no error for this operation
		}
	}()

	// Wait for both concurrent operations to finish
	wg.Wait()

	// Close channels to prevent goroutine leaks (good practice)
	close(mikrotikErrCh)
	close(dbErrCh)

	// Collect results
	var mikrotikQueueError error = <-mikrotikErrCh
	var dbQueueError error = <-dbErrCh

	// 6. Formulate HTTP Response based on combined outcomes
	responseStatus := http.StatusOK
	responseMessage := "Payment processed successfully."
	responseErrors := make(map[string]string)

	if mikrotikQueueError != nil {
		responseErrors["mikrotik_queue_error"] = mikrotikQueueError.Error()
		responseStatus = http.StatusInternalServerError // Mikrotik failure is critical
		responseMessage = "Payment processed, but Mikrotik command failed to enqueue."
	}

	if dbQueueError != nil {
		responseErrors["db_queue_error"] = dbQueueError.Error()
		if mikrotikQueueError == nil { // Only downgrade if Mikrotik was okay
			responseStatus = http.StatusAccepted // Accept the payment, but warn about DB
			responseMessage = "Payment processed, Mikrotik command enqueued, but DB save failed."
		} else {
			responseMessage += " Also, DB save failed." // Both failed
		}
	}

	if len(responseErrors) > 0 {
		c.JSON(responseStatus, gin.H{
			"status":  "partial_failure",
			"message": responseMessage,
			"errors":  responseErrors,
		})
	} else {
		c.JSON(responseStatus, gin.H{"status": "success", "message": responseMessage})
	}
}

// --- Safaricom M-Pesa Callback Parser (moved to be a method of the handler if it needs access to members, or keep as global if stateless) ---
// Note: It's often good practice to have stateless utility functions as package-level functions
// or put them in a dedicated `util` or `parser` package. For now, it's fine here.

// ParseCallback parses the raw M-Pesa STK push callback JSON into a structured payload.
func ParseCallback(data []byte) (*model.MpesaCallbackPayload, error) {
	var raw struct {
		Body struct {
			StkCallback model.StkCallback `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal M-Pesa callback raw data error: %w", err)
	}

	cb := raw.Body.StkCallback
	payload := &model.MpesaCallbackPayload{
		MerchantRequestID: cb.MerchantRequestID,
		CheckoutRequestID: cb.CheckoutRequestID,
		ResultCode:        cb.ResultCode,
		ResultDesc:        cb.ResultDesc,
	}

	if cb.CallbackMetadata != nil {
		for _, item := range cb.CallbackMetadata.Item {
			switch item.Name {
			case "Amount":
				if val, ok := item.Value.(float64); ok {
					payload.Amount = decimal.NewFromFloat(val)
				}
			case "MpesaReceiptNumber":
				if val, ok := item.Value.(string); ok {
					payload.MpesaReceiptNumber = val
				}
			case "TransactionDate":
				// M-Pesa transaction date format: YYYYMMDDHHmmss
				if val, ok := item.Value.(string); ok {
					t, err := time.Parse("20060102150405", val)
					if err == nil {
						payload.TransactionDate = t.Format("2006-01-02 15:04:05") // Store in a standard format
					} else {
						fmt.Printf("WARNING: Failed to parse M-Pesa TransactionDate %q: %v\n", val, err)
						payload.TransactionDate = val // Store raw string if parsing fails
					}
				}
			case "PhoneNumber":
				if val, ok := item.Value.(float64); ok {
					// Format phone number to start with 254 (e.g., 2547XXXXXXXX)
					// Assumes M-Pesa provides 254XXXXXXXXX or 07XXXXXXXX
					// Using Sprintf to handle the float conversion to string cleanly
					rawPhone := fmt.Sprintf("%.0f", val)
					if len(rawPhone) == 9 && (rawPhone[0] == '7' || rawPhone[0] == '1') { // Starts with 7 or 1, is 9 digits (e.g., 7XXXXXXX)
						payload.PhoneNumber = "254" + rawPhone
					} else if len(rawPhone) == 12 && rawPhone[:3] == "254" { // Already 254...
						payload.PhoneNumber = rawPhone
					} else if len(rawPhone) == 10 && rawPhone[0] == '0' { // Starts with 0, is 10 digits
						payload.PhoneNumber = "254" + rawPhone[1:]
					} else {
						payload.PhoneNumber = rawPhone // Fallback: store as is
						fmt.Printf("WARNING: Unexpected M-Pesa PhoneNumber format: %s\n", rawPhone)
					}
				}
			}
		}
	}

	return payload, nil
}

package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

type MpesaCallbackHandler struct {
	queue *queue.Client
	wsHub *websocket.Hub
}

func NewMpesaCallbackHandler(queueClient *queue.Client, wsHub *websocket.Hub) *MpesaCallbackHandler {
	return &MpesaCallbackHandler{
		queue: queueClient,
		wsHub: wsHub,
	}
}

func (h *MpesaCallbackHandler) MpesaHandlerCallback(c *gin.Context) {
	// Parse raw Safaricom callback
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid M-Pesa callback"})
		return
	}

	data, _ := json.Marshal(raw)
	payload, err := ParseCallback(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse callback", "details": err.Error()})
		return
	}

	// Look up matching order
	db := gdatabase.GetDB(config.AppDB)
	var order model.Order
	if err := db.Preload("ServicePlan").Where("CheckoutRequestID = ?", payload.CheckoutRequestID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order", "details": err})
		return
	}

	// Handle failed payments (log or queue for audit)
	if payload.ResultCode != 0 {
		_, _ = h.queue.EnqueueDatabaseOperation(c.Request.Context(), queue.ActionSaveMpesaCallback, payload, queue.QueueReporting)
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"payment", "status": "failed", "message": %q}`, payload.ResultDesc)))
		c.JSON(http.StatusOK, gin.H{"status": "Ignored failed payment", "ResultDesc": payload.ResultDesc})
		return
	}

	
	// Prepare subscription
	subscription := dto.HotspotSubscriptionRequest{
		Phone:       payload.PhoneNumber,
		Username:    order.Username,
		IsHomeUser:  order.IsHomeUser,
		ISP:         order.ISP,
		ServiceName: order.ServicePlan.Name,
		Duration:    order.ServicePlan.Duration,
		Devices:     order.Devices,
	}

	resp, status := ManageHotspotUser(subscription, true)
	if status == http.StatusInternalServerError {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RADIUS user"})
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "failed", "message": %q}`, "Failed to create Account")))
		return
	} else if status == http.StatusConflict {
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "failed", "message": %q}`, "User already subscribed")))
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"payment", "status": "success", "message": %q}`, "Payment already done")))
	}else{
		h.wsHub.SendToIP(order.Ip, []byte(fmt.Sprintf(`{"type":"create_account", "status": "success", "message": %q}`, "Account created")))
	}

	username := order.Username
	args := []string{"=user=" + username, "=ip=" + order.Ip}
	if password, ok := resp["password"].(string); ok && password != "" {
		args = append(args, "=password="+password)
	}

	// Queue Mikrotik login
	loginPayload := &queue.MikrotikCommandPayload{
		DeviceID: order.DeviceID,
		Command:  "/ip/hotspot/active/login",
		Args:     args,
		Ip:       order.Ip,
	}
	if _, err := h.queue.EnqueueMikrotikCommand(c.Request.Context(), loginPayload, queue.QueueCritical); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue Mikrotik login"})
		return
	}

	//If already paid dont update DB
	if(status != http.StatusConflict ){
		if _, err := h.queue.EnqueueDatabaseOperation(c.Request.Context(), queue.ActionSaveMpesaCallback, payload, queue.QueueCritical); err != nil {
			c.JSON(http.StatusAccepted, gin.H{"warning": "Login queued, but DB save failed"})
			return
		}
	}
	

	c.JSON(http.StatusOK, gin.H{"status": "Payment processed"})
}

// Safaricom M-Pesa Callback Parser
func ParseCallback(data []byte) (*model.MpesaCallbackPayload, error) {
	var raw struct {
		Body struct {
			StkCallback model.StkCallback `json:"stkCallback"`
		} `json:"Body"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
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
				if val, ok := item.Value.(string); ok {
					payload.TransactionDate = val
				}
				/*if val, ok := item.Value.(float64); ok {
					timestamp := fmt.Sprintf("%d", int64(val))
					if t, err := time.Parse("20060102150405", timestamp); err == nil {
						payload.TransactionDate = t
					}
				}*/
			case "PhoneNumber":
				if val, ok := item.Value.(float64); ok {
					payload.PhoneNumber = fmt.Sprintf("254%.0f", val-254000000000)
				}
			}
		}
	}

	return payload, nil
}

package handler

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/mikrotik"
	badger "github.com/ortupik/wifigo/badger"
	"github.com/ortupik/wifigo/queue"
)

// MikrotikHandler handles MikroTik-related operations
type MikrotikHandler struct {
	store    *badger.Store // Storage layer for MikroTik storage.
	manager  *mikrotik.Manager
	queue    *queue.Client
}

// NewMikrotikHandler creates a new MikroTik handler
func NewMikrotikHandler(store *badger.Store, manager *mikrotik.Manager, queueClient *queue.Client) *MikrotikHandler {
	return &MikrotikHandler{
		store:   store,
		manager: manager,
		queue:   queueClient,
	}
}

// ListDevices lists all MikroTik devices for the ISP
func (h *MikrotikHandler) ListDevices(c *gin.Context) {
	// Get ISP ID from session or JWT token
	// This is a simplified example; you'd need authentication middleware
	ispID := c.GetString("ispID")
	if ispID == "" {
		ispID = "isp1" // Default for testing
	}
	
	devices, err := h.store.ListDeviceConfigsByISP(ispID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list devices: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
	})
}

// SelectDevice selects a device for the current session
func (h *MikrotikHandler) SelectDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	var deviceWrapper badger.DeviceConfigWrapper
	err := h.store.GetConfig(badger.DeviceConfigType, deviceID, &deviceWrapper)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Device not found: " + err.Error(),
		})
		return
	}
	
	// Store in session
	session := sessions.Default(c)
	session.Set("selected_device", deviceID)
	err = session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save session: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Device selected successfully",
		"deviceID": deviceID,
	})
}

// GetSelectedDevice gets the currently selected device
func (h *MikrotikHandler) GetSelectedDevice(c *gin.Context) {
	session := sessions.Default(c)
	//deviceID := session.Get("selected_device")
	deviceID := "mikrotik1" //default selected one since sesion NOT working

	if deviceID == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No device selected",
			"session": session,
		})
		return
	}
	
	var deviceWrapper badger.DeviceConfigWrapper
	err := h.store.GetConfig(badger.DeviceConfigType, deviceID, &deviceWrapper)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get device: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"device": deviceWrapper.DeviceConfig,
	})
}

// ExecuteCommand queues a command to be executed on the selected MikroTik device
func (h *MikrotikHandler) ExecuteCommand(c *gin.Context) {
	// Get selected device from session
	//session := sessions.Default(c)
	//deviceID := session.Get("selected_device")
	deviceID := "mikrotik1" //default selected one since sesion NOT working
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No device selected",
		})
		return
	}
	
	// Parse command from request
	var req struct {
		Command  string   `json:"command" binding:"required"`
		Args     []string `json:"args"`
		Priority string   `json:"priority"`
		Ip       string   `json:"ip"`
        CallbackURL string `json:"callbackUrl"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}
	
	// Set default priority if not specified
	if req.Priority == "" {
		req.Priority = queue.QueueDefault
	}
	
	// Create command payload
	payload := &queue.MikrotikCommandPayload{
		DeviceID: deviceID,
		Command:  req.Command,
		Args:     req.Args,
		Ip        : req.Ip,
        CallbackURL: req.CallbackURL,
	}
	
	// Enqueue command
	taskInfo, err := h.queue.EnqueueMikrotikCommand(c.Request.Context(), payload, req.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to queue command: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Command queued successfully",
		"task_id":  taskInfo.ID,
		"priority": req.Priority,
		"queued_at": "TODO",
	})
}


// GetCommandStatus gets the status of a queued command
func (h *MikrotikHandler) GetCommandStatus(c *gin.Context) {
	taskID := c.Param("taskID")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}
	
	// In a real implementation, you would query the task status from Redis
	// This is a placeholder implementation
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "pending", // or "processing", "completed", "failed"
	})
}
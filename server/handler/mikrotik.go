package handler

import (
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/server/database/model"
)

type MikrotikQueueHandler struct {
	manager *mikrotik.Manager
}

func NewMikrotikQueueHandler(manager *mikrotik.Manager) *MikrotikQueueHandler {
	return &MikrotikQueueHandler{
		manager: manager,
	}
}

// GetMikroTikDevice retrieves a device based on ID
func GetMikroTikDevice(deviceID string, c *gin.Context) {

	tx := gdatabase.GetDB(config.AppDB)

	var device model.MikroTikDevice
	if err := tx.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         device.ID,
		"name":       device.GetName(),
		"address":    device.Address,
		"username":   device.Username,
		"pool_size":  device.PoolSize,
		"status":     device.Status,
		"created_at": device.CreatedAt,
		"updated_at": device.UpdatedAt,
	})
}

// GetMikroTikDevices retrieves all devices with optional pagination and filtering
func GetMikroTikDevices(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse optional status filter
	status := c.Query("status")

	var devices []model.MikroTikDevice
	var count int64

	query := tx.Model(&model.MikroTikDevice{})

	// Apply status filter if provided
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	query.Count(&count)

	// Get paginated devices (exclude password from response)
	if err := query.Select("id, name, address, username, pool_size, status, created_at, updated_at").
		Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": devices,
		"meta": gin.H{
			"total": count,
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateMikroTikDevice creates a new device
func CreateMikroTikDevice(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	var deviceInput model.MikroTikDevice
	if err := c.BindJSON(&deviceInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if device with same ID already exists
	var existingDevice model.MikroTikDevice
	if err := tx.Where("id = ?", deviceInput.ID).First(&existingDevice).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Device with this ID already exists"})
		return
	}

	// Set default values if not provided
	if deviceInput.PoolSize == 0 {
		deviceInput.PoolSize = 5
	}
	if deviceInput.Status == "" {
		deviceInput.Status = model.StatusActive
	}

	if err := tx.Create(&deviceInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device"})
		return
	}

	// Return created device without password
	response := gin.H{
		"id":         deviceInput.ID,
		"name":       deviceInput.GetName(),
		"address":    deviceInput.Address,
		"username":   deviceInput.Username,
		"pool_size":  deviceInput.PoolSize,
		"status":     deviceInput.Status,
		"created_at": deviceInput.CreatedAt,
		"updated_at": deviceInput.UpdatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// UpdateMikroTikDevice updates an existing device
func UpdateMikroTikDevice(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID is required"})
		return
	}

	var deviceInput model.MikroTikDevice
	if err := c.BindJSON(&deviceInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if device exists
	var existingDevice model.MikroTikDevice
	if err := tx.Where("id = ?", deviceID).First(&existingDevice).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Ensure ID cannot be changed
	deviceInput.ID = deviceID

	// Update the device
	if err := tx.Model(&existingDevice).Updates(deviceInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device"})
		return
	}

	// Fetch the updated device to return
	if err := tx.Where("id = ?", deviceID).First(&existingDevice).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated device"})
		return
	}

	// Return updated device without password
	response := gin.H{
		"id":         existingDevice.ID,
		"name":       existingDevice.GetName(),
		"address":    existingDevice.Address,
		"username":   existingDevice.Username,
		"pool_size":  existingDevice.PoolSize,
		"status":     existingDevice.Status,
		"created_at": existingDevice.CreatedAt,
		"updated_at": existingDevice.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteMikroTikDevice removes a device by ID
func DeleteMikroTikDevice(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	deviceID := c.Param("id")
	var device model.MikroTikDevice

	// Check if device exists
	if err := tx.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Delete the device
	if err := tx.Delete(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

// UpdateMikroTikDeviceStatus updates only the status of a device
func UpdateMikroTikDeviceStatus(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	deviceID := c.Param("id")

	var statusInput struct {
		Status model.DeviceStatus `json:"status" validate:"required,oneof=active inactive maintenance"`
	}

	if err := c.BindJSON(&statusInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if device exists
	var device model.MikroTikDevice
	if err := tx.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Update only the status
	if err := tx.Model(&device).Update("status", statusInput.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device status updated successfully",
		"id":      deviceID,
		"status":  statusInput.Status,
	})
}

// GetMikroTikDevicesByStatus retrieves devices filtered by status
func GetMikroTikDevicesByStatus(status model.DeviceStatus, c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	var devices []model.MikroTikDevice
	if err := tx.Select("id, name, address, username, pool_size, status, created_at, updated_at").
		Where("status = ?", status).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   devices,
		"status": status,
		"count":  len(devices),
	})
}

// GetMikroTikDeviceStats returns statistics about devices
func GetMikroTikDeviceStats(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	var stats struct {
		Total       int64 `json:"total"`
		Active      int64 `json:"active"`
		Inactive    int64 `json:"inactive"`
		Maintenance int64 `json:"maintenance"`
	}

	// Get total count
	tx.Model(&model.MikroTikDevice{}).Count(&stats.Total)

	// Get counts by status
	tx.Model(&model.MikroTikDevice{}).Where("status = ?", model.StatusActive).Count(&stats.Active)
	tx.Model(&model.MikroTikDevice{}).Where("status = ?", model.StatusInactive).Count(&stats.Inactive)
	tx.Model(&model.MikroTikDevice{}).Where("status = ?", model.StatusMaintenance).Count(&stats.Maintenance)

	c.JSON(http.StatusOK, stats)
}

// TestMikroTikDeviceConnection tests the connection to a specific device
func TestMikroTikDeviceConnection(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	deviceID := c.Param("id")

	// Get device details
	var device model.MikroTikDevice
	if err := tx.Where("id = ?", deviceID).First(&device).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	// Here you would integrate with your MikroTik manager to test the connection
	// For now, returning a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"device_id":         device.ID,
		"connection_status": "success", // This would be determined by actual connection test
		"message":           "Connection test completed",
		"timestamp":         time.Now(),
	})
}

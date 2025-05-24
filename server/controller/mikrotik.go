package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/server/database/model"
	"github.com/ortupik/wifigo/server/handler"
)

type MikroTikController struct {
	handler *handler.MikrotikQueueHandler
}

func NewMikroTikController(manager *mikrotik.Manager) *MikroTikController {
	return &MikroTikController{
		handler: handler.NewMikrotikQueueHandler(manager),
	}
}

// GetDevice handles GET /devices/:id
func (ctrl *MikroTikController) GetDevice(c *gin.Context) {
	deviceID := c.Param("id")
	handler.GetMikroTikDevice(deviceID, c)
}

// GetDevices handles GET /devices
func (ctrl *MikroTikController) GetDevices(c *gin.Context) {
	handler.GetMikroTikDevices(c, nil)
}

// CreateDevice handles POST /devices
func (ctrl *MikroTikController) CreateDevice(c *gin.Context) {
	handler.CreateMikroTikDevice(c, nil)
}

// UpdateDevice handles PUT /devices/:id
func (ctrl *MikroTikController) UpdateDevice(c *gin.Context) {
	handler.UpdateMikroTikDevice(c, nil)
}

// DeleteDevice handles DELETE /devices/:id
func (ctrl *MikroTikController) DeleteDevice(c *gin.Context) {
	handler.DeleteMikroTikDevice(c, nil)
}

// UpdateDeviceStatus handles PATCH /devices/:id/status
func (ctrl *MikroTikController) UpdateDeviceStatus(c *gin.Context) {
	handler.UpdateMikroTikDeviceStatus(c, nil)
}

// GetDevicesByStatus handles GET /devices/status/:status
func (ctrl *MikroTikController) GetDevicesByStatus(c *gin.Context) {
	statusParam := c.Param("status")
	status := model.DeviceStatus(statusParam)
	handler.GetMikroTikDevicesByStatus(status, c, nil)
}

// GetDeviceStats handles GET /devices/stats
func (ctrl *MikroTikController) GetDeviceStats(c *gin.Context) {
	handler.GetMikroTikDeviceStats(c, nil)
}

// TestDeviceConnection handles POST /devices/:id/test
func (ctrl *MikroTikController) TestDeviceConnection(c *gin.Context) {
	handler.TestMikroTikDeviceConnection(c, nil)
}

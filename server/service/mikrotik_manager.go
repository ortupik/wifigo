package service

import (
	"fmt"
	"sync"

	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/server/database/model"
	"gorm.io/gorm"
)

// MikroTikService handles the integration between database and MikroTik manager
type MikroTikMangerService struct {
	db      *gorm.DB
	manager *mikrotik.Manager
	mu      sync.RWMutex
}

// NewMikroTikService creates a new MikroTik service
func NewMikroTikManagerService(manager *mikrotik.Manager) *MikroTikMangerService {
	return &MikroTikMangerService{
		db:      gdatabase.GetDB(config.AppDB),
		manager: manager,
	}
}

// LoadAllDevices loads all active devices from database into the manager
func (s *MikroTikMangerService) LoadAllDevices() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var devices []model.MikroTikDevice
	if err := s.db.Where("status = ?", model.StatusActive).Find(&devices).Error; err != nil {
		return fmt.Errorf("failed to load devices from database: %w", err)
	}

	for _, device := range devices {
		deviceConfig := config.DeviceConfig{
			ID:       device.ID,
			Address:  device.Address,
			Username: device.Username,
			Password: device.Password,
			PoolSize: device.PoolSize,
			Port:     device.Port,
			// Add ISPID if you have realm relationships
		}

		if err := s.manager.AddDevice(deviceConfig); err != nil {
			return fmt.Errorf("failed to add device %s to manager: %w", device.ID, err)
		}
	}

	return nil
}

// AddDeviceToManager adds a device to both database and manager
func (s *MikroTikMangerService) AddDeviceToManager(device *model.MikroTikDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First save to database
	if err := s.db.Create(device).Error; err != nil {
		return fmt.Errorf("failed to save device to database: %w", err)
	}

	// Then add to manager if active
	if device.IsActive() {
		deviceConfig := config.DeviceConfig{
			ID:       device.ID,
			Address:  device.Address,
			Port:  device.Port,
			Username: device.Username,
			Password: device.Password,
			PoolSize: device.PoolSize,
		}

		if err := s.manager.AddDevice(deviceConfig); err != nil {
			// Rollback database operation if manager fails
			s.db.Delete(device)
			return fmt.Errorf("failed to add device to manager: %w", err)
		}
	}

	return nil
}

// UpdateDeviceInManager updates a device in both database and manager
func (s *MikroTikMangerService) UpdateDeviceInManager(deviceID string, updates *model.MikroTikDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get existing device
	var existingDevice model.MikroTikDevice
	if err := s.db.Where("id = ?", deviceID).First(&existingDevice).Error; err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// Update database
	if err := s.db.Model(&existingDevice).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update device in database: %w", err)
	}

	// Remove from manager first
	s.manager.RemoveDevice(deviceID)

	// Re-add to manager if active
	if updates.IsActive() {
		deviceConfig := config.DeviceConfig{
			ID:       updates.ID,
			Address:  updates.Address,
			Port:     updates.Port,
			Username: updates.Username,
			Password: updates.Password,
			PoolSize: updates.PoolSize,
		}

		if err := s.manager.AddDevice(deviceConfig); err != nil {
			return fmt.Errorf("failed to re-add device to manager: %w", err)
		}
	}

	return nil
}

// RemoveDeviceFromManager removes a device from both database and manager
func (s *MikroTikMangerService) RemoveDeviceFromManager(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from manager first
	s.manager.RemoveDevice(deviceID)

	// Then remove from database
	if err := s.db.Where("id = ?", deviceID).Delete(&model.MikroTikDevice{}).Error; err != nil {
		return fmt.Errorf("failed to delete device from database: %w", err)
	}

	return nil
}

// GetDevicePool gets a device pool from the manager
func (s *MikroTikMangerService) GetDevicePool(deviceID string) (*mikrotik.DevicePool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.manager.GetDevice(deviceID)
}

// Close closes all connections
func (s *MikroTikMangerService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.manager != nil {
		s.manager.Close()
	}
}


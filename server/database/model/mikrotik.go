package model

import (
	"time"
)

// DeviceStatus represents the status enum for MikroTik devices
type DeviceStatus string

const (
	StatusActive      DeviceStatus = "active"
	StatusInactive    DeviceStatus = "inactive"
	StatusMaintenance DeviceStatus = "maintenance"
)

// MikroTikDevice represents a MikroTik device configuration
type MikroTikDevice struct {
	ID        string       `json:"id" gorm:"primaryKey;size:64" validate:"required,max=64"`
	Name      *string      `json:"name" gorm:"size:128" validate:"omitempty,max=128"`
	Address   string       `json:"address" gorm:"size:128;not null" validate:"required,max=128"`
	Port   string          `json:"port" gorm:"size:128;not null" validate:"required,max=128"`
	Username  string       `json:"username" gorm:"size:64;not null" validate:"required,max=64"`
	Password  string       `json:"password" gorm:"size:255;not null" validate:"required,max=255"`
	PoolSize  int          `json:"pool_size" gorm:"default:5" validate:"min=1,max=100"`
	CreatedAt time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
	Status    DeviceStatus `json:"status" gorm:"type:enum('active','inactive','maintenance');default:'active'" validate:"oneof=active inactive maintenance"`
}

// TableName specifies the table name for GORM
func (MikroTikDevice) TableName() string {
	return "devices"
}

// NewMikroTikDevice creates a new MikroTik device with default values
func NewMikroTikDevice(id, name, address, port, username, password string) *MikroTikDevice {
	return &MikroTikDevice{
		ID:       id,
		Name:     &name,
		Address:  address,
		Port:     port,
		Username: username,
		Password: password,
		PoolSize: 5,
		Status:   StatusActive,
	}
}

// SetName sets the device name
func (d *MikroTikDevice) SetName(name string) {
	if name == "" {
		d.Name = nil
	} else {
		d.Name = &name
	}
}

// GetName returns the device name or empty string if not set
func (d *MikroTikDevice) GetName() string {
	if d.Name == nil {
		return ""
	}
	return *d.Name
}

// IsActive returns true if the device status is active
func (d *MikroTikDevice) IsActive() bool {
	return d.Status == StatusActive
}

// Activate sets the device status to active
func (d *MikroTikDevice) Activate() {
	d.Status = StatusActive
}

// Deactivate sets the device status to inactive
func (d *MikroTikDevice) Deactivate() {
	d.Status = StatusInactive
}

// SetMaintenance sets the device status to maintenance
func (d *MikroTikDevice) SetMaintenance() {
	d.Status = StatusMaintenance
}
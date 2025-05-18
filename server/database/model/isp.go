package model

import (
	"time"
)

type ServiceType string

const (
	ServiceTypeHotspot ServiceType = "Hotspot"
	ServiceTypeHome    ServiceType = "Home"
)

// ISP struct represents an Internet Service Provider.
type ISP struct {
	ID           int64          `gorm:"primaryKey;autoIncrement;column:id"`
	Name         string         `gorm:"column:name"`
	LogoURL      string         `gorm:"column:logoUrl"`
	DeviceID     *string        `gorm:"column:deviceId"` // Device ID, nullable, for ISP-level association if needed
	ServicePlans []ServicePlan  `gorm:"foreignKey:ISPID"` // One-to-Many: ISP has many ServicePlans
	DnsName      string         `gorm:"column:dns_name"`
}

// ServicePlan struct represents a service plan offered by the ISP.
type ServicePlan struct {
	ID            int          `gorm:"primaryKey;autoIncrement;column:id"`
	Name          string       `gorm:"uniqueIndex;type:varchar(255);column:name"`
	ServiceType   ServiceType  `gorm:"column:serviceType;default:'Hotspot'"` // Correct default
	Description   string      `gorm:"column:description"`
	Price         int          `gorm:"column:price"`
	Duration      int          `gorm:"column:duration"`
	DataLimitMB   *int         `gorm:"column:dataLimitMB"`
	SpeedLimitMbps string       `gorm:"column:speedLimitMbps"`
	IsActive      bool         `gorm:"column:isActive;default:true"`
	Validity      string       `gorm:"column:validity"`
	Speed         string       `gorm:"column:speed"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ISPID         int64        `gorm:"column:isp_id"` // Foreign Key to ISP
	// DeviceID is now NOT in ServicePlan
	//Orders      []Order    // One-to-Many relationship: Orders for this service plan - removed for now
}

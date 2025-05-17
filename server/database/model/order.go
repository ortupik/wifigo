package model // You can change this package name if needed

import (
	"time"
)

// ServicePlan - Defines the different service plans available
type ServicePlan struct {
	ID             int             `gorm:"primaryKey;autoIncrement;column:id"`
	Name           string          `gorm:"uniqueIndex;type:varchar(255);column:name"`             // Name of the service plan
	Description    *string         `gorm:"column:description"`                  // Nullable
	Price          int             `gorm:"column:price"`     // Price of the plan
	Duration       int             `gorm:"column:duration"`                     // Duration in seconds (not nullable)
	DataLimitMB    *int            `gorm:"column:dataLimitMB"`                  // Data limit in MB (nullable)
	SpeedLimitMbps string          `gorm:"column:speedLimitMbps"`              // Speed limit (not nullable String)
	IsActive       bool            `gorm:"column:isActive;default:true"`        // Boolean with default

	// GORM handles @default(now()) and @updatedAt automatically for these field names
	CreatedAt time.Time
	UpdatedAt time.Time

	Orders []Order // One-to-Many relationship: Orders for this service plan
}

// Order - Represents a user's order for a service plan
type Order struct {
	ID                int             `gorm:"primaryKey;autoIncrement;column:id"`
	OrderNumber       string          `gorm:"uniqueIndex;type:varchar(255);column:orderNumber;index:orderNumber"` // Unique order identifier
	Status            string          `gorm:"column:status;index:status"`                      // Status of the order
	Amount            int             `gorm:"column:amount"`                 // Total amount
	Username          string          `gorm:"column:username;"`
	Ip                string          `gorm:"column:ip;"`
	Mac               string          `gorm:"column:mac;"`
	Phone             string          `gorm:"column:phone;index:phone"`
	CheckoutRequestID string          `gorm:"column:CheckoutRequestID;index:checkoutRequestID"`
	MerchantRequestID string          `gorm:"column:MerchantRequestID;"`
	ResponseCode      string             `gorm:"column:ResponseCode;ResponseCode"`
	ResultDesc        string          `gorm:"column:ResultDesc;index:resultDesc"`
	ISP               string          `gorm:"column:isp;index:isp"`
	Zone              string          `gorm:"column:zone;"`
	DeviceID          string          `gorm:"column:DeviceID;"`
	IsHomeUser        bool           `gorm:"column:isHomeUser;default:false"` // Nullable boolean with default false
	Devices           int             `gorm:"column:devices;default:1;not null"`

	// Link to the Service Plan ordered (non-nullable)
	ServicePlanID int         `gorm:"column:servicePlanId;index:servicePlanId"` // Foreign key field for ServicePlan
	ServicePlan   ServicePlan // Relationship field to ServicePlan (non-pointer)

	Payments []Payment `gorm:"foreignKey:OrderID"` // One-to-Many relationship: Payments towards this order.

	// GORM handles @default(now()) and @updatedAt automatically for these field names
	CreatedAt time.Time
	UpdatedAt time.Time
}
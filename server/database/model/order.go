package model // You can change this package name if needed

import (
	"time"
)

type CheckoutPageData struct {
	ISP       ISP
	ServicePlan ServicePlan
	PageTitle   string
	Zone      string
	DnsName   string
	Ip        string
	Mac       string
	DeviceId  string
}

// Order - Represents a user's order for a service plan
type Order struct {
	ID                int             `gorm:"primaryKey;autoIncrement;column:id"`
	OrderNumber       string          `gorm:"uniqueIndex;type:varchar(255);column:orderNumber;index:orderNumber"` // Unique order identifier
	Status            string          `gorm:"column:status;index:status"`                      // Status of the order
	Amount            int             `gorm:"column:amount"`                 // Total amount
	Username          string          `gorm:"column:username;"`
	Ip                string          `gorm:"column:ip;"`
	DnsName           string          `gorm:"column:dnsName;"`
	Mac               string          `gorm:"column:mac;"`
	Phone             string          `gorm:"column:phone;index:phone"`
	CheckoutRequestID string          `gorm:"column:CheckoutRequestID;index:checkoutRequestID"`
	MerchantRequestID string          `gorm:"column:MerchantRequestID;"`
	ResponseCode      string          `gorm:"column:ResponseCode;ResponseCode"`
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
package model // You can change this package name if needed

import (
	"time"

	"github.com/shopspring/decimal" // Required for the Decimal type
)

// Payment - Represents a payment made for an order
type Payment struct {
	ID                 int             `gorm:"primaryKey;autoIncrement;column:id"`
	Amount             decimal.Decimal `gorm:"type:decimal(20,2);column:Amount"`
	MpesaReceiptNumber *string         `gorm:"column:MpesaReceiptNumber;index:mpesaReceiptNumber"` // Nullable
	Phone              *string         `gorm:"column:Phone;index:phone"`                           // Nullable
	TransactionDate    string          `gorm:"column:TransactionDate;"`       // Nullable, keeping as string to match DESCRIBE
	MerchantRequestID  *string         `gorm:"column:MerchantRequestID;index:merchantRequestID"`   // Nullable
	CheckoutRequestID  string          `gorm:"column:CheckoutRequestID;index:checkoutRequestID;not null"`
	ResultCode         int             `gorm:"column:ResultCode;default:0;not null"`
	ResultDesc         string          `gorm:"type:text;column:ResultDesc;not null"`
	Username *string `gorm:"column:username;index:username"`

	// Foreign Key to Order (assuming the relationship)
	OrderID *int   `gorm:"column:orderId;index:orderId"`
	Order   *Order // Relationship to the Order model

	CreatedAt time.Time
	UpdatedAt time.Time
}



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
	TransactionDate    time.Time         `gorm:"column:TransactionDate;index:transactionDate"`       // Nullable, keeping as string to match DESCRIBE
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


// Transaction - Represents a record of a transaction within your system
// (e.g., applying a service plan after a payment)
type Transaction struct {
	ID          int             `gorm:"primaryKey;autoIncrement;column:id"`
	Type        string          `gorm:"column:type;index:type"`          // Type of transaction (e.g., "service_activation", "credit_topup")
	Amount      decimal.Decimal `gorm:"type:decimal(10,2);column:amount"`// Amount related to the transaction (can be 0)
	Status      string          `gorm:"column:status;index:status"`      // Status of the transaction

	// GORM handles @default(now()) and @updatedAt automatically for these field names
	// These map to `createdAt` and `updatedAt` columns
	CreatedAt time.Time
	UpdatedAt time.Time

	// Link to the User involved (non-nullable)
	UserID int `gorm:"column:userId;index:userId"` // Foreign key field for User
	User   User // Relationship field to User (non-pointer because non-nullable)

	// Optional link to the Payment that triggered this transaction (nullable)
	PaymentID *int `gorm:"column:paymentId;index:paymentId"` // Foreign key field for Payment
	Payment   *Payment // Relationship field to Payment

	// Optional link to the Order related to this transaction (nullable)
	OrderID *int `gorm:"column:orderId;index:orderId"` // Foreign key field for Order
	Order   *Order // Relationship field to Order

	Description *string `gorm:"column:description"` // Optional description of the transaction

}


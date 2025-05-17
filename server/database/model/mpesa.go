package model
import (
	"time"
   "github.com/shopspring/decimal"
)


type StkCallback struct {
	MerchantRequestID string           `json:"MerchantRequestID"`
	CheckoutRequestID string           `json:"CheckoutRequestID"`
	ResultCode        int              `json:"ResultCode"`
	ResultDesc        string           `json:"ResultDesc"`
	CallbackMetadata  *CallbackMetadata `json:"CallbackMetadata,omitempty"`
}

type CallbackMetadata struct {
	Item []CallbackItem `json:"Item"`
}

type CallbackItem struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value,omitempty"`
}

// MpesaCallbackPayload represents the extracted callback data
type MpesaCallbackPayload struct {
	MerchantRequestID  string          `json:"MerchantRequestID"`
	CheckoutRequestID  string          `json:"CheckoutRequestID"`
	ResultCode         int             `json:"ResultCode"`
	ResultDesc         string          `json:"ResultDesc"`
	Amount             decimal.Decimal `json:"Amount"`
	MpesaReceiptNumber string          `json:"MpesaReceiptNumber"`
	TransactionDate    time.Time        `json:"TransactionDate"`
	PhoneNumber        string          `json:"PhoneNumber"`
}
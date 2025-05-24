package service

import (
	"fmt"
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/server/database/model"
)



// SaveMpesaPayment saves payment information after Mpesa callback
func SaveMpesaPayment(payload *model.MpesaCallbackPayload) (map[string]interface{}, error) {
	db := gdatabase.GetDB(config.AppDB)

	// Define message based on ResultCode
    status := "error"
    message := "An unknown error occurred"

    // Map common M-Pesa result codes to user-friendly messages
    switch payload.ResultCode {
    case 0:
        status = "success"
        message = "Payment received successfully"
    case 1:
        message = "Insufficient balance"
    case 1001:
        message = "Payment is being processed"
    case 1002:
        message = "Payment request is being processed"
    case 1031:
        message = "Request cancelled by user"
    case 1032:
        message = "The request was canceled by the user"
    case 1037:
        message = "Your M-Pesa phone cannot be reached!"
    case 2001:
        message = "Wrong PIN provided"
    case 17:
        message = "User account does not exist"
    case 20:
        message = "User account is inactive"
    case 26:
        message = "Payment request timed out"
    default:
        message = fmt.Sprintf("Payment failed: %s", payload.ResultDesc)
    }

	var order model.Order
	if err := db.Where("CheckoutRequestID = ?", payload.CheckoutRequestID).First(&order).Error; err != nil {
		return nil, err
	}

	// Prepare payment data
	payment := &model.Payment{
		Amount:             payload.Amount,
		MpesaReceiptNumber: &payload.MpesaReceiptNumber,
		Phone:              &payload.PhoneNumber,
		TransactionDate:    payload.TransactionDate,
		MerchantRequestID:  &payload.MerchantRequestID,
		CheckoutRequestID:  payload.CheckoutRequestID,
		ResultCode:         payload.ResultCode,
		ResultDesc:         payload.ResultDesc,
		Username:           &order.Username,
		OrderID:            &order.ID,
	}

	// Begin a transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Save the payment
	if err := tx.Create(payment).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update order status based on payment result
	if payload.ResultCode == 0 {
		order.Status = "paid"
	} else {
		order.Status = "payment_failed"
	}

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":    status,
		"ip":        order.Ip,
		"paymentID": payment.ID,
		"message":  message,
	}, nil
}


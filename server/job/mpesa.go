package job

import (
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/server/database/model"
)

// SaveMpesaPayment saves payment information after Mpesa callback
func SaveMpesaPayment(payload *model.MpesaCallbackPayload) (map[string]interface{}, error) {
	db := gdatabase.GetDB(config.AppDB)
	
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
		"status":    "success",
		"ip":        order.Ip,
		"paymentID": payment.ID,
		"message":   "Payment saved to database",
	}, nil
}


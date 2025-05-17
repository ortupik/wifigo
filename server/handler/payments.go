package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/server/database/model"
	"gorm.io/gorm"
)

// GetPayment retrieves a payment based on input
func GetPayment(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var paymentInput model.Payment
	if err := c.BindJSON(&paymentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var payment model.Payment
	if err := tx.Where(&paymentInput).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}
	
	c.JSON(http.StatusOK, payment)
}

// GetPayments retrieves all payments with optional filtering
func GetPayments(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var paymentInput model.Payment
	if err := c.BindJSON(&paymentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var payments []model.Payment
	query := tx
	
	// Execute the query
	if err := query.Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payments"})
		return
	}
	
	c.JSON(http.StatusOK, payments)
}

// CreatePayment creates a new payment
func CreatePayment(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var paymentInput model.Payment
	if err := c.BindJSON(&paymentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if err := tx.Create(&paymentInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}
	
	c.JSON(http.StatusOK, paymentInput)
}

// UpdatePayment updates an existing payment
func UpdatePayment(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var paymentInput model.Payment
	if err := c.BindJSON(&paymentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if payment exists
	var existingPayment model.Payment
	if err := tx.Where("id = ?", paymentInput.ID).First(&existingPayment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}
	
	// Update the payment
	if err := tx.Model(&existingPayment).Updates(paymentInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}
	
	// Fetch the updated payment to return
	if err := tx.Where("id = ?", paymentInput.ID).First(&existingPayment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated payment"})
		return
	}
	
	c.JSON(http.StatusOK, existingPayment)
}

// DeletePayment removes a payment
func DeletePayment(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var paymentInput model.Payment
	if err := c.BindJSON(&paymentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if payment exists
	var existingPayment model.Payment
	if err := tx.Where("id = ?", paymentInput.ID).First(&existingPayment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}
	
	// Delete the payment
	if err := tx.Delete(&existingPayment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Payment deleted successfully"})
}


package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/server/database/model"
	"gorm.io/gorm"
)

// GetOrder retrieves an order based on input
func GetOrder(orderNumber string, c *gin.Context, tx *gorm.DB) {
	db := gdatabase.GetDB(config.AppDB)
	var order model.Order
	if err := db.Where("orderNumber = ?", orderNumber).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orderNumber":   order.OrderNumber,
		"status":        order.Status,
		"amount":        order.Amount,
		"responseCode":  order.ResponseCode,
		"resultDesc":    order.ResultDesc,
		"phone":         order.Phone,
		"username":      order.Username,
		"checkoutId":    order.CheckoutRequestID,
		"merchantId":    order.MerchantRequestID,
		"isHomeUser":    order.IsHomeUser,
	})
}

// GetOrders retrieves all orders with optional pagination
func GetOrders(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit
	
	var orders []model.Order
	var count int64
	
	// Get total count
	tx.Model(&model.Order{}).Count(&count)
	
	// Get paginated orders
	if err := tx.Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"meta": gin.H{
			"total": count,
			"page":  page,
			"limit": limit,
		},
	})
}

// CreateOrder creates a new order
func CreateOrder(c *gin.Context, tx *gorm.DB, orderInput model.Order) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}

	if err := tx.Create(&orderInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusOK, orderInput)
}


// UpdateOrder updates an existing order
func UpdateOrder(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	var orderInput model.Order
	if err := c.BindJSON(&orderInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if order exists
	var existingOrder model.Order
	if err := tx.Where("id = ?", orderInput.ID).First(&existingOrder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	
	// Update the order
	if err := tx.Model(&existingOrder).Updates(orderInput).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}
	
	// Fetch the updated order to return
	if err := tx.Where("id = ?", orderInput.ID).First(&existingOrder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated order"})
		return
	}
	
	c.JSON(http.StatusOK, existingOrder)
}

// DeleteOrder removes an order by ID
func DeleteOrder(c *gin.Context, tx *gorm.DB) {
	if tx == nil {
		tx = gdatabase.GetDB(config.AppDB)
	}
	
	id := c.Param("id")
	var order model.Order
	
	// Check if order exists
	if err := tx.Where("id = ?", id).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	
	// Delete the order
	if err := tx.Delete(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
}
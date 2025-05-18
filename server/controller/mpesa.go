package controller

import (
	"fmt"
	"net/http"
    "time"
	"math"
	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/server/dto"
	"github.com/ortupik/wifigo/server/handler"
	"github.com/ortupik/wifigo/server/database/model"
)

type MpesaController struct {
	mpesaHandler *handler.MpesaHandler
}

func NewMpesaController(mpesaHandler *handler.MpesaHandler) *MpesaController {
	return &MpesaController{mpesaHandler: mpesaHandler}
}

func (mc *MpesaController) ExpressStkHandler(c *gin.Context) {

	req := dto.STKPushRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	expirationStatus, err := handler.IsUserExpired(req.Username)
	if(err == nil){
		if expirationStatus == "NOT_EXPIRED" {
			c.JSON(http.StatusConflict, gin.H{"error": "Active subscription already exists"})
			return
		}
	}

	plan, err := mc.mpesaHandler.GetServicePlan(req.PlanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid plan"})
		return
	}

    amount := plan.Price 

	if req.DeviceCount > 2 {
		amount = int(math.Round(float64(amount) * 0.7)) // Apply 30% discount
	}

	res, err := mc.mpesaHandler.SendStkPush(req.Phone, fmt.Sprintf("%d", amount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STK Push failed: " + err.Error()})
		return
	}

	if errCode, exists := res["errorCode"]; exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"errorCode":    fmt.Sprint(errCode),
			"errorMessage": fmt.Sprint(res["errorMessage"]),
			"requestId":    fmt.Sprint(res["requestId"]),
		})
		return
	}

	username := req.Phone + "@Tecsurf"
	isHomeUser := false

	order := model.Order{
		OrderNumber:       fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		Status:            "PENDING",
		Amount:            amount,
		Username:          username,
		Ip:                req.Ip,
		Mac:               req.Mac,
		Phone:             req.Phone,
		ISP:               req.IspID,
		Zone:              req.Zone,
		DeviceID:          req.DeviceID,
		IsHomeUser:        isHomeUser,
		Devices:           req.DeviceCount,
		ServicePlanID:     plan.ID,
		ResultDesc:        fmt.Sprint(res["ResponseDescription"]),
		CheckoutRequestID: fmt.Sprint(res["CheckoutRequestID"]),
		MerchantRequestID: fmt.Sprint(res["MerchantRequestID"]),
		ResponseCode:      fmt.Sprint(res["ResponseCode"]),
	}
	
	handler.CreateOrder(c, nil, order)

}

// GetTransactionStatus handles the transaction status request
func (mc *MpesaController) GetTransactionStatus(c *gin.Context) {
	orderNumber := c.Query("orderNumber")
	if orderNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderNumber is required"})
		return
	}

	handler.GetOrder(orderNumber, c, nil)
}

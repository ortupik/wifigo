package controller

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/server/database/model"
	"github.com/ortupik/wifigo/server/dto"
	"github.com/ortupik/wifigo/server/handler"
)


type MpesaController struct {
	MpesaStkHandler *handler.MpesaStkHandler
}

func NewMpesaController() *MpesaController {
	mpesaStkhandler, err := handler.NewMpesaStkHandler()
	if(err != nil) {
		fmt.Println(err)
		return nil
	}

	return &MpesaController{
		MpesaStkHandler : mpesaStkhandler,
	}
}

func (mc *MpesaController) ExpressStkHandler(c *gin.Context) {

	//do proper validations of the request
	req := dto.STKPushRequest
	username := req.Username

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "err": err.Error()})
		return
	}

	//fetch (fetch Realm based on domain from badger key value)
	realm := "Tecsurf"
	if(username == ""){
        username = req.Phone + "@" + realm
	}

	//check if user is Home User from req.IsHomeUser(true ? then username changes not to use phone)
	isHomeUser := false

	expirationStatus, err := handler.IsUserExpired(username)

	if err == nil {
		if expirationStatus == "NOT_EXPIRED" {
			c.JSON(http.StatusConflict, gin.H{"error": "Active subscription already exists"})
			return
		}
	}

	plan, err := mc.MpesaStkHandler.GetServicePlan(req.PlanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid plan"})
		return
	}

	amount := plan.Price

	if req.DeviceCount > 1 {
		amount = int(math.Round(float64(amount) * 0.7)) // Apply 30% discount
		fmt.Printf("amount %d", amount)
	}

	res, err := mc.MpesaStkHandler.SendStkPush(req.Phone, fmt.Sprintf("%d", amount))
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

package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/server/database/model" // Make sure this is the correct path
	"github.com/ortupik/wifigo/server/handler"  // Import your handler
)

// CheckoutController handles the checkout process.
func CheckoutController(c *gin.Context) {
	// Check for required query parameters and their validity.
	ispIDStr := c.Query("isp_id")
	if ispIDStr == "" {
		renderErrorPage(c, "Missing required parameter: isp_id", "Missing Parameter", http.StatusBadRequest)
		return
	}

	planIDStr := c.Query("plan_id")
	if planIDStr == "" {
		renderErrorPage(c, "Missing required parameter: plan_id", "Missing Parameter", http.StatusBadRequest)
		return
	}

	zoneStr := c.Query("zone")
	if zoneStr == "" {
		renderErrorPage(c, "Missing required parameter: zone", "Missing Parameter", http.StatusBadRequest)
		return
	}

	ip := c.Query("ip")
	if ip == "" {
		renderErrorPage(c, "Missing required parameter: ip", "Missing Parameter", http.StatusBadRequest)
		return
	}

	mac := c.Query("mac")
	
	deviceID := c.Query("device_id")
	if deviceID == "" {
		renderErrorPage(c, "Missing required parameter: device_id", "Missing Parameter", http.StatusBadRequest)
		return
	}

	// Convert string IDs to int64.
	ispID, err := strconv.ParseInt(ispIDStr, 10, 64)
	if err != nil {
		renderErrorPage(c, "Invalid ISP ID format", "Invalid Input", http.StatusBadRequest)
		return
	}

	planID, err := strconv.ParseInt(planIDStr, 10, 64)
	if err != nil {
		renderErrorPage(c, "Invalid Plan ID format", "Invalid Input", http.StatusBadRequest)
		return
	}

	// Fetch ISP and Plan using the handler.
	ispData, err := handler.GetISPAndPlan(c, ispID, planID) // Use the handler
	if err != nil {
		// The handler already writes the error to the context, so we just return.
		return
	}

	// Prepare data for the checkout page.
	pageData := model.CheckoutPageData{
		ISP:       ispData.ISP,       // Access the ISP from the returned struct
		ServicePlan: ispData.Plan,    // Access the Plan
		PageTitle:   fmt.Sprintf("%s Payment - %s", ispData.ISP.Name, ispData.Plan.Name),
		Zone:      zoneStr,
		DnsName:   ispData.ISP.DnsName, // Access DnsName from the ISP
		Ip:        ip,
		Mac:       mac,
		DeviceId: deviceID,
	}

	// Render the checkout page with the data.
	c.HTML(http.StatusOK, "checkout.html", pageData)
}

func ConfirmController(c *gin.Context) {
	c.HTML(http.StatusOK, "confirm.html", nil)
}

// renderErrorPage is a helper function to render the error page.
func renderErrorPage(c *gin.Context, message, title string, status int) {
	c.HTML(status, "error.html", gin.H{
		"Message": message,
		"Title":   title,
	})
	c.AbortWithStatus(status) // Important:  Abort the handler chain.
}
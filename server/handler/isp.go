package handler

import (
	"net/http"
    "gorm.io/gorm"

	"github.com/gin-gonic/gin"

	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/server/database/model"
	
)

type ISPAndPlanData struct {
	ISP  model.ISP
	Plan model.ServicePlan
}

// GetISPAndPlan retrieves a single ISP and a specific service plan.
func GetISPAndPlan(c *gin.Context,ispID, planID int64) (ISPAndPlanData, error) {
	db := gdatabase.GetDB(config.AppDB)
	var isp model.ISP
	var servicePlan model.ServicePlan

	// Fetch the ISP.  We don't preload here, because we only want one plan.
	if err := db.Where("id = ?", ispID).First(&isp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "ISP not found", "status": http.StatusNotFound})
			return ISPAndPlanData{}, err // Return the error
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "status": http.StatusInternalServerError})
			return ISPAndPlanData{}, err
		}

	}

	// Fetch the specific ServicePlan.
	if err := db.Where("id = ? AND isp_id = ?", planID, ispID).First(&servicePlan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Service Plan not found for this ISP", "status": http.StatusNotFound})
			return ISPAndPlanData{}, err
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "status": http.StatusInternalServerError})
			return ISPAndPlanData{}, err
		}
	}

	// Return the data.
	return ISPAndPlanData{ISP: isp, Plan: servicePlan}, nil
}
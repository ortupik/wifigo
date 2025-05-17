package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	grenderer "github.com/ortupik/wifigo/lib/renderer"
)

// APIStatus - check status of the API
func APIStatus(c *gin.Context) {
	grenderer.Render(c, gin.H{"message": "live"}, http.StatusOK)
}

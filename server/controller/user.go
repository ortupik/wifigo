// Package controller contains all the controllers
// of the application
package controller

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"

	grenderer "github.com/ortupik/wifigo/lib/renderer"

	"github.com/ortupik/wifigo/server/database/model"
	"github.com/ortupik/wifigo/server/handler"
)

// GetUsers - GET /users
func GetUsers(c *gin.Context) {
	resp, statusCode := handler.GetUsers()

	grenderer.Render(c, resp, statusCode)
}

// GetUser - GET /users/:id
func GetUser(c *gin.Context) {
	id := strings.TrimSpace(c.Params.ByName("id"))

	resp, statusCode := handler.GetUser(id)

	if reflect.TypeOf(resp.Message).Kind() == reflect.String {
		grenderer.Render(c, resp, statusCode)
		return
	}

	grenderer.Render(c, resp.Message, statusCode)
}

// CreateUser - POST /users
func CreateUser(c *gin.Context) {
	userIDAuth := c.GetUint64("authID")
	user := model.User{}

	// bind JSON
	if err := c.ShouldBindJSON(&user); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	resp, statusCode := handler.CreateUser(userIDAuth, user)

	if reflect.TypeOf(resp.Message).Kind() == reflect.String {
		grenderer.Render(c, resp, statusCode)
		return
	}

	grenderer.Render(c, resp.Message, statusCode)
}

// UpdateUser - PUT /users
func UpdateUser(c *gin.Context) {
	userIDAuth := c.GetUint64("authID")
	user := model.User{}

	// bind JSON
	if err := c.ShouldBindJSON(&user); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	resp, statusCode := handler.UpdateUser(userIDAuth, user)

	if reflect.TypeOf(resp.Message).Kind() == reflect.String {
		grenderer.Render(c, resp, statusCode)
		return
	}

	grenderer.Render(c, resp.Message, statusCode)
}

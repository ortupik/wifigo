package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	grenderer "github.com/ortupik/wifigo/lib/renderer"
	radiusHandler "github.com/ortupik/wifigo/server/handler"
	dto "github.com/ortupik/wifigo/server/dto"

)

var input dto.HotspotUserInput


// --- Controller Functions ---

// CreateHotspotUser - POST /hotspot/users
// Creates a new hotspot user by adding entries in radcheck, radreply, and radusergroup.
func CreateHotspotUser(c *gin.Context) {
	// Bind JSON request body to input struct
	if err := c.ShouldBindJSON(&input); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	// Call handler to create the user in the database
	// The handler will need to process the input struct and create/update
	// entries in radcheck, radreply, and radusergroup.
	resp, statusCode := radiusHandler.CreateHotspotUser(input)

	grenderer.Render(c, resp, statusCode)
}

// GetHotspotUser - GET /hotspot/users/:username
// Retrieves all radcheck, radreply, and radusergroup entries for a given username.
func GetHotspotUser(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	// Call handler to get user details from the database
	// The handler will query radcheck, radreply, and radusergroup tables for the username.
	resp, statusCode := radiusHandler.GetHotspotUser(username)

	// Assuming the handler returns a structured response (e.g., with user details)
	grenderer.Render(c, resp, statusCode)
}

// UpdateHotspotUser - PUT /hotspot/users/:username
// Updates the configuration for an existing hotspot user.
// This might involve adding, modifying, or deleting entries in radcheck, radreply, and radusergroup.
func UpdateHotspotUser(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	// Bind JSON request body to input struct
	if err := c.ShouldBindJSON(&input); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	// Ensure the username in the path matches the username in the body, if present
	if input.Username != "" && input.Username != username {
		grenderer.Render(c, gin.H{"message": "Username mismatch in path and body"}, http.StatusBadRequest)
		return
	}
	input.Username = username // Use the username from the path

	// Call handler to update the user configuration
	// The handler will compare the desired state (input) with the current state in the DB
	// and perform necessary create/update/delete operations across the tables.
	resp, statusCode := radiusHandler.UpdateHotspotUser(input)

	grenderer.Render(c, resp, statusCode)
}

// DeleteHotspotUser - DELETE /hotspot/users/:username
// Deletes a hotspot user by removing all associated entries from radcheck, radreply, and radusergroup.
func DeleteHotspotUser(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	// Call handler to delete the user from the database
	// The handler will delete all entries for the username from radcheck, radreply, and radusergroup.
	resp, statusCode := radiusHandler.DeleteHotspotUser(username)

	grenderer.Render(c, resp, statusCode)
}

// --- Granular Attribute/Group Management (Optional but recommended) ---
// The functions below provide more granular control over individual attributes and groups.
// You can choose to implement these in addition to or instead of the comprehensive UpdateHotspotUser.

// AddOrUpdateRadCheckAttribute - POST /hotspot/users/:username/check
func AddOrUpdateRadCheckAttribute(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	var input dto.RadCheckInput
	if err := c.ShouldBindJSON(&input); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.AddOrUpdateRadCheckAttribute(username, input)
	grenderer.Render(c, resp, statusCode)
}

// DeleteRadCheckAttribute - DELETE /hotspot/users/:username/check/:attribute
func DeleteRadCheckAttribute(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	attribute := strings.TrimSpace(c.Params.ByName("attribute"))
	if username == "" || attribute == "" {
		grenderer.Render(c, gin.H{"message": "Username and attribute are required"}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.DeleteRadCheckAttribute(username, attribute)
	grenderer.Render(c, resp, statusCode)
}

// AddOrUpdateRadReplyAttribute - POST /hotspot/users/:username/reply
func AddOrUpdateRadReplyAttribute(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	var input dto.RadReplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.AddOrUpdateRadReplyAttribute(username, input)
	grenderer.Render(c, resp, statusCode)
}

// DeleteRadReplyAttribute - DELETE /hotspot/users/:username/reply/:attribute
func DeleteRadReplyAttribute(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	attribute := strings.TrimSpace(c.Params.ByName("attribute"))
	if username == "" || attribute == "" {
		grenderer.Render(c, gin.H{"message": "Username and attribute are required"}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.DeleteRadReplyAttribute(username, attribute)
	grenderer.Render(c, resp, statusCode)
}

// AddRadUserGroup - POST /hotspot/users/:username/group
func AddRadUserGroup(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	if username == "" {
		grenderer.Render(c, gin.H{"message": "Username is required"}, http.StatusBadRequest)
		return
	}

	var input dto.RadUserGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		grenderer.Render(c, gin.H{"message": err.Error()}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.AddRadUserGroup(username, input)
	grenderer.Render(c, resp, statusCode)
}

// DeleteRadUserGroup - DELETE /hotspot/users/:username/group/:groupname
func DeleteRadUserGroup(c *gin.Context) {
	username := strings.TrimSpace(c.Params.ByName("username"))
	groupname := strings.TrimSpace(c.Params.ByName("groupname"))
	if username == "" || groupname == "" {
		grenderer.Render(c, gin.H{"message": "Username and groupname are required"}, http.StatusBadRequest)
		return
	}

	resp, statusCode := radiusHandler.DeleteRadUserGroup(username, groupname)
	grenderer.Render(c, resp, statusCode)
}
package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	dto "github.com/ortupik/wifigo/server/dto"
)

func ManageHotspotUser(req dto.HotspotSubscriptionRequest, isSubscribing bool) (gin.H, int) {

	username := req.Username
	isHomeUser := req.IsHomeUser
	group := req.ServiceName
	duration := req.Duration
	devices := req.Devices
	password := req.Password

	if isHomeUser {
		userAttr, err := getRadCheckAttributes(username)
		if err == nil && userAttr != nil {
			for _, userDetails := range userAttr {
				if userDetails.Attribute == "Cleartext-Password" {
					password = &userDetails.Value
					break
				}
			}
		} else {
			if isSubscribing {
				return gin.H{"error": fmt.Sprintf("User %s does not exist!", username)}, http.StatusNotFound
			}
		}
	}

	hotspotUser := dto.HotspotUserInput{
		Username: username,
		Password: password,
		CheckAttributes: []dto.RadCheckInput{
			{
				Attribute: "Expiration",
				Op:        ":=",
				Value:     time.Now().Add(time.Duration(duration) * time.Second).Format("Jan 2 2006 15:04:05"),
			},
			{
				Attribute: "Simultaneous-Use",
				Op:        ":=",
				Value:     fmt.Sprint(devices),
			},
		},
		Groups: []dto.RadUserGroupInput{
			{
				Groupname: group,
				Priority:  new(int),
			},
		},
	}
	*hotspotUser.Groups[0].Priority = 1

	userStatus, err := IsUserExpired(username)

	if err == nil && userStatus == "NOT_EXPIRED" && isSubscribing {
		return gin.H{"error": fmt.Sprintf("User %s already has an active subscription", username)}, http.StatusConflict
	} else if err == nil && userStatus == "NO_EXIST" {
		resp, statusCode := CreateHotspotUser(hotspotUser)
		if statusCode != http.StatusCreated {
			return gin.H{"error": fmt.Sprintf("Failed to create user: %v", resp["error"])}, statusCode
		}
	} else if err == nil && userStatus == "EXPIRED" {
		resp, statusCode := UpdateHotspotUser(hotspotUser)
		if statusCode != http.StatusOK {
			return gin.H{"error": fmt.Sprintf("Failed to update user: %v", resp["error"])}, statusCode
		}
	} else {
		return gin.H{"error": err}, http.StatusInternalServerError
	}

	return gin.H{
		"message":  "Subscription made successfully",
		"username": username,
		"password": password,
		"group":    group,
	}, http.StatusOK
}

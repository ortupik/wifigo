package dto

import (
   radiusmodel "github.com/ortupik/wifigo/server/database/model"
)

type HotspotSubscriptionRequest struct {
	Phone       string
	Username    string
	Password    *string
	IsHomeUser  bool
	ISP         string
	ServiceName string
	Duration    int
	Devices     int
}
// HotspotUser represents the complete user configuration
type HotspotUser struct {
	Username        string         `json:"username"`
	CheckAttributes []radiusmodel.RadCheck     `json:"checkAttributes"`
	ReplyAttributes []radiusmodel.RadReply     `json:"replyAttributes"`
	Groups          []radiusmodel.RadUserGroup `json:"groups"`
}

// HotspotUserInput is the structure for creating a new hotspot user
type HotspotUserInput struct {
	Username        string              `json:"username" binding:"required"`
	Password        *string             `json:"password"` // Optional Cleartext-Password
	CheckAttributes []RadCheckInput     `json:"checkAttributes"`
	ReplyAttributes []RadReplyInput     `json:"replyAttributes"`
	Groups          []RadUserGroupInput `json:"groups"`
}

// RadCheckInput is the structure for creating/updating a RadCheck attribute
type RadCheckInput struct {
	Attribute string `json:"attribute" binding:"required"`
	Op        string `json:"op"` // Defaults to ":=" if empty
	Value     string `json:"value" binding:"required"`
}

// RadReplyInput is the structure for creating/updating a RadReply attribute
type RadReplyInput struct {
	Attribute string `json:"attribute" binding:"required"`
	Op        string `json:"op"` // Defaults to ":=" if empty
	Value     string `json:"value" binding:"required"`
}

// RadUserGroupInput is the structure for adding a user to a group
type RadUserGroupInput struct {
	Groupname string `json:"groupname" binding:"required"`
	Priority  *int   `json:"priority"` // Defaults to 1 if nil
}

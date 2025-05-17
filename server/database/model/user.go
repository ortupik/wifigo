package model

import (
	"time"

	"gorm.io/gorm"
)

// User model - `users` table
type User struct {
	UserID    uint64         `gorm:"primaryKey" json:"userID,omitempty"`
	CreatedAt time.Time      `json:"createdAt,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	FirstName string         `json:"firstName,omitempty"`
	LastName  string         `json:"lastName,omitempty"`
	IDAuth    uint64         `json:"-"`
}

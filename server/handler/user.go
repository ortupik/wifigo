// Package handler of the server application
package handler

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	gmodel "github.com/ortupik/wifigo/database/model"

	"github.com/ortupik/wifigo/server/database/model"
)

// GetUsers handles jobs for controller.GetUsers
func GetUsers() (httpResponse gmodel.HTTPResponse, httpStatusCode int) {
	db := gdatabase.GetDB(config.AppDB)
	users := []model.User{}

	if err := db.Find(&users).Error; err != nil {
		log.WithError(err).Error("error code: 1101")
		httpResponse.Message = "internal server error"
		httpStatusCode = http.StatusInternalServerError
		return
	}

	if len(users) == 0 {
		httpResponse.Message = "no user found"
		httpStatusCode = http.StatusNotFound
		return
	}

	httpResponse.Message = users
	httpStatusCode = http.StatusOK
	return
}

// GetUser handles jobs for controller.GetUser
func GetUser(id string) (httpResponse gmodel.HTTPResponse, httpStatusCode int) {
	db := gdatabase.GetDB(config.AppDB)
	user := model.User{}

	if err := db.Where("user_id = ?", id).First(&user).Error; err != nil {
		httpResponse.Message = "user not found"
		httpStatusCode = http.StatusNotFound
		return
	}

	httpResponse.Message = user
	httpStatusCode = http.StatusOK
	return
}

// CreateUser handles jobs for controller.CreateUser
func CreateUser(userIDAuth uint64, user model.User) (httpResponse gmodel.HTTPResponse, httpStatusCode int) {
	db := gdatabase.GetDB(config.AppDB)
	userFinal := model.User{}

	// does the user have an existing profile
	if err := db.Where("id_auth = ?", userIDAuth).First(&userFinal).Error; err == nil {
		httpResponse.Message = "user profile found, no need to create a new one"
		httpStatusCode = http.StatusForbidden
		return
	}

	// user must not be able to manipulate all fields
	userFinal.FirstName = user.FirstName
	userFinal.LastName = user.LastName
	userFinal.IDAuth = userIDAuth

	tx := db.Begin()
	if err := tx.Create(&userFinal).Error; err != nil {
		tx.Rollback()
		log.WithError(err).Error("error code: 1111")
		httpResponse.Message = "internal server error"
		httpStatusCode = http.StatusInternalServerError
		return
	}
	tx.Commit()

	httpResponse.Message = userFinal
	httpStatusCode = http.StatusCreated
	return
}

// UpdateUser handles jobs for controller.UpdateUser
func UpdateUser(userIDAuth uint64, user model.User) (httpResponse gmodel.HTTPResponse, httpStatusCode int) {
	db := gdatabase.GetDB(config.AppDB)
	userFinal := model.User{}

	// does the user have an existing profile
	if err := db.Where("id_auth = ?", userIDAuth).First(&userFinal).Error; err != nil {
		httpResponse.Message = "no user profile found"
		httpStatusCode = http.StatusNotFound
		return
	}

	// user must not be able to manipulate all fields
	userFinal.UpdatedAt = time.Now()
	userFinal.FirstName = user.FirstName
	userFinal.LastName = user.LastName

	tx := db.Begin()
	if err := tx.Save(&userFinal).Error; err != nil {
		tx.Rollback()
		log.WithError(err).Error("error code: 1121")
		httpResponse.Message = "internal server error"
		httpStatusCode = http.StatusInternalServerError
		return
	}
	tx.Commit()

	httpResponse.Message = userFinal
	httpStatusCode = http.StatusOK
	return
}

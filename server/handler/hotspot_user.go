package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"gorm.io/gorm"
	dto "github.com/ortupik/wifigo/server/dto"
	radiusmodel "github.com/ortupik/wifigo/server/database/model"

)

// Default operator for RADIUS attributes
const defaultOp = ":="

// Common attribute names
const (
	AttrCleartextPassword = "Cleartext-Password"
	AttrAuthType            = "Auth-Type"
	AttrExpirationDate    = "Expiration"
	AttrSessionTimeout    = "Session-Timeout"
	AttrIdleTimeout       = "Idle-Timeout"
	AttrMaxAllSession     = "Max-All-Session"
	AttrSimultaneousUse   = "Simultaneous-Use"
)


// CreateHotspotUser creates a new hotspot user
func CreateHotspotUser(input dto.HotspotUserInput) (gin.H, int) {
	// Start a transaction
	db := gdatabase.GetDB(config.RadiusDB)
	tx := db.Begin()
	if tx.Error != nil {
		return gin.H{"error": "Failed to start transaction: " + tx.Error.Error()}, http.StatusInternalServerError
	}

	// Ensure we either commit or rollback the transaction
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r) // Re-panic after rollback
		}
	}()

	// Check if user already exists
	exists, err := userExists(tx, input.Username)
	if err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if exists {
		_ = tx.Rollback()
		return gin.H{"error": "User already exists"}, http.StatusConflict
	}

	// Add password if provided
	if input.Password != nil && *input.Password != "" {
		// Add Cleartext-Password in radcheck
		err = insertRadCheck(tx, input.Username, AttrCleartextPassword, defaultOp, *input.Password)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to add password: " + err.Error()}, http.StatusInternalServerError
		}
	}else{
		err = insertRadCheck(tx, input.Username, AttrAuthType, defaultOp, "Accept")
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to add accept: " + err.Error()}, http.StatusInternalServerError
		}
	}

	// Add check attributes
	for _, attr := range input.CheckAttributes {
		op := attr.Op
		if op == "" {
			op = defaultOp
		}

		err = insertRadCheck(tx, input.Username, attr.Attribute, op, attr.Value)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{
				"error": fmt.Sprintf("Failed to add check attribute '%s': %s", attr.Attribute, err.Error()),
			}, http.StatusInternalServerError
		}
	}

	// Add reply attributes
	for _, attr := range input.ReplyAttributes {
		op := attr.Op
		if op == "" {
			op = defaultOp
		}

		err = insertRadReply(tx, input.Username, attr.Attribute, op, attr.Value)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{
				"error": fmt.Sprintf("Failed to add reply attribute '%s': %s", attr.Attribute, err.Error()),
			}, http.StatusInternalServerError
		}
	}

	// Add group memberships
	for _, group := range input.Groups {
		priority := 1
		if group.Priority != nil {
			priority = *group.Priority
		}

		err = insertRadUserGroup(tx, input.Username, group.Groupname, priority)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{
				"error": fmt.Sprintf("Failed to add user to group '%s': %s", group.Groupname, err.Error()),
			}, http.StatusInternalServerError
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to commit transaction: " + err.Error()}, http.StatusInternalServerError
	}
	

	// Return success response
	return gin.H{
		"message":  "Hotspot user created successfully",
		"username": input.Username,
	}, http.StatusCreated
}

// GetHotspotUser retrieves all configuration for a user
func GetHotspotUser(username string) (gin.H, int) {
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Get check attributes
	checkAttrs, err := getRadCheckAttributes(username)
	if err != nil {
		return gin.H{"error": "Failed to get check attributes: " + err.Error()}, http.StatusInternalServerError
	}

	// Get reply attributes
	replyAttrs, err := getRadReplyAttributes(username)
	if err != nil {
		return gin.H{"error": "Failed to get reply attributes: " + err.Error()}, http.StatusInternalServerError
	}

	// Get group memberships
	groups, err := getRadUserGroups(username)
	if err != nil {
		return gin.H{"error": "Failed to get group memberships: " + err.Error()}, http.StatusInternalServerError
	}

	// Build user configuration
	user := dto.HotspotUser{
		Username:        username,
		CheckAttributes: checkAttrs,
		ReplyAttributes: replyAttrs,
		Groups:          groups,
	}

	return gin.H{
		"user": user,
	}, http.StatusOK
}

// UpdateHotspotUser updates a user's configuration
func UpdateHotspotUser(input dto.HotspotUserInput) (gin.H, int) {
	db := gdatabase.GetDB(config.RadiusDB)
	username := input.Username
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Start a transaction
	tx := db.Begin()
	if err != nil {
		return gin.H{"error": "Failed to start transaction: " + err.Error()}, http.StatusInternalServerError
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Update password if provided (nil means no change, empty string means remove password)
	if input.Password != nil {
		// Delete existing password
		err = deleteRadCheckAttributeTx(tx, username, AttrCleartextPassword)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to update password: " + err.Error()}, http.StatusInternalServerError
		}

		// Add new password if not empty
		if *input.Password != "" {
			err = insertRadCheck(tx, username, AttrCleartextPassword, defaultOp, *input.Password)
			if err != nil {
				_ = tx.Rollback()
				return gin.H{"error": "Failed to update password: " + err.Error()}, http.StatusInternalServerError
			}
		}
	}else{
		err = insertRadCheck(tx, username, AttrAuthType, defaultOp, "Accept")
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to add accept: " + err.Error()}, http.StatusInternalServerError
		}
	}

	// Update check attributes if provided (nil means no change, empty slice means remove all non-password checks)
	if input.CheckAttributes != nil {
		// Get current attributes to compare
		currentAttrs, err := getRadCheckAttributesTx(tx, username)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to get current check attributes: " + err.Error()}, http.StatusInternalServerError
		}

		// Create a map for easier lookup and tracking processed attributes
		currentAttrMap := make(map[string] radiusmodel.RadCheck)
		for _, attr := range currentAttrs {
			if attr.Attribute != AttrCleartextPassword { // Skip password as it's handled separately
				currentAttrMap[attr.Attribute] = attr
			}
		}

		// Process input attributes
		processedAttrs := make(map[string]bool)
		for _, attr := range input.CheckAttributes {
			op := attr.Op
			if op == "" {
				op = defaultOp
			}

			// Check if attribute exists and needs update
			if existing, exists := currentAttrMap[attr.Attribute]; exists {
				processedAttrs[attr.Attribute] = true
				if existing.Op != op || existing.Value != attr.Value {
					// Update existing attribute
					err = updateRadCheckAttributeTx(tx, username, attr.Attribute, op, attr.Value)
					if err != nil {
						_ = tx.Rollback()
						return gin.H{
							"error": fmt.Sprintf("Failed to update check attribute '%s': %s", attr.Attribute, err.Error()),
						}, http.StatusInternalServerError
					}
				}
			} else {
				// Insert new attribute
				err = insertRadCheck(tx, username, attr.Attribute, op, attr.Value)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to add check attribute '%s': %s", attr.Attribute, err.Error()),
					}, http.StatusInternalServerError
				}
				processedAttrs[attr.Attribute] = true
			}
		}

		// Delete attributes not in the input (and not password)
		for attrName, attr := range currentAttrMap {
			if attr.Attribute != AttrCleartextPassword && !processedAttrs[attrName] {
				err = deleteRadCheckAttributeTx(tx, username, attrName)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to delete check attribute '%s': %s", attrName, err.Error()),
					}, http.StatusInternalServerError
				}
			}
		}
	}

	// Update reply attributes if provided (nil means no change, empty slice means remove all)
	if input.ReplyAttributes != nil {
		// Get current attributes to compare
		currentAttrs, err := getRadReplyAttributesTx(tx, username)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to get current reply attributes: " + err.Error()}, http.StatusInternalServerError
		}

		// Create a map for easier lookup
		currentAttrMap := make(map[string] radiusmodel.RadReply)
		for _, attr := range currentAttrs {
			currentAttrMap[attr.Attribute] = attr
		}

		// Process input attributes
		processedAttrs := make(map[string]bool)
		for _, attr := range input.ReplyAttributes {
			op := attr.Op
			if op == "" {
				op = defaultOp
			}

			// Check if attribute exists and needs update
			if existing, exists := currentAttrMap[attr.Attribute]; exists {
				processedAttrs[attr.Attribute] = true
				if existing.Op != op || existing.Value != attr.Value {
					// Update existing attribute
					err = updateRadReplyAttributeTx(tx, username, attr.Attribute, op, attr.Value)
					if err != nil {
						_ = tx.Rollback()
						return gin.H{
							"error": fmt.Sprintf("Failed to update reply attribute '%s': %s", attr.Attribute, err.Error()),
						}, http.StatusInternalServerError
					}
				}
			} else {
				// Insert new attribute
				err = insertRadReply(tx, username, attr.Attribute, op, attr.Value)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to add reply attribute '%s': %s", attr.Attribute, err.Error()),
					}, http.StatusInternalServerError
				}
				processedAttrs[attr.Attribute] = true
			}
		}

		// Delete attributes not in the input
		for attrName := range currentAttrMap {
			if !processedAttrs[attrName] {
				err = deleteRadReplyAttributeTx(tx, username, attrName)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to delete reply attribute '%s': %s", attrName, err.Error()),
					}, http.StatusInternalServerError
				}
			}
		}
	}

	// Update group memberships if provided (nil means no change, empty slice means remove all)
	if input.Groups != nil {
		// Get current groups to compare
		currentGroups, err := getRadUserGroupsTx(tx, username)
		if err != nil {
			_ = tx.Rollback()
			return gin.H{"error": "Failed to get current groups: " + err.Error()}, http.StatusInternalServerError
		}

		// Create a map for easier lookup
		currentGroupMap := make(map[string] radiusmodel.RadUserGroup)
		for _, group := range currentGroups {
			currentGroupMap[group.Groupname] = group
		}

		// Process input groups
		processedGroups := make(map[string]bool)
		for _, group := range input.Groups {
			priority := 1
			if group.Priority != nil {
				priority = *group.Priority
			}

			// Check if group exists and needs update
			if existing, exists := currentGroupMap[group.Groupname]; exists {
				processedGroups[group.Groupname] = true
				if existing.Priority != priority {
					// Update priority
					err = updateRadUserGroupTx(tx, username, group.Groupname, priority)
					if err != nil {
						_ = tx.Rollback()
						return gin.H{
							"error": fmt.Sprintf("Failed to update group '%s': %s", group.Groupname, err.Error()),
						}, http.StatusInternalServerError
					}
				}
			} else {
				// Add new group
				err = insertRadUserGroup(tx, username, group.Groupname, priority)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to add user to group '%s': %s", group.Groupname, err.Error()),
					}, http.StatusInternalServerError
				}
				processedGroups[group.Groupname] = true
			}
		}

		// Delete groups not in the input
		for groupName := range currentGroupMap {
			if !processedGroups[groupName] {
				err = deleteRadUserGroupTx(tx, username, groupName)
				if err != nil {
					_ = tx.Rollback()
					return gin.H{
						"error": fmt.Sprintf("Failed to remove user from group '%s': %s", groupName, err.Error()),
					}, http.StatusInternalServerError
				}
			}
		}
	}

	// Commit the transaction
	if err = tx.Commit().Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to commit transaction: " + err.Error()}, http.StatusInternalServerError
	}

	return gin.H{
		"message":  "Hotspot user updated successfully",
		"username": username,
	}, http.StatusOK
}

// DeleteHotspotUser removes a user and all associated configuration
func DeleteHotspotUser(username string) (gin.H, int) {
	db := gdatabase.GetDB(config.RadiusDB)
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Start a transaction
	tx := db.Begin()
	if tx.Error != nil {
		return gin.H{"error": "Failed to start transaction: " + tx.Error.Error()}, http.StatusInternalServerError
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Delete from radcheck
	if err := tx.Where("username = ?", username).Delete(& radiusmodel.RadCheck{}).Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to delete from radcheck: " + err.Error()}, http.StatusInternalServerError
	}

	// Delete from radreply
	if err := tx.Where("username = ?", username).Delete(&radiusmodel.RadReply{}).Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to delete from radreply: " + err.Error()}, http.StatusInternalServerError
	}

	// Delete from radusergroup
	if err := tx.Where("username = ?", username).Delete(&radiusmodel.RadUserGroup{}).Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to delete from radusergroup: " + err.Error()}, http.StatusInternalServerError
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		_ = tx.Rollback()
		return gin.H{"error": "Failed to commit transaction: " + err.Error()}, http.StatusInternalServerError
	}

	return gin.H{
		"message":  "Hotspot user deleted successfully",
		"username": username,
	}, http.StatusOK
}


// AddOrUpdateRadCheckAttribute adds or updates a check attribute
func AddOrUpdateRadCheckAttribute(username string, input dto.RadCheckInput) (gin.H, int) {
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	op := input.Op
	if op == "" {
		op = defaultOp
	}

	// Check if attribute exists
	exists, err = checkAttributeExists(username, input.Attribute)
	if err != nil {
		return gin.H{"error": "Failed to check if attribute exists: " + err.Error()}, http.StatusInternalServerError
	}

	if exists {
		// Update existing attribute
		err = updateRadCheckAttribute(username, input.Attribute, op, input.Value)
		if err != nil {
			return gin.H{"error": "Failed to update attribute: " + err.Error()}, http.StatusInternalServerError
		}
	} else {
		// Insert new attribute
		err = insertRadCheck(nil, username, input.Attribute, op, input.Value)
		if err != nil {
			return gin.H{"error": "Failed to add attribute: " + err.Error()}, http.StatusInternalServerError
		}
	}

	return gin.H{
		"message":   "Attribute updated successfully",
		"username":  username,
		"attribute": input.Attribute,
		"value":     input.Value,
	}, http.StatusOK
}

// DeleteRadCheckAttribute deletes a check attribute
func DeleteRadCheckAttribute(username string, attribute string) (gin.H, int) {
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Check if attribute exists
	exists, err = checkAttributeExists(username, attribute)
	if err != nil {
		return gin.H{"error": "Failed to check if attribute exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "Attribute not found"}, http.StatusNotFound
	}

	// Delete the attribute
	err = deleteRadCheckAttribute(username, attribute)
	if err != nil {
		return gin.H{"error": "Failed to delete attribute: " + err.Error()}, http.StatusInternalServerError
	}

	return gin.H{
		"message":   "Attribute deleted successfully",
		"username":  username,
		"attribute": attribute,
	}, http.StatusOK
}

// AddOrUpdateRadReplyAttribute adds or updates a reply attribute
func AddOrUpdateRadReplyAttribute(username string, input dto.RadReplyInput) (gin.H, int) {
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	op := input.Op
	if op == "" {
		op = defaultOp
	}

	// Check if attribute exists
	exists, err = replyAttributeExists(username, input.Attribute)
	if err != nil {
		return gin.H{"error": "Failed to check if attribute exists: " + err.Error()}, http.StatusInternalServerError
	}

	if exists {
		// Update existing attribute
		err = updateRadReplyAttribute(username, input.Attribute, op, input.Value)
		if err != nil {
			return gin.H{"error": "Failed to update attribute: " + err.Error()}, http.StatusInternalServerError
		}
	} else {
		// Insert new attribute
		err = insertRadReply(nil, username, input.Attribute, op, input.Value)
		if err != nil {
			return gin.H{"error": "Failed to add attribute: " + err.Error()}, http.StatusInternalServerError
		}
	}

	return gin.H{
		"message":   "Reply attribute updated successfully",
		"username":  username,
		"attribute": input.Attribute,
		"value":     input.Value,
	}, http.StatusOK
}

// DeleteRadReplyAttribute deletes a reply attribute
func DeleteRadReplyAttribute(username string, attribute string) (gin.H, int) {	
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Check if attribute exists
	exists, err = replyAttributeExists(username, attribute)
	if err != nil {
		return gin.H{"error": "Failed to check if attribute exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "Attribute not found"}, http.StatusNotFound
	}

	// Delete the attribute
	err = deleteRadReplyAttribute(username, attribute)
	if err != nil {
		return gin.H{"error": "Failed to delete attribute: " + err.Error()}, http.StatusInternalServerError
	}

	return gin.H{
		"message":   "Reply attribute deleted successfully",
		"username":  username,
		"attribute": attribute,
	}, http.StatusOK
}

// AddRadUserGroup adds a user to a group
func AddRadUserGroup(username string, input dto.RadUserGroupInput) (gin.H, int) {	
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	priority := 1
	if input.Priority != nil {
		priority = *input.Priority
	}

	// Check if user is already in the group
	exists, err = userGroupExists(username, input.Groupname)
	if err != nil {
		return gin.H{"error": "Failed to check if user is in group: " + err.Error()}, http.StatusInternalServerError
	}

	if exists {
		// Update priority
		err = updateRadUserGroup(username, input.Groupname, priority)
		if err != nil {
			return gin.H{"error": "Failed to update user group: " + err.Error()}, http.StatusInternalServerError
		}
	} else {
		// Add user to group
		err = insertRadUserGroup(nil, username, input.Groupname, priority)
		if err != nil {
			return gin.H{"error": "Failed to add user to group: " + err.Error()}, http.StatusInternalServerError
		}
	}

	return gin.H{
		"message":   "User added to group successfully",
		"username":  username,
		"groupname": input.Groupname,
		"priority":  priority,
	}, http.StatusOK
}

// DeleteRadUserGroup removes a user from a group
func DeleteRadUserGroup(username string, groupname string) (gin.H, int) {
	// Check if user exists
	exists, err := userExists(nil, username)
	if err != nil {
		return gin.H{"error": "Failed to check if user exists: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User not found"}, http.StatusNotFound
	}

	// Check if user is in the group
	exists, err = userGroupExists(username, groupname)
	if err != nil {
		return gin.H{"error": "Failed to check if user is in group: " + err.Error()}, http.StatusInternalServerError
	}
	if !exists {
		return gin.H{"error": "User is not a member of this group"}, http.StatusNotFound
	}

	// Remove user from group
	err = deleteRadUserGroup(username, groupname)
	if err != nil {
		return gin.H{"error": "Failed to delete user from group: " + err.Error()}, http.StatusInternalServerError
	}

	return gin.H{
		"message":   "User removed from group successfully",
		"username":  username,
		"groupname": groupname,
	}, http.StatusOK
}

// --- Helper Functions for Database Operations ---
// These functions abstract the raw SQL interactions.

// userExists checks if a user exists in the radcheck table.
func userExists(tx *gorm.DB, username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM radcheck WHERE username = ?"

	if tx == nil {
		tx = gdatabase.GetDB(config.RadiusDB) // must return *gorm.DB
		if tx == nil {
			return false, errors.New("database connection not available")
		}
	}

	if err := tx.Raw(query, username).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("failed to query user existence: %w", err)
	}

	return count > 0, nil
}



// insertRadCheck inserts a new row into the radcheck table.
func insertRadCheck(tx *gorm.DB, username, attribute, op, value string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	query := "INSERT INTO radcheck (username, attribute, op, value) VALUES (?, ?, ?, ?)"
	var err error
	if tx != nil {
		 err = tx.Exec(query, username, attribute, op, value).Error
	} else {
		result := db.Exec(query, username, attribute, op, value)
		if result.Error != nil {
			return fmt.Errorf("failed to insert into radcheck: %w", result.Error)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to insert into radcheck: %w", err)
	}
	return nil
}

// insertRadReply inserts a new row into the radreply table.
// insertRadReply inserts a new row into the radreply table.
func insertRadReply(tx *gorm.DB, username, attribute, op, value string) error {
	query := "INSERT INTO radreply (username, attribute, op, value) VALUES (?, ?, ?, ?)"
	if tx == nil {
		tx = gdatabase.GetDB(config.RadiusDB)
		if tx == nil {
			return errors.New("database connection not available")
		}
	}
	if err := tx.Exec(query, username, attribute, op, value).Error; err != nil {
		return fmt.Errorf("failed to insert into radreply: %w", err)
	}
	return nil
}

// insertRadUserGroup inserts a new row into the radusergroup table.
func insertRadUserGroup(tx *gorm.DB, username, groupname string, priority int) error {
	if tx == nil {
		tx = gdatabase.GetDB(config.RadiusDB)
		if tx == nil {
			return errors.New("database connection not available")
		}
	}

	if err := tx.Exec(
		"INSERT INTO radusergroup (username, groupname, priority) VALUES (?, ?, ?)",
		username, groupname, priority,
	).Error; err != nil {
		return fmt.Errorf("failed to insert into radusergroup: %w", err)
	}

	return nil
}


// getRadCheckAttributes retrieves all check attributes for a given username.
func getRadCheckAttributes(username string) ([]radiusmodel.RadCheck, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var attributes []radiusmodel.RadCheck
	err := db.Raw("SELECT id, username, attribute, op, value FROM radcheck WHERE username = ?", username).
		Scan(&attributes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query radcheck attributes: %w", err)
	}

	return attributes, nil
}


// getRadCheckAttributesTx retrieves all check attributes for a given username within a transaction.
func getRadCheckAttributesTx(tx *gorm.DB, username string) ([]radiusmodel.RadCheck, error) {
	var attributes []radiusmodel.RadCheck
	if err := tx.
		Where("username = ?", username).
		Find(&attributes).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve radcheck attributes (tx): %w", err)
	}
	return attributes, nil
}


// getRadReplyAttributes retrieves all reply attributes for a given username.
func getRadReplyAttributes(username string) ([]radiusmodel.RadReply, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var attributes []radiusmodel.RadReply
	err := db.Raw("SELECT id, username, attribute, op, value FROM radreply WHERE username = ?", username).
		Scan(&attributes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query radreply attributes: %w", err)
	}

	return attributes, nil
}

// getRadReplyAttributesTx retrieves all reply attributes for a given username within a transaction.
func getRadReplyAttributesTx(tx *gorm.DB, username string) ([]radiusmodel.RadReply, error) {
	var attributes []radiusmodel.RadReply
	if err := tx.
		Where("username = ?", username).
		Find(&attributes).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve radreply attributes (tx): %w", err)
	}
	return attributes, nil
}


// getRadUserGroups retrieves all group memberships for a given username.
func getRadUserGroups(username string) ([]radiusmodel.RadUserGroup, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var groups []radiusmodel.RadUserGroup
	err := db.Raw("SELECT id, username, groupname, priority FROM radusergroup WHERE username = ?", username).
		Scan(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query radusergroup memberships: %w", err)
	}

	return groups, nil
}


// getRadUserGroupsTx retrieves all group memberships for a given username within a transaction.
func getRadUserGroupsTx(tx *gorm.DB, username string) ([]radiusmodel.RadUserGroup, error) {
	var groups []radiusmodel.RadUserGroup
	if err := tx.
		Where("username = ?", username).
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve radusergroup memberships (tx): %w", err)
	}
	return groups, nil
}


// updateRadCheckAttribute updates an existing radcheck attribute.
func updateRadCheckAttribute(username, attribute, op, value string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	err := db.Exec("UPDATE radcheck SET op = ?, value = ? WHERE username = ? AND attribute = ?", op, value, username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to update radcheck attribute: %w", err)
	}
	return nil
}


// updateRadCheckAttributeTx updates an existing radcheck attribute within a transaction.
func updateRadCheckAttributeTx(tx *gorm.DB, username, attribute, op, value string) error {
	query := "UPDATE radcheck SET op = ?, value = ? WHERE username = ? AND attribute = ?"
	err := tx.Exec(query, op, value, username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to update radcheck attribute (tx): %w", err)
	}
	return nil
}

// updateRadReplyAttribute updates an existing radreply attribute.
func updateRadReplyAttribute(username, attribute, op, value string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	err := db.Exec("UPDATE radreply SET op = ?, value = ? WHERE username = ? AND attribute = ?", op, value, username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to update radreply attribute: %w", err)
	}
	return nil
}


// updateRadReplyAttributeTx updates an existing radreply attribute within a transaction.
func updateRadReplyAttributeTx(tx *gorm.DB, username, attribute, op, value string) error {
	query := "UPDATE radreply SET op = ?, value = ? WHERE username = ? AND attribute = ?"
	if err := tx.Exec(query, op, value, username, attribute).Error; err != nil {
		return fmt.Errorf("failed to update radreply attribute (tx): %w", err)
	}
	return nil
}


// updateRadUserGroup updates an existing radusergroup entry's priority.
func updateRadUserGroup(username, groupname string, priority int) error {
	db := gdatabase.GetDB(config.RadiusDB)
	err := db.Exec("UPDATE radusergroup SET priority = ? WHERE username = ? AND groupname = ?", priority, username, groupname).Error
	if err != nil {
		return fmt.Errorf("failed to update radusergroup: %w", err)
	}
	return nil
}


// updateRadUserGroupTx updates an existing radusergroup entry's priority within a transaction.
func updateRadUserGroupTx(tx *gorm.DB, username, groupname string, priority int) error {
	query := "UPDATE radusergroup SET priority = ? WHERE username = ? AND groupname = ?"
	err := tx.Exec(query, priority, username, groupname).Error
	if err != nil {
		return fmt.Errorf("failed to update radusergroup (tx): %w", err)
	}
	return nil
}

// deleteRadCheckAttribute deletes a specific radcheck attribute for a user.
func deleteRadCheckAttribute(username, attribute string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	err := db.Exec("DELETE FROM radcheck WHERE username = ? AND attribute = ?", username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to delete radcheck attribute: %w", err)
	}
	return nil
}


// deleteRadCheckAttributeTx deletes a specific radcheck attribute for a user within a transaction.
func deleteRadCheckAttributeTx(tx *gorm.DB, username, attribute string) error {
	query := "DELETE FROM radcheck WHERE username = ? AND attribute = ?"
	err := tx.Exec(query, username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to delete radcheck attribute (tx): %w", err)
	}
	return nil
}

// deleteRadReplyAttribute deletes a specific radreply attribute for a user.
func deleteRadReplyAttribute(username, attribute string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	err := db.Exec("DELETE FROM radreply WHERE username = ? AND attribute = ?", username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to delete radreply attribute: %w", err)
	}
	return nil
}


// deleteRadReplyAttributeTx deletes a specific radreply attribute for a user within a transaction.
func deleteRadReplyAttributeTx(tx *gorm.DB, username, attribute string) error {
	query := "DELETE FROM radreply WHERE username = ? AND attribute = ?"
	err := tx.Exec(query, username, attribute).Error
	if err != nil {
		return fmt.Errorf("failed to delete radreply attribute (tx): %w", err)
	}
	return nil
}

// deleteRadUserGroup deletes a user's membership from a specific group.
func deleteRadUserGroup(username, groupname string) error {
	db := gdatabase.GetDB(config.RadiusDB)
	query := "DELETE FROM radusergroup WHERE username = ? AND groupname = ?"
	 err := db.Exec(query, username, groupname).Error
	if err != nil {
		return fmt.Errorf("failed to delete radusergroup: %w", err)
	}
	return nil
}


// deleteRadUserGroupTx deletes a user's membership from a specific group within a transaction.
func deleteRadUserGroupTx(tx *gorm.DB, username, groupname string) error {
	query := "DELETE FROM radusergroup WHERE username = ? AND groupname = ?"
	err := tx.Exec(query, username, groupname).Error
	if err != nil {
		return fmt.Errorf("failed to delete radusergroup (tx): %w", err)
	}
	return nil
}
// checkAttributeExists checks if a radcheck attribute exists for a user.
func checkAttributeExists(username, attribute string) (bool, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var count int
	err := db.Raw("SELECT COUNT(*) FROM radcheck WHERE username = ? AND attribute = ?", username, attribute).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check radcheck attribute existence: %w", err)
	}
	return count > 0, nil
}

// replyAttributeExists checks if a radreply attribute exists for a user.
func replyAttributeExists(username, attribute string) (bool, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var count int
	err := db.Raw("SELECT COUNT(*) FROM radreply WHERE username = ? AND attribute = ?", username, attribute).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check radreply attribute existence: %w", err)
	}
	return count > 0, nil
}

// userGroupExists checks if a user is a member of a specific group.
func userGroupExists(username, groupname string) (bool, error) {
	db := gdatabase.GetDB(config.RadiusDB)
	var count int
	err := db.Raw("SELECT COUNT(*) FROM radusergroup WHERE username = ? AND groupname = ?", username, groupname).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check user group existence: %w", err)
	}
	return count > 0, nil
}

func IsUserExpired(username string) (string, error)  {
	db := gdatabase.GetDB(config.RadiusDB)
	
	exists, errEx := userExists(nil, username)
	if errEx != nil {
		return "error", errEx
	}
	if !exists {
		return  "NO_EXIST", nil
	}

	var expiration string
	err := db.Raw(`
		SELECT value FROM radcheck 
		WHERE username = ? AND attribute = 'Expiration' 
		LIMIT 1
	`, username).Scan(&expiration).Error
	if err != nil {
		return "error", fmt.Errorf("failed to fetch expiration for user %s: %w", username, err)
	}

	if expiration == "" {
		return "EXPIRED", nil
	}

	loc := time.Now().Location() // e.g., "Africa/Nairobi" if you're in EAT

	expireTime, err := time.ParseInLocation("Jan 2 2006 15:04:05", expiration, loc)
	if err != nil {
		return "error", fmt.Errorf("failed to parse expiration for user %s: %w", username, err)
	}

	currentTime := time.Now()

	if currentTime.After(expireTime) {
		return "EXPIRED", nil
	}


	return "NOT_EXPIRED", nil
}


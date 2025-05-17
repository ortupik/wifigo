// Package migrate to handle database schema migrations
package migrate

import (
	"fmt"

	"github.com/ortupik/wifigo/config"
	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	radiusmodel"github.com/ortupik/wifigo/server/database/model"
)

// MigrateRadiusModels automatically migrates only the FreeRADIUS related tables.
//
// - Only creates tables or columns if they are missing.
// - Does not change/delete existing columns or their types.
func MigrateRadiusModels( configure gconfig.Configuration) error {
	fmt.Println("Starting migration for RADIUS models...")

	db := gdatabase.GetDB(config.RadiusDB)
	configureDB := configure.Database.RDBMS[config.RadiusDB]
	driver := configureDB.Env.Driver

	// List of RADIUS models to migrate
	radiusModelsToMigrate := []interface{}{
		&radiusmodel.RadCheck{},
		&radiusmodel.RadReply{},
		&radiusmodel.RadUserGroup{},
		&radiusmodel.RadGroupCheck{},
		&radiusmodel.RadGroupReply{},
		&radiusmodel.RadAcct{},
	}

	if driver == "mysql" {
		// Set InnoDB engine specifically for MySQL
		if err := db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(
			radiusModelsToMigrate..., // Use the slice of RADIUS models
		); err != nil {
			return fmt.Errorf("failed to auto migrate RADIUS models for MySQL: %w", err)
		}
	} else {
		// AutoMigrate for other database drivers
		if err := db.AutoMigrate(
			radiusModelsToMigrate..., // Use the slice of RADIUS models
		); err != nil {
			return fmt.Errorf("failed to auto migrate RADIUS models: %w", err)
		}
	}

	fmt.Println("RADIUS tables migrated successfully!")
	return nil
}

// DropRadiusTables drops only the FreeRADIUS related tables.
// Use with caution as this will delete data in these tables.
func DropRadiusTables() error {
	fmt.Println("Dropping RADIUS tables...")
	db := gdatabase.GetDB(config.RadiusDB)

	// List of RADIUS models to drop.
	// Dropping order might be important if there were FK constraints,
	// but standard FreeRADIUS schema usually doesn't have them between these tables.
	// Start with radacct as it can grow large and is often less critical to drop first.
	radiusModelsToDrop := []interface{}{
		&radiusmodel.RadAcct{},
		&radiusmodel.RadCheck{},
		&radiusmodel.RadReply{},
		&radiusmodel.RadUserGroup{},
		&radiusmodel.RadGroupCheck{},
		&radiusmodel.RadGroupReply{},
	}

	if err := db.Migrator().DropTable(radiusModelsToDrop...); err != nil {
		return fmt.Errorf("failed to drop RADIUS tables: %w", err)
	}

	fmt.Println("RADIUS tables dropped successfully!")
	return nil
}
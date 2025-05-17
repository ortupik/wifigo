// Package migrate to migrate the schema for the server application
package migrate

import (
	"fmt"

	"github.com/ortupik/wifigo/config"
	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	gmodel "github.com/ortupik/wifigo/database/model"

	"github.com/ortupik/wifigo/server/database/model"
)

// Load all the models
type auth gmodel.Auth
type twoFA gmodel.TwoFA
type twoFABackup gmodel.TwoFABackup
type tempEmail gmodel.TempEmail
type user model.User

// Add aliases for the new payments and orders models
type payment model.Payment
type transaction model.Transaction
type order model.Order
type servicePlan model.ServicePlan

// DropAllTables - careful! It will drop all the tables!
func DropAllTables() error {
	db := gdatabase.GetDB(config.AppDB)

	if err := db.Migrator().DropTable(
		&transaction{},
		&payment{},
		&order{},
		&servicePlan{},
		&user{},
		&tempEmail{},
		&twoFABackup{},
		&twoFA{},
		&auth{},
		
	); err != nil {
		return err
	}

	fmt.Println("old tables are deleted!")
	return nil
}

// StartMigration - automatically migrate all the tables
//
// - Only create tables with missing columns and missing indexes
// - Will not change/delete any existing columns and their types
func StartMigration(configure gconfig.Configuration) error {
	db := gdatabase.GetDB(config.AppDB)
	configureDB := configure.Database.RDBMS[config.AppDB]
	driver := configureDB.Env.Driver

	if driver == "mysql" {
		// db.Set() --> add table suffix during auto migration
		if err := db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(
			&auth{},
			&twoFA{},
			&twoFABackup{},
			&tempEmail{},
			&user{},

			// Payments/Orders/ServicePlan models
			&servicePlan{},   // ServicePlan needs to exist before Order
			&order{},         // Order needs User and ServicePlan
			&payment{},       // Payment needs Order (and maybe User)
			&transaction{}, // Transaction needs User, Payment, and Order
		); err != nil {
			return err
		}

		fmt.Println("new tables are  migrated successfully!")
		return nil
	}

	fmt.Println("new tables are  migrated successfully!")
	return nil
}


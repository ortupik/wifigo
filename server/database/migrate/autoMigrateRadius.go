package migrate

import (
	"fmt"

	"github.com/ortupik/wifigo/config"
	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	radiusmodel "github.com/ortupik/wifigo/server/database/model" // Correct import alias
	"gorm.io/gorm"
)

// MigrateRadiusModels automatically migrates only the FreeRADIUS related tables.
//
// - Only creates tables or columns if they are missing.
// - Does not change/delete existing columns or their types.
func MigrateRadiusModels(configure gconfig.Configuration) error {
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

// SeedRadiusData seeds the FreeRADIUS database with initial data.
func SeedRadiusData() error {
	db := gdatabase.GetDB(config.RadiusDB)

	// Define the radgroupreply data.
	groupReplies := []radiusmodel.RadGroupReply{
		// 20 mins
		{Groupname: "min20@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "1200"},
		{Groupname: "min20@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "min40@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 2 hours
		{Groupname: "hour2@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "7200"},
		{Groupname: "hour2@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "hour2@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 6 hours
		{Groupname: "hour6@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "21600"},
		{Groupname: "hour6@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "hour6@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 12 hours
		{Groupname: "hour12@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "43200"},
		{Groupname: "hour12@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "hour12@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 1 day
		{Groupname: "daily@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "86400"},
		{Groupname: "daily@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "daily@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 3 days
		{Groupname: "daily3@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "259200"},
		{Groupname: "daily3@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "daily3@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 1 week
		{Groupname: "weekly@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "604800"},
		{Groupname: "weekly@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "weekly@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// 1 month
		{Groupname: "monthly@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthly@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthly@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 1"},

		// Home Wifi groups
		// Daily Home (5 Devices)
		{Groupname: "dailyHome@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "86400"},
		{Groupname: "dailyHome@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "dailyHome@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 5"},

		// 3 Days
		{Groupname: "daily3Home@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "259200"},
		{Groupname: "daily3Home@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "daily3Home@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 5"},

		// Weekly
		{Groupname: "weeklyHome@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "604800"},
		{Groupname: "weeklyHome@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "weeklyHome@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 5"},

		// Monthly Home 700 (2 Devices)
		{Groupname: "monthlyHome700@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthlyHome700@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthlyHome700@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 2"},

		// Monthly Home 1000 (5 Devices)
		{Groupname: "monthlyHome1000@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthlyHome1000@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthlyHome1000@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "3072k/3072k 3072k/3072k 5"},

		// 5 Mbps
		{Groupname: "monthlyHome5@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthlyHome5@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthlyHome5@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "5120k/5120k 5120k/5120k"},

		// 10 Mbps
		{Groupname: "monthlyHome10@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthlyHome10@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthlyHome10@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "10240k/10240k 10240k/10240k"},

		// 20 Mbps
		{Groupname: "monthlyHome20@Tecsurf", Attribute: "Session-Timeout", Op: ":=", Value: "2592000"},
		{Groupname: "monthlyHome20@Tecsurf", Attribute: "Idle-Timeout", Op: ":=", Value: "600"},
		{Groupname: "monthlyHome20@Tecsurf", Attribute: "Mikrotik-Rate-Limit", Op: ":=", Value: "20480k/20480k 20480k/20480k"},
	}

	// Loop through the group replies and create them if they don't exist.
	for _, gr := range groupReplies {
		var existingGR radiusmodel.RadGroupReply
		if err := db.Where("GroupName = ? AND Attribute = ?", gr.Groupname, gr.Attribute).First(&existingGR).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&gr).Error; err != nil {
					return fmt.Errorf("failed to create radgroupreply: %w", err)
				}
				fmt.Printf("RadGroupReply created for GroupName '%s', Attribute '%s'\n", gr.Groupname, gr.Attribute)
			} else {
				return fmt.Errorf("failed to query existing radgroupreply: %w", err)
			}
		} else {
			fmt.Printf("RadGroupReply already exists for GroupName '%s', Attribute '%s'\n", gr.Groupname, gr.Attribute)
		}
	}

	return nil
}

// RunRadiusMigrations - runs FreeRADIUS migrations and seeders.
func RunRadiusMigrations(configure gconfig.Configuration) error {
	if err := MigrateRadiusModels(configure); err != nil {
		return err
	}
	if err := SeedRadiusData(); err != nil {
		return err
	}
	return nil
}

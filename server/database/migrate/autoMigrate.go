package migrate

import (
	"fmt"
	"time"

	"github.com/ortupik/wifigo/config"
	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	gmodel "github.com/ortupik/wifigo/database/model"

	"github.com/ortupik/wifigo/server/database/model"
	"gorm.io/gorm"
)

// Load all the models
type auth gmodel.Auth
type twoFA gmodel.TwoFA
type twoFABackup gmodel.TwoFABackup
type tempEmail gmodel.TempEmail
type user model.User

type payment model.Payment
type order model.Order
type isp model.ISP
type servicePlan model.ServicePlan

// DropAllTables - careful! It will drop all the tables!
func DropAllTables() error {
	db := gdatabase.GetDB(config.AppDB)

	if err := db.Migrator().DropTable(
		&user{},
		&tempEmail{},
		&twoFABackup{},
		&twoFA{},
		&auth{},
		&payment{},
		&order{},
		&servicePlan{},
		&isp{},
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
			&isp{},
			&servicePlan{}, // ServicePlan needs to exist before Order
			&order{},       // Order needs User and ServicePlan
			&payment{},     // Payment needs Order (and maybe User)
		); err != nil {
			return err
		}

		fmt.Println("new tables are migrated successfully!")
		return nil
	}

	fmt.Println("new tables are migrated successfully!")
	return nil
}

// Seed - seeds the database with initial data
func Seed() error {
	db := gdatabase.GetDB(config.AppDB)

	// Seed ISP
	tecsurf := model.ISP{
		Name:    "Tecsurf",
		LogoURL: "https://example.com/tecsurf-logo.png", // Replace with actual URL
		DnsName: "tecsurf.co.ke",                   // Add the DnsName here
	}

	// Check if Tecsurf ISP already exists
	var existingISP model.ISP
	if err := db.Where("name = ?", tecsurf.Name).First(&existingISP).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create Tecsurf ISP
			if err := db.Create(&tecsurf).Error; err != nil {
				return err
			}
			fmt.Println("Tecsurf ISP created")
		} else {
			return err
		}
	} else {
		fmt.Println("Tecsurf ISP already exists")
	}

	// Seed Service Plans
	servicePlans := [] model.ServicePlan{
		{Name: "min1", Description: "1 Minute plan @ 3 Mbps for 1 device", Price: 1, Duration: 60, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Minute", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "min40", Description: "20 Minutes plan @ 3 Mbps for 1 device", Price: 5, Duration: 1200, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "20 Minutes", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "quickHour", Description: "2 Hours plan @ 3 Mbps for 1 device", Price: 10, Duration: 7200, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "2 Hours", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "hour6", Description: "6 Hours plan @ 3 Mbps for 1 device", Price: 20, Duration: 21600, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "6 Hours", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "hour8", Description: "12 Hours plan @ 3 Mbps for 1 device", Price: 27, Duration: 43200, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "12 Hours", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "daily", Description: "1 Day plan @ 3 Mbps for 1 device", Price: 35, Duration: 86400, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Day", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "daily3", Description: "3 Days plan @ 3 Mbps for 1 device", Price: 85, Duration: 259200, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "3 Days", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "weekly", Description: "1 Week plan @ 3 Mbps for 1 device", Price: 150, Duration: 604800, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Week", Speed: "3 Mbps", ServiceType: "Hotspot"},
		{Name: "monthly", Description: "1 Month plan @ 3 Mbps for 1 device", Price: 450, Duration: 2592000, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "3 Mbps", ServiceType: "Hotspot"},

		{Name: "dailyHome", Description: "1 Day plan @ 5 Mbps for 5 devices", Price: 60, Duration: 86400, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Day", Speed: "5 Mbps", ServiceType: "Home"},
		{Name: "daily3Home", Description: "3 Days plan @ 5 Mbps for 5 devices", Price: 150, Duration: 259200, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "3 Days", Speed: "5 Mbps", ServiceType: "Home"},
		{Name: "weeklyHome", Description: "1 Week plan @ 5 Mbps for 5 devices", Price: 250, Duration: 604800, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Week", Speed: "5 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome5", Description: "1 Month plan @ 5 Mbps with unlimited devices", Price: 1400, Duration: 2592000, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "5 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome10", Description: "1 Month plan @ 10 Mbps with unlimited devices", Price: 2200, Duration: 2592000, SpeedLimitMbps: "10", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "10 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome20", Description: "1 Month plan @ 20 Mbps with unlimited devices", Price: 3000, Duration: 2592000, SpeedLimitMbps: "20", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "20 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome500", Description: "1 Month plan @ 5 Mbps for 3 devices", Price: 500, Duration: 2592000, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "5 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome700", Description: "1 Month plan @ 3 Mbps for 2 devices", Price: 700, Duration: 2592000, SpeedLimitMbps: "3", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "3 Mbps", ServiceType: "Home"},
		{Name: "monthlyHome1000", Description: "1 Month plan @ 5 Mbps for 5 devices", Price: 1000, Duration: 2592000, SpeedLimitMbps: "5", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(), Validity: "1 Month", Speed: "5 Mbps", ServiceType: "Home"},
	}
	// Loop through the plans and create them if they don't exist
	for _, plan := range servicePlans {
		var existingPlan model.ServicePlan
		if err := db.Where("name = ?", plan.Name).First(&existingPlan).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				plan.ISPID = tecsurf.ID // Associate the plan with Tecsurf ISP
				if err := db.Create(&plan).Error; err != nil {
					return err
				}
				fmt.Printf("Service Plan '%s' created\n", plan.Name)
			} else {
				return err
			}
		} else {
			fmt.Printf("Service Plan '%s' already exists\n", plan.Name)
		}
	}

	return nil
}

// RunMigrations - runs all the migrations and seeders
func RunMigrations(configure gconfig.Configuration) error {
	if err := StartMigration(configure); err != nil {
		return err
	}
	if err := Seed(); err != nil {
		return err
	}
	return nil
}

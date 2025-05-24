package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ortupik/wifigo/badger"
	gconfig "github.com/ortupik/wifigo/config"
	gdatabase "github.com/ortupik/wifigo/database"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/queue"
	nconfig "github.com/ortupik/wifigo/server/config"
	//migrate "github.com/ortupik/wifigo/server/database/migrate"
	"github.com/ortupik/wifigo/server/router"
	service "github.com/ortupik/wifigo/server/service"
	"github.com/ortupik/wifigo/websocket"
)

// handleError simplifies error handling by logging and exiting the program.
func handleError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}

}

func main() {
	// Set configs
	err := gconfig.Config()
	handleError(err, "Failed to load configuration")

	err2 := nconfig.Config()
	handleError(err2, "Failed to NEW load configuration")

	// Read configurations
	configure := gconfig.GetConfig()

	// Initialize RDBMS client if configured
	if gconfig.IsRDBMS() {
		handleError(gdatabase.InitDB(), "Failed to initialize RDBMS")
		// Run migrations
		/*handleError(migrate.DropAllTables(), "Failed to drop app migrations")
		//handleError(migrate.DropRadiusTables(), "Failed to drop radius migrations")
		handleError(migrate.StartMigration(*configure), "Failed to run app migrations")
		//handleError(migrate.MigrateRadiusModels(*configure), "Failed to run radius migrations")
		handleError(migrate.Seed(), "Failed to run app seeders")
		//handleError(migrate.SeedRadiusData(), "Failed to run radius seeders")*/
	}

	if gconfig.IsRedis() {
		_, err = gdatabase.InitRedis()
		handleError(err, "Failed to initialize Redis")
	}

	store, err := badger.NewStore()
	handleError(err, "Failed to initialize BadgerDb")
	defer store.Close()

	// Initialize MikroTik manager
	mikrotikManager := mikrotik.NewManager()
	defer mikrotikManager.Close()

	mikrotikService := service.NewMikroTikManagerService(mikrotikManager)
	//should be put in a queue job -> adding only active devices(ping) to device manager
	err = mikrotikService.LoadAllDevices()
	if err != nil {
		handleError(err, "Failed to load devices from database!")
	}

	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Redis address from config
	redisAddr := configure.Database.REDIS.Env.Host + ":" + configure.Database.REDIS.Env.Port

	// Initialize queue client
	queueClient, err := queue.NewClient(redisAddr)
	if err != nil {
		log.Fatalf("Failed to create queue client: %v", err)
	}
	defer func() {
		if err := queueClient.Close(); err != nil {
			log.Printf("Error closing queue client: %v", err)
		}
	}()

	// Initialize handlers for queues
	MikrotikQueueHandler := queue.NewMikrotikQueueHandler(mikrotikService, wsHub)
	databaseQueueHandler := queue.NewDatabaseQueueHandler(wsHub)
	handlers := &queue.Handlers{
		MikrotikQueueHandler: *MikrotikQueueHandler,
		DatabaseQueueHandler: *databaseQueueHandler,
	}
	// Initialize and start queue server in a goroutine
	queueServer, err := queue.NewServer(redisAddr, mikrotikManager, wsHub, handlers) // Pass handlers
	if err != nil {
		log.Fatalf("Failed to create queue server: %v", err)
	}

	go func() {
		log.Println("Starting queue server...")
		if err := queueServer.Start(); err != nil {
			log.Fatalf("Failed to start queue server: %v", err)
		}
	}()

	// Set up router with our dependencies
	r, err := router.SetupRouter(configure, store, mikrotikManager, queueClient, wsHub)
	handleError(err, "Failed to setup router")

	// Set up graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%s", configure.Server.ServerHost, configure.Server.ServerPort)
		err := r.Run(configure.Server.ServerHost + ":" + configure.Server.ServerPort)
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for termination signal
	<-sigs
	log.Println("Received termination signal, starting graceful shutdown...")

	// Create a deadline to wait for
	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Wait for connections to drain
	log.Println("Shutting down server...")

	// Close queue server gracefully
	queueServer.GracefullyShutdown()
	log.Println("Queue server shut down successfully.")

	// Close MikroTik connections
	mikrotikManager.Close()
	log.Println("MikroTik connections closed successfully.")

	// Close database connections
	if gconfig.IsRDBMS() {
		gdatabase.CloseDB()
	}

	log.Println("Server shutdown complete.")
}

// Package database handles connections to different
// types of databases
package database

import (
	"context"
	"database/sql"
	"errors" // Import the errors package
	"fmt"
	"os"
	"strings" // Import the strings package
	"time"

	"github.com/ortupik/wifigo/config" // Assuming this is the updated config package

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql" // Import MySQL driver
	_ "github.com/jackc/pgx/v5/stdlib"  // Import PostgreSQL driver (using pgx)

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"

	// Import Redis Driver
	"github.com/mediocregopher/radix/v4"



	// Import Mongo driver
	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/event"
	opts "go.mongodb.org/mongo-driver/mongo/options"

	log "github.com/sirupsen/logrus"


)

// RecordNotFound record not found error message
const RecordNotFound string = "record not found"

// Map to hold multiple GORM database clients, keyed by name (e.g., "app", "radius")
var dbClients map[string]*gorm.DB

// Map to hold the underlying SQL database connections if needed, keyed by name
// var sqlDBs map[string]*sql.DB // Optional, uncomment if you need *sql.DB access

var err error // General error variable

// redisClient variable to access the redis client
var redisClient *radix.Client

// RedisConnTTL - context deadline in second
var RedisConnTTL int

// mongoClient instance
var mongoClient *qmgo.Client

// InitDB - function to initialize all configured RDBMS databases
// It now initializes connections for all RDBMS instances defined in the config
func InitDB() error {
	configureRDBMSMap := config.GetConfig().Database.RDBMS // Get the map of RDBMS configs

	// Initialize the map to store GORM clients
	dbClients = make(map[string]*gorm.DB)
	// sqlDBs = make(map[string]*sql.DB) // Initialize if needed

	// Check if RDBMS is activated globally in the config (assuming the config.Database struct has an Activate field or similar for RDBMS overall)
    // If ACTIVATE_RDBMS is "yes", the map will be populated. If not, the map will be empty.
    // We should check the activate field on each RDBMSConfig within the map.
	overallRDBMSActivated := strings.ToLower(strings.TrimSpace(os.Getenv("ACTIVATE_RDBMS"))) == config.Activated

	if !overallRDBMSActivated {
		log.Info("RDBMS is not activated, skipping database connection.")
		return nil // No RDBMS activated, nothing to do
	}

	if len(configureRDBMSMap) == 0 {
		log.Warning("RDBMS is activated, but no specific RDBMS configurations found in the config map.")
		return nil // Activated but no configurations found
	}


	var connectionErrors []error // Collect errors for multiple connections

	for dbName, dbConfig := range configureRDBMSMap {
		if dbConfig.Activate != config.Activated {
			log.Printf("RDBMS database '%s' is not activated, skipping connection.", dbName)
			continue // Skip if this specific RDBMS is not activated
		}

		log.Printf("Attempting to connect to RDBMS database: %s (Driver: %s)", dbName, dbConfig.Env.Driver)

		var db *gorm.DB
		var currentSQLDB *sql.DB // Use a local variable for the current SQL DB

		driver := dbConfig.Env.Driver
		username := dbConfig.Access.User
		password := dbConfig.Access.Pass
		database := dbConfig.Access.DbName
		host := dbConfig.Env.Host
		port := dbConfig.Env.Port
		sslmode := dbConfig.Ssl.Sslmode
		timeZone := dbConfig.Env.TimeZone
		maxIdleConns := dbConfig.Conn.MaxIdleConns
		maxOpenConns := dbConfig.Conn.MaxOpenConns
		connMaxLifetime := dbConfig.Conn.ConnMaxLifetime
		logLevel := dbConfig.Log.LogLevel

		var dsn string
		var openErr error

		switch driver {
		case "mysql":
			address := host
			if port != "" {
				address += ":" + port
			}
			// GORM recommends "&parseTime=True" and "&loc=Local" for handling time.Time
			dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				username, password, address, database)

			if sslmode == "" {
				sslmode = "disable"
			}
			if sslmode != "disable" {
				if sslmode == "require" {
					dsn += "&tls=true"
				}
				if sslmode == "verify-ca" || sslmode == "verify-full" {
					dsn += "&tls=custom"
					// Assuming InitTLSMySQL is generic or can be adapted for multiple connections
					// You might need to pass dbConfig.Ssl parameters to it.
					// If it's modifying global state, this might need careful consideration.
					// For now, keeping the original call, but note this potential issue.
					// Ensure InitTLSMySQL is defined and available or adapt it
					errTLS := InitTLSMySQL()
					if errTLS != nil {
						log.WithError(errTLS).Errorf("Failed to initialize TLS for MySQL database '%s': %v", dbName, errTLS)
						connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' TLS setup failed: %w", dbName, errTLS))
						continue // Skip connecting to this database if TLS setup fails
					}
				}
			}

			currentSQLDB, openErr = sql.Open("mysql", dsn)
			if openErr != nil {
				log.WithError(openErr).Errorf("Failed to open SQL connection for MySQL database '%s': %v", dbName, openErr)
				connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' SQL open failed: %w", dbName, openErr))
				continue // Skip connecting to this database
			}

			// Ping the database to verify connection
			pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Add a timeout for the ping
            // defer cancel() // Don't defer cancel here, do it inside the loop
            if pingErr := currentSQLDB.PingContext(pingCtx); pingErr != nil {
                cancel() // Call cancel on error
                log.WithError(pingErr).Errorf("Failed to ping database '%s': %v", dbName, pingErr)
                connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' ping failed: %w", dbName, pingErr))
                currentSQLDB.Close() // Close the failed connection
                continue // Skip connecting to this database
            }
             cancel() // Call cancel on success


			currentSQLDB.SetMaxIdleConns(maxIdleConns)       // max number of connections in the idle connection pool
			currentSQLDB.SetMaxOpenConns(maxOpenConns)       // max number of open connections in the database
			currentSQLDB.SetConnMaxLifetime(connMaxLifetime) // max amount of time a connection may be reused

			db, err = gorm.Open(mysql.New(mysql.Config{
				Conn: currentSQLDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.LogLevel(logLevel)),
			})
			if err != nil {
				log.WithError(err).Errorf("Failed to open GORM connection for MySQL database '%s': %v", dbName, err)
				connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' GORM open failed: %w", dbName, err))
				currentSQLDB.Close() // Close the underlying SQL connection
				continue             // Skip storing this connection
			}

			log.Printf("Successfully connected to MySQL database: %s", dbName)
			dbClients[dbName] = db
			// sqlDBs[dbName] = currentSQLDB // Store SQL DB if needed

		case "postgres":
			// Ensure you have imported the pgx driver: `_ "github.com/jackc/pgx/v5/stdlib"`
			address := "host=" + host
			if port != "" {
				address += " port=" + port
			}
			// Using fmt.Sprintf for cleaner string formatting
			dsn := fmt.Sprintf("%s user=%s dbname=%s password=%s TimeZone=%s",
				address, username, database, password, timeZone)

			if sslmode == "" {
				sslmode = "disable"
			}
			// Append SSL parameters only if sslmode is not disable
			if sslmode != "disable" {
				if dbConfig.Ssl.RootCA != "" {
					dsn += " sslrootcert=" + dbConfig.Ssl.RootCA
				} else if dbConfig.Ssl.ServerCert != "" {
					// Note: Using ServerCert as RootCA is unusual. Double-check this logic.
					// Typically, you use sslrootcert for the CA that signed the server cert.
					log.Warningf("Using ServerCert as sslrootcert for database '%s'", dbName)
					dsn += " sslrootcert=" + dbConfig.Ssl.ServerCert
				}
				if dbConfig.Ssl.ClientCert != "" {
					dsn += " sslcert=" + dbConfig.Ssl.ClientCert
				}
				if dbConfig.Ssl.ClientKey != "" {
					dsn += " sslkey=" + dbConfig.Ssl.ClientKey
				}
				dsn += " sslmode=" + sslmode
			} else {
                 dsn += " sslmode=" + sslmode // Still append sslmode=disable
            }


			currentSQLDB, openErr = sql.Open("pgx", dsn) // Use the pgx driver name
			if openErr != nil {
				log.WithError(openErr).Errorf("Failed to open SQL connection for PostgreSQL database '%s': %v", dbName, openErr)
				connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' SQL open failed: %w", dbName, openErr))
				continue // Skip connecting to this database
			}

            // Ping the database to verify connection
            pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Add a timeout for the ping
            // defer cancel() // Don't defer cancel here, do it inside the loop
            if pingErr := currentSQLDB.PingContext(pingCtx); pingErr != nil {
                cancel() // Call cancel on error
                log.WithError(pingErr).Errorf("Failed to ping database '%s': %v", dbName, pingErr)
                connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' ping failed: %w", dbName, pingErr))
                currentSQLDB.Close() // Close the failed connection
                continue // Skip connecting to this database
            }
            cancel() // Call cancel on success


			currentSQLDB.SetMaxIdleConns(maxIdleConns)       // max number of connections in the idle connection pool
			currentSQLDB.SetMaxOpenConns(maxOpenConns)       // max number of open connections in the database
			currentSQLDB.SetConnMaxLifetime(connMaxLifetime) // max amount of time a connection may be reused


			db, err = gorm.Open(postgres.New(postgres.Config{
				Conn: currentSQLDB,
			}), &gorm.Config{
				Logger: logger.Default.LogMode(logger.LogLevel(logLevel)),
			})
			if err != nil {
				log.WithError(err).Errorf("Failed to open GORM connection for PostgreSQL database '%s': %v", dbName, err)
				connectionErrors = append(connectionErrors, fmt.Errorf("database '%s' GORM open failed: %w", dbName, err))
				currentSQLDB.Close() // Close the underlying SQL connection
				continue             // Skip storing this connection
			}

			log.Printf("Successfully connected to PostgreSQL database: %s", dbName)
			dbClients[dbName] = db
			// sqlDBs[dbName] = currentSQLDB // Store SQL DB if needed

		default:
			log.Errorf("Unsupported RDBMS driver for database '%s': %s. Skipping.", dbName, driver)
			connectionErrors = append(connectionErrors, fmt.Errorf("database '%s': unsupported driver '%s'", dbName, driver))
			continue // Skip this driver
		}

	}

	if len(dbClients) == 0 && overallRDBMSActivated {
		return errors.New("RDBMS activated but no successful database connections were made")
	}

	// Return a combined error if any connections failed
	if len(connectionErrors) > 0 {
        // Join multiple errors into a single error
        errorMsgs := make([]string, len(connectionErrors))
        for i, e := range connectionErrors {
            errorMsgs[i] = e.Error()
        }
		return fmt.Errorf("multiple database connection errors: %s", strings.Join(errorMsgs, "; "))
	}

	return nil // All configured databases connected successfully (or none were configured/activated)
}

// GetDB - get a specific database connection by name
// Returns the GORM client for the specified database name, or nil if not found/connected
func GetDB(dbName string) *gorm.DB {
	client, ok := dbClients[dbName]
	if !ok {
		log.Warningf("Requested database connection '%s' not found or not initialized.", dbName)
		return nil // Or return an error, depending on desired behavior
	}
	return client
}

// GetSQLDB - get the underlying *sql.DB connection for a specific database by name
// Returns the *sql.DB connection, or nil if not found/connected (Requires uncommenting sqlDBs map)
/*
func GetSQLDB(dbName string) *sql.DB {
	client, ok := sqlDBs[dbName]
	if !ok {
		log.Warningf("Requested SQL database connection '%s' not found or not initialized.", dbName)
		return nil
	}
	return client
}
*/


// CloseDB - close all active RDBMS database connections
func CloseDB() {
	if dbClients == nil || len(dbClients) == 0 {
		log.Info("No RDBMS database connections to close.")
		return
	}

	for dbName, db := range dbClients {
		if db == nil {
			continue // Skip if the client is somehow nil
		}
		log.Printf("Closing RDBMS database connection: %s", dbName)
		sqlDB, err := db.DB()
		if err != nil {
			log.WithError(err).Errorf("Failed to get underlying SQL DB for '%s' before closing.", dbName)
			continue // Try to close the next one
		}
		err = sqlDB.Close()
		if err != nil {
			log.WithError(err).Errorf("Failed to close database connection '%s': %v", dbName, err)
		} else {
			log.Printf("Successfully closed database connection: %s", dbName)
		}
	}
	dbClients = nil // Clear the map after closing
	// sqlDBs = nil // Clear if using sqlDBs map
}


// --- Keep the Redis and Mongo functions as they were, accessing their specific config fields ---

// InitRedis - function to initialize redis client
func InitRedis() (*radix.Client, error) {
	configureRedis := config.GetConfig().Database.REDIS

	if configureRedis.Activate != config.Activated {
		log.Info("Redis is not activated, skipping Redis connection.")
		redisClient = nil // Ensure redisClient is nil if not activated
		return nil, nil // Not activated is not an error
	}


	RedisConnTTL = configureRedis.Conn.ConnTTL
	// Use a reasonable default timeout if ConnTTL is not set or 0
	timeout := time.Duration(RedisConnTTL) * time.Second
	if timeout <= 0 {
        timeout = 10 * time.Second // Default timeout
    }
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel() // Defer outside the function or handle within if needed

	// Using 0.0.0.0:6379 if host or port are empty for Radix's default behavior
	host := configureRedis.Env.Host
	port := configureRedis.Env.Port
	address := fmt.Sprintf("%s:%s", host, port)
	if host == "" || port == "" {
        address = "127.0.0.1:6379" // Radix default
        log.Warningf("Redis host or port not specified in config, using default address: %s", address)
    }


	rClient, err := (radix.PoolConfig{
		Size: configureRedis.Conn.PoolSize,
	}).New(ctx, "tcp", address)

    cancel() // Call cancel after New returns

	if err != nil {
		// Panic is strong; consider returning the error and letting the caller decide
		log.WithError(err).Errorf("Failed to initialize REDIS pool: %v", err)
		// log.WithError(err).Panic("panic code: 161") // Original panic
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

    // Optional: Ping Redis to verify connection
    // Check the error returned by the Do method directly
   // Optional: Ping Redis to verify connection
    // Check the error returned by the Do method directly
    pingErr := rClient.Do(context.Background(), radix.Cmd(nil, "PING")) // Call Do, which returns the error directly
    if pingErr != nil {
        log.WithError(pingErr).Errorf("Failed to ping Redis server: %v", pingErr)
         rClient.Close() // Close the pool if ping fails
        return nil, fmt.Errorf("redis ping failed: %w", pingErr)
    }
	

	log.Info("REDIS pool connection successful!")
	redisClient = &rClient

	return redisClient, nil
}

// GetRedis - get a connection
func GetRedis() *radix.Client {
	return redisClient
}


// InitMongo - function to initialize mongo client
func InitMongo() (*qmgo.Client, error) {
	configureMongo := config.GetConfig().Database.MongoDB

	if configureMongo.Activate != config.Activated {
		log.Info("MongoDB is not activated, skipping MongoDB connection.")
		mongoClient = nil // Ensure mongoClient is nil if not activated
		return nil, nil // Not activated is not an error
	}


	// Connect to the database or cluster
	uri := configureMongo.Env.URI
    if uri == "" {
        log.Error("MongoDB URI is not set in configuration.")
        return nil, errors.New("mongodb uri is not set")
    }

	// Use a reasonable default timeout if ConnTTL is not set or 0
	timeout := time.Duration(configureMongo.Env.ConnTTL) * time.Second
	if timeout <= 0 {
        timeout = 10 * time.Second // Default timeout
    }
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel() // Defer outside the function or handle within if needed

	clientConfig := &qmgo.Config{
		Uri: uri,
		// Use pointers for optional values in options.ClientOptions
		MaxPoolSize: func(v uint64) *uint64 { return &v }(configureMongo.Env.PoolSize),
	}

	// Only apply ServerAPIOptions if needed (depends on MongoDB version and setup)
	var serverAPIOptions *opts.ServerAPIOptions
	// if needed, configure serverAPIOptions here, e.g.:
	// serverAPIOptions = opts.ServerAPI(opts.ServerAPIVersion1)


	opt := opts.Client().SetAppName(configureMongo.Env.AppName)
	if serverAPIOptions != nil {
        opt.SetServerAPIOptions(serverAPIOptions)
    }


	// for monitoring pool
	if strings.ToLower(strings.TrimSpace(configureMongo.Env.PoolMon)) == config.Activated {
		poolMonitor := &event.PoolMonitor{
			Event: func(evt *event.PoolEvent) {
				switch evt.Type {
				case event.GetSucceeded:
					// log.Debug("MongoDB Pool: GetSucceeded") // Use Debug or Trace level
				case event.ConnectionReturned:
					// log.Debug("MongoDB Pool: ConnectionReturned") // Use Debug or Trace level
				case event.ConnectionCreated:
					// log.Debug("MongoDB Pool: ConnectionCreated")
				case event.ConnectionClosed:
					// log.Debugf("MongoDB Pool: ConnectionClosed, Reason: %s", evt.Reason)
				case event.PoolCleared:
                     // log.Debug("MongoDB Pool: PoolCleared")
                // Add other relevant events if needed
				default:
                    // log.Debugf("MongoDB Pool Event: %s", evt.Type)
				}
			},
		}
		opt.SetPoolMonitor(poolMonitor)
	}

	client, err := qmgo.NewClient(ctx, clientConfig, options.ClientOptions{ClientOptions: opt})

    cancel() // Call cancel after NewClient returns

	if err != nil {
        log.WithError(err).Errorf("Failed to initialize MongoDB client: %v", err)
		return nil, fmt.Errorf("mongodb connection failed: %w", err)
	}

    // Ping the deployment to see if the connection was successful
    pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second) // Add a timeout for the ping
    defer pingCancel()

   // Attempt to get the deadline from the context
   deadline, ok := pingCtx.Deadline()
   var pingTimeout int64 = 0 // Default to 0 or a small value if no deadline

   if ok {
	   // Calculate remaining duration and convert to milliseconds
	   remaining := time.Until(deadline)
	   if remaining > 0 {
		   pingTimeout = remaining.Milliseconds()
	   }
   }

   // Call Ping with the calculated timeout (assuming the signature wants int64 based on your error)
   // NOTE: The standard qmgo.Client.Ping signature is usually (context.Context, readpref.ReadPref).
   // This call assumes a non-standard signature based on the compilation error message you provided.
   if pingErr := client.Ping(pingTimeout); pingErr != nil { // Pass int64 here
	   log.WithError(pingErr).Errorf("Failed to ping MongoDB deployment: %v", pingErr)
	   client.Close(context.Background()) // Close the client if ping fails
	   return nil, fmt.Errorf("mongodb ping failed: %w", pingErr)
   }


	log.Info("MongoDB pool connection successful!")
	mongoClient = client

	return mongoClient, nil
}

// GetMongo - get a connection
func GetMongo() *qmgo.Client {
	return mongoClient
}


// --- Assume InitTLSMySQL exists elsewhere and handles MySQL TLS setup ---
// You need to ensure this function is defined or remove calls to it if not needed.
// It might need to be adapted to handle specific paths/configs for APP_DB and RADIUS_DB.
/*
func InitTLSMySQL() error {
   // Example (might need adaptation based on your actual TLS setup logic)
   // This function would typically register the custom TLS config with the MySQL driver
   // based on root CA, client cert, and client key paths provided in the config.
   // Check the go-sql-driver/mysql documentation for details on tls.Register.
   log.Info("Calling InitTLSMySQL - ensure this function is implemented correctly.")
   return nil // Placeholder
}
*/


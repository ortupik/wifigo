package config

import (
	"time"
)

// DatabaseConfig - all database variables
type DatabaseConfig struct {
	// relational database
	RDBMS map[string] RDBMSConfig // Use a map for multiple RDBMS instances

	// redis database
	REDIS REDIS

	// mongo database
	MongoDB MongoDB

	BadgerDB BadgerDb

	
}

// RDBMS - relational database variables
type RDBMS struct {
	Activate string
	Env      struct {
		Driver   string
		Host     string
		Port     string
		TimeZone string
	}
	Access struct {
		DbName string
		User   string
		Pass   string
	}
	Ssl struct {
		Sslmode    string
		MinTLS     string
		RootCA     string
		ServerCert string
		ClientCert string
		ClientKey  string
	}
	Conn struct {
		MaxIdleConns    int
		MaxOpenConns    int
		ConnMaxLifetime time.Duration
	}
	Log struct {
		LogLevel int
	}
}

// REDIS - redis database variables
type REDIS struct {
	Activate string
	Env      struct {
		Host string
		Port string
	}
	Conn struct {
		PoolSize int
		ConnTTL  int
	}
}

// MongoDB - mongo database variables
type MongoDB struct {
	Activate string
	Env      struct {
		AppName  string
		URI      string
		PoolSize uint64
		PoolMon  string
		ConnTTL  int
	}
}

type BadgerDb struct {
	Activate string
	Env      struct {
		DataDir  string
	}
}

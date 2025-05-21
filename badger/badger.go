package badger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/dgraph-io/badger/v3"
	gconfig "github.com/ortupik/wifigo/config" // Assuming your main config package
	// Assuming these are your actual config package paths
	// mpesaconfig "github.ortupik/wifigo/mpesa"
	// radiusconfig "github.ortupik/wifigo/radius"
	// databaseconfig "github.ortupik/wifigo/databaseconfig"
)

// ConfigType is an enum to represent different configuration types
type ConfigType string

const (
	DeviceConfigType  ConfigType = "device"
	MpesaConfigType   ConfigType = "mpesa"
	RadiusConfigType  ConfigType = "radius"
	DatabaseConfigType ConfigType = "database"
)

// StorableConfig is an interface that all configuration types should implement
type StorableConfig interface {
	GetID() string
	GetISPID() string // Optional, but useful for organization
	GetType() ConfigType
}

// DeviceConfigWrapper wraps the gconfig.DeviceConfig to implement StorableConfig
type DeviceConfigWrapper struct {
	gconfig.DeviceConfig
}

// GetID implements StorableConfig
func (w DeviceConfigWrapper) GetID() string {
	return w.ID
}

// GetISPID implements StorableConfig
func (w DeviceConfigWrapper) GetISPID() string {
	return w.ISPID
}

// GetType implements StorableConfig
func (w DeviceConfigWrapper) GetType() ConfigType {
	return DeviceConfigType
}

// MpesaConfigWrapper wraps the hypothetical mpesaconfig.MpesaConfig
type MpesaConfigWrapper struct {
	// mpesaconfig.MpesaConfig
	ID    string
	ISPID string
	Name  string
	// Add other M-Pesa specific fields
}

// GetID implements StorableConfig
func (w MpesaConfigWrapper) GetID() string {
	return w.ID
}

// GetISPID implements StorableConfig
func (w MpesaConfigWrapper) GetISPID() string {
	return w.ISPID
}

// GetType implements StorableConfig
func (w MpesaConfigWrapper) GetType() ConfigType {
	return MpesaConfigType
}

// RadiusConfigWrapper wraps the hypothetical radiusconfig.RadiusConfig
type RadiusConfigWrapper struct {
	// radiusconfig.RadiusConfig
	ID          string
	ISPID       string
	Server      string
	Port        int
	Secret      string
	Description string
	// Add other RADIUS specific fields
}

// GetID implements StorableConfig
func (w RadiusConfigWrapper) GetID() string {
	return w.ID
}

// GetISPID implements StorableConfig
func (w RadiusConfigWrapper) GetISPID() string {
	return w.ISPID
}

// GetType implements StorableConfig
func (w RadiusConfigWrapper) GetType() ConfigType {
	return RadiusConfigType
}

// DatabaseConfigWrapper wraps the hypothetical databaseconfig.DatabaseConfig
type DatabaseConfigWrapper struct {
	// databaseconfig.DatabaseConfig
	ID          string
	ISPID       string
	Host        string
	Port        int
	Username    string
	Password    string
	Database    string
	Description string
	// Add other Database specific fields
}

// GetID implements StorableConfig
func (w DatabaseConfigWrapper) GetID() string {
	return w.ID
}

// GetISPID implements StorableConfig
func (w DatabaseConfigWrapper) GetISPID() string {
	return w.ISPID
}

// GetType implements StorableConfig
func (w DatabaseConfigWrapper) GetType() ConfigType {
	return DatabaseConfigType
}

// Store represents the generic storage layer
type Store struct {
	db *badger.DB
}

// NewStore creates a new storage instance
func NewStore(dbPath string) (*Store, error) {
	opts := badger.DefaultOptions(gconfig.GetConfig().Badger.DataDir)
	opts.Logger = nil // Disable logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveConfig saves a configuration of any supported type
func (s *Store) SaveConfig(config StorableConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config of type %s: %w", config.GetType(), err)
	}

	key := []byte(fmt.Sprintf("%s:%s", config.GetType(), config.GetID()))

	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})

	if err != nil {
		return fmt.Errorf("failed to save config of type %s: %w", config.GetType(), err)
	}

	// Optionally store a reference by ISP if the config has an ISP ID
	ispID := config.GetISPID()
	if ispID != "" {
		ispKey := []byte(fmt.Sprintf("isp:%s:%s:%s", ispID, config.GetType(), config.GetID()))
		return s.db.Update(func(txn *badger.Txn) error {
			return txn.Set(ispKey, []byte(config.GetID()))
		})
	}

	return nil
}

// GetConfig retrieves a configuration by its type and ID
func (s *Store) GetConfig(configType ConfigType, id string, out interface{}) error {
	key := []byte(fmt.Sprintf("%s:%s", configType, id))

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, out)
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return fmt.Errorf("%s config not found: %s", configType, id)
		}
		return fmt.Errorf("failed to get %s config: %w", configType, err)
	}

	return nil
}

// ListConfigsByISP lists all configurations of a specific type for a given ISP
func (s *Store) ListConfigsByISP(ispID string, configType ConfigType, outSlice interface{}) error {
	prefix := []byte(fmt.Sprintf("isp:%s:%s:", ispID, configType))
	slicePtrValue := reflect.ValueOf(outSlice)
	if slicePtrValue.Kind() != reflect.Ptr || slicePtrValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("outSlice must be a pointer to a slice")
	}
	sliceValue := slicePtrValue.Elem()
	elementType := sliceValue.Type().Elem()

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix);  it.Next() {
			var configID string
			err := it.Item().Value(func(val []byte) error {
				configID = string(val)
				return nil
			})
			if err != nil {
				return err
			}

			// Fetch the actual config
			configKey := []byte(fmt.Sprintf("%s:%s", configType, configID))
			configItem, err := txn.Get(configKey)
			if err != nil {
				fmt.Printf("Error fetching config %s:%s: %v\n", configType, configID, err)
				continue
			}

			configValue := reflect.New(elementType).Interface()
			err = configItem.Value(func(val []byte) error {
				return json.Unmarshal(val, configValue)
			})
			if err != nil {
				return err
			}
			sliceValue = reflect.Append(sliceValue, reflect.ValueOf(configValue).Elem())
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to list %s configs for ISP %s: %w", configType, ispID, err)
	}

	reflect.ValueOf(outSlice).Elem().Set(sliceValue)
	return nil
}

// DeleteConfig deletes a configuration by its type and ID
func (s *Store) DeleteConfig(configType ConfigType, id string) error {
	key := []byte(fmt.Sprintf("%s:%s", configType, id))

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s config: %w", configType, id, err)
	}

	// Optionally delete the ISP reference as well
	// This requires knowing the ISP ID, which might involve a Get operation first
	return nil
}

// SaveSessionValue saves a value associated with a session ID
func (s *Store) SaveSessionValue(sessionID, key string, value []byte, ttl time.Duration) error {
	sessionKey := []byte(fmt.Sprintf("session:%s:%s", sessionID, key))
	entry := badger.NewEntry(sessionKey, value).WithTTL(ttl)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(entry)
	})
}

// GetSessionValue retrieves a value associated with a session ID
func (s *Store) GetSessionValue(sessionID, key string) ([]byte, error) {
	sessionKey := []byte(fmt.Sprintf("session:%s:%s", sessionID, key))
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(sessionKey)
		if err != nil {
			return  err
		}
		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...) // Copy the value
			return nil
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("no value found for session %s and key %s", sessionID, key)
		}
		return nil, fmt.Errorf("failed to get value for session %s and key %s: %w", sessionID, key, err)
	}
	return value, nil
}

// DeleteSessionValue deletes a value associated with a session ID
func (s *Store) DeleteSessionValue(sessionID, key string) error {
	sessionKey := []byte(fmt.Sprintf("session:%s:%s", sessionID, key))
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(sessionKey)
	})
}

// Helper functions to list specific config types by ISP

func (s *Store) ListDeviceConfigsByISP(ispID string) ([]gconfig.DeviceConfig, error) {
	var wrappers []DeviceConfigWrapper
	err := s.ListConfigsByISP(ispID, DeviceConfigType, &wrappers)
	if err != nil {
		return nil, err
	}
	configs := make([]gconfig.DeviceConfig, len(wrappers))
	for i, w := range wrappers {
		configs[i] = w.DeviceConfig
	}
	return configs, nil
}

func (s *Store) ListMpesaConfigsByISP(ispID string) ([]MpesaConfigWrapper, error) {
	var wrappers []MpesaConfigWrapper
	err := s.ListConfigsByISP(ispID, MpesaConfigType, &wrappers)
	if err != nil {
		return nil, err
	}
	return wrappers, nil
}

func (s *Store) ListRadiusConfigsByISP(ispID string) ([]RadiusConfigWrapper, error) {
	var wrappers []RadiusConfigWrapper
	err := s.ListConfigsByISP(ispID, RadiusConfigType, &wrappers)
	if err != nil {
		return nil, err
	}
	return wrappers, nil
}

func (s *Store) ListDatabaseConfigsByISP(ispID string) ([]DatabaseConfigWrapper, error) {
	var wrappers []DatabaseConfigWrapper
	err := s.ListConfigsByISP(ispID, DatabaseConfigType, &wrappers)
	if err != nil {
		return nil, err
	}
	return wrappers, nil
}
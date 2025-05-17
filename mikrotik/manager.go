package mikrotik

import (
	"errors"
	"log"
	"fmt"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	config "github.com/ortupik/wifigo/config"
)


// Manager handles multiple MikroTik device pools
type Manager struct {
	devices map[string]*DevicePool
	mu      sync.RWMutex
}

// DevicePool maintains a pool of connections for a specific MikroTik device
type DevicePool struct {
	config  config.DeviceConfig
	clients chan *routeros.Client
	mu      sync.Mutex
}

// NewManager creates a new MikroTik manager
func NewManager() *Manager {
	return &Manager{
		devices: make(map[string]*DevicePool),
	}
}

// AddDevice adds a new MikroTik device to the manager and establishes connections
func (m *Manager) AddDevice(config config.DeviceConfig) error {
	if config.ID == "" {
		return errors.New("device ID cannot be empty")
	}
	
	if config.PoolSize <= 0 {
		config.PoolSize = 5 // Default pool size
	}
	
	pool, err := newDevicePool(config)
	if err != nil {
		return err
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[config.ID] = pool
	
	return nil
}

// GetDevice returns a specific device pool by ID
func (m *Manager) GetDevice(deviceID string) (*DevicePool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	pool, exists := m.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("no device with ID %s found", deviceID)
	}
	
	return pool, nil
}

// GetDevicesByISP returns all device pools for a specific ISP
func (m *Manager) GetDevicesByISP(ispID string) []*DevicePool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var results []*DevicePool
	for _, pool := range m.devices {
		if pool.config.ISPID == ispID {
			results = append(results, pool)
		}
	}
	
	return results
}

// ListAllDevices returns all device configurations
func (m *Manager) ListAllDevices() []config.DeviceConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var configs []config.DeviceConfig
	for _, pool := range m.devices {
		configs = append(configs, pool.config)
	}
	
	return configs
}

// RemoveDevice removes a device and closes all connections
func (m *Manager) RemoveDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	pool, exists := m.devices[deviceID]
	if !exists {
		return fmt.Errorf("no device with ID %s found", deviceID)
	}
	
	pool.Close()
	delete(m.devices, deviceID)
	
	return nil
}

// Close closes all connections for all devices
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, pool := range m.devices {
		pool.Close()
	}
}

// newDevicePool creates a new connection pool for a MikroTik device
func newDevicePool(config config.DeviceConfig) (*DevicePool, error) {
	p := &DevicePool{
		config:  config,
		clients: make(chan *routeros.Client, config.PoolSize),
	}

	// Create initial connections
	for i := 0; i < config.PoolSize; i++ {
		client, err := routeros.Dial(config.Address, config.Username, config.Password)
		if err != nil {
			// Close the connections we've already established
			for j := 0; j < i; j++ {
				client := <-p.clients
				client.Close()
			}
			return nil, fmt.Errorf("failed to create client %d: %w", i, err)
		}
		p.clients <- client
	}

	return p, nil
}

// GetClient gets a client from the pool
func (p *DevicePool) GetClient() *routeros.Client {
	select {
	case client := <-p.clients:
		return client
	case <-time.After(5 * time.Second):
		// If we can't get a client in 5 seconds, try to create a new one
		client, err := routeros.Dial(p.config.Address, p.config.Username, p.config.Password)
		if err != nil {
			// Return nil if we can't create a new client
			return nil
		}
		return client
	}
}

// ReturnClient returns a client to the pool
func (p *DevicePool) ReturnClient(client *routeros.Client) {
	if client == nil {
		return
	}
	
	select {
	case p.clients <- client:
		// Client returned to pool
	default:
		// Pool is full, close this connection
		client.Close()
	}
}

// Execute executes a command on the MikroTik device
func (p *DevicePool) Execute(command string, args ...string) ([]map[string]string, error) {
	client := p.GetClient()
	if client == nil {
		return nil, errors.New("failed to get client from pool")
	}
	defer p.ReturnClient(client)

	sentence := append([]string{command}, args...)

	reply, err := client.RunArgs(sentence)

	if err != nil {
		log.Println("error:", err)
		return nil, fmt.Errorf("%w", err)
	}

	var result []map[string]string
	for _, re := range reply.Re {
		entry := make(map[string]string)
		for k, v := range re.Map {
			entry[k] = v
		}
		result = append(result, entry)
	}
	return result, nil
}




// Close closes all connections in the pool
func (p *DevicePool) Close() {
	for i := 0; i < cap(p.clients); i++ {
		select {
		case client := <-p.clients:
			client.Close()
		default:
			// No more clients in the pool
			return
		}
	}
}
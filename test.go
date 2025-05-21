package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/go-routeros/routeros/v3"
)

// RouterConfig holds the configuration for connecting to the MikroTik router
type RouterConfig struct {
	Address  string
	Username string
	Password string
}

// loginHotspotDeviceByAddress logs in a device to the hotspot using its IP address
// It first finds the device's to-address, then performs a hotspot login
func loginHotspotDeviceByAddress(config RouterConfig, address string) error {
	// Connect to MikroTik router
	client, err := routeros.Dial(config.Address, config.Username, config.Password)
	if err != nil {
		return fmt.Errorf("failed to connect to router: %v", err)
	}
	defer client.Close()

	// Find the hotspot host entry for the given address
	reply, err := client.Run("/ip/hotspot/host/print", "?address="+address)
	if err != nil {
		return fmt.Errorf("host print command failed: %v", err)
	}

	// Check if we found any hosts
	if len(reply.Re) == 0 {
		return errors.New("no hotspot host found with the specified address")
	}

	// Extract the to-address
	toAddress, ok := reply.Re[0].Map["to-address"]
	if !ok {
		return errors.New("to-address not found for the specified host")
	}

	// Extract the mac-address (may be needed for login)
	macAddress, hasMac := reply.Re[0].Map["mac-address"]
	
	fmt.Printf("Found hotspot host: address=%s, to-address=%s\n", address, toAddress)
	
	// Perform the hotspot login
	loginCmd := []string{"/ip/hotspot/active/login"}

	user := "0704624179@Tecsurf"
	
	// Add the IP address and to-address as parameters
	loginCmd = append(loginCmd, "=ip="+toAddress, "=user="+user)
	
	// Add MAC address if available
	if hasMac {
		loginCmd = append(loginCmd, "=mac-address="+macAddress)
	}
	
	// Execute the login command
	loginReply, err := client.RunArgs(loginCmd)
	if err != nil {
		return fmt.Errorf("hotspot login command failed: %v", err)
	}
	
	fmt.Printf("Login successful. Response: %v\n", loginReply)
	return nil
}

func main2() {
	// Configure router connection
	config := RouterConfig{
		Address:  "192.168.6.1:8728",
		Username: "admin",
		Password: "12345678",
	}
	
	// Login a device by its IP address
	address := "192.168.6.109"
	err := loginHotspotDeviceByAddress(config, address)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
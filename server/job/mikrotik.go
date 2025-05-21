package job

import (
	"errors"
	"fmt"

	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/server/dto"
)

// LoginHotspotDeviceByAddress logs in a device to the hotspot using its IP address
func LoginHotspotDeviceByAddress(manager *mikrotik.Manager, payload dto.MikrotikLogin) error {
	pool, err := manager.GetDevice(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Find the hotspot host entry for the given address
	hosts, err := pool.Execute("/ip/hotspot/host/print", "?address="+payload.Address)
	if err != nil {
		return fmt.Errorf("host print command failed: %v", err)
	}

	// Check if we found any hosts
	if len(hosts) == 0 {
		return errors.New("no hotspot host found with the specified address: "+payload.Address)
	}

	// Extract the to-address
	toAddress, ok := hosts[0]["to-address"]
	if !ok {
		return errors.New("to-address not found for the specified host")
	}

	// Extract the mac-address (may be needed for login)
	macAddress, hasMac := hosts[0]["mac-address"]
	
	fmt.Printf("Found hotspot host: address=%s, to-address=%s\n", payload.Address, toAddress)
	
	// Perform the hotspot login
	loginCmd := "/ip/hotspot/active/login"
	
	// Add the IP address, user and other parameters
	loginArgs := []string{
		"=ip=" + toAddress,
		"=user=" + payload.Username,
	}
	
	// Add password if provided
	if payload.Password != "" {
		loginArgs = append(loginArgs, "=password="+payload.Password)
	}
	
	// Add MAC address if available
	if hasMac {
		loginArgs = append(loginArgs, "=mac-address="+macAddress)
	}
	
	// Execute the login command
	loginReply, err := pool.Execute(loginCmd, loginArgs...)
	if err != nil {
		return fmt.Errorf("hotspot login command failed: %v", err)
	}
	
	fmt.Printf("Login successful. Response: %v\n", loginReply)
	return nil
}
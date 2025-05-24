package service

import (
	"errors"
	"fmt"
	"github.com/ortupik/wifigo/server/dto"
)

// ExecuteOnDevice executes a command on a specific device
func (s *MikroTikMangerService) ExecuteOnDevice(deviceID, command string, args ...string) ([]map[string]string, error) {
	pool, err := s.GetDevicePool(deviceID)
	if err != nil {
		return nil, err
	}

	return pool.Execute(command, args...)
}

// TestDeviceConnection tests the connection to a specific device
func (s *MikroTikMangerService) TestDeviceConnection(deviceID string) error {
	pool, err := s.GetDevicePool(deviceID)
	if err != nil {
		return err
	}

	// Try a simple command to test connection
	_, err = pool.Execute("/system/identity/print")
	return err
}

// GetDeviceStats gets basic statistics from a device
func (s *MikroTikMangerService) GetDeviceStats(deviceID string) (map[string]string, error) {
	pool, err := s.GetDevicePool(deviceID)
	if err != nil {
		return nil, err
	}

	// Get system resource information
	resources, err := pool.Execute("/system/resource/print")
	if err != nil {
		return nil, err
	}

	if len(resources) > 0 {
		return resources[0], nil
	}

	return make(map[string]string), nil
}

func LoginHotspotDeviceByAddress(s *MikroTikMangerService, payload dto.MikrotikLogin) error {

	pool, err := s.GetDevicePool(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	hosts, err := pool.Execute("/ip/hotspot/host/print", "?address="+payload.Address)
	if err != nil {
		return fmt.Errorf("host print command failed: %v", err)
	}

	if len(hosts) == 0 {
		return fmt.Errorf("no hotspot host found with address: %s", payload.Address)
	}

	hostEntry := hosts[0]
	toAddress, ok := hostEntry["to-address"]
	if !ok {
		// If to-address is not found, we can't proceed with the first preferred login method.
		// We will directly try with payload.Address.
		// Or, you could return an error here if to-address is strictly required for a first attempt.
		// For this requirement, we'll log it and then try payload.Address.
		fmt.Printf("Warning: to-address not found for host with address %s. Will attempt login with %s.\n", payload.Address, payload.Address)
		toAddress = "" // Ensure toAddress is empty if not found
	}

	macAddress, hasMac := hostEntry["mac-address"]

	fmt.Printf("Found hotspot host: initial address=%s, to-address=%s\n", payload.Address, toAddress)

	// --- Helper function for login attempt ---
	attemptLogin := func(loginIP string) error {
		if loginIP == "" {
			return errors.New("login IP address is empty")
		}
		loginCmd := "/ip/hotspot/active/login"
		loginArgs := []string{
			"=ip=" + loginIP,
			"=user=" + payload.Username,
		}

		if payload.Password != "" {
			loginArgs = append(loginArgs, "=password="+payload.Password)
		}
		if hasMac { // Use MAC address found from the host print command
			loginArgs = append(loginArgs, "=mac-address="+macAddress)
		}

		fmt.Printf("Attempting login with IP: %s, User: %s\n", loginIP, payload.Username)
		loginReply, err := pool.Execute(loginCmd, loginArgs...)
		if err != nil {
			return fmt.Errorf("hotspot login command failed for IP %s: %v", loginIP, err)
		}
		fmt.Printf("Login successful with IP %s. Response: %v\n", loginIP, loginReply)
		return nil
	}
	// --- End of helper function ---

	var firstAttemptErr error

	// 1. Try login with toAddress (if available)
	if toAddress != "" {
		fmt.Printf("Attempt 1: Logging in with to-address: %s\n", toAddress)
		err = attemptLogin(toAddress)
		if err == nil {
			return nil // Success
		}
		firstAttemptErr = err // Store the error from the first attempt
		fmt.Printf("Login attempt with to-address (%s) failed: %v\n", toAddress, err)
	} else {
		// This case occurs if toAddress was not found in the host entry.
		// We can consider this as the "first attempt" (which was skipped) having failed conceptually.
		firstAttemptErr = errors.New("to-address was not found, skipping first login attempt")
		fmt.Println("Skipping login attempt with to-address as it was not found.")
	}


	// 2. If first attempt failed (or was skipped), try with payload.Address
	fmt.Printf("Attempt 2: Logging in with payload address: %s\n", payload.Address)
	err = attemptLogin(payload.Address)
	if err == nil {
		return nil // Success
	}
	fmt.Printf("Login attempt with payload address (%s) failed: %v\n", payload.Address, err)

	// 3. If both attempts fail, return a consolidated error
	if firstAttemptErr != nil {
		return fmt.Errorf("all login attempts failed. Attempt 1 (toAddress: %s): %v. Attempt 2 (payload.Address: %s): %v", toAddress, firstAttemptErr, payload.Address, err)
	}
	// This case would be if toAddress was empty, so only payload.Address was tried and failed.
	return fmt.Errorf("login attempt with address %s failed: %v", payload.Address, err)

}
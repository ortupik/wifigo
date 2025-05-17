package queue

import "encoding/json"



type GenericTaskPayload struct {
	System string      `json:"system"`
	Action string      `json:"action"`
	Payload json.RawMessage `json:"payload"` // Raw JSON, to be decoded later
	Ip      string   `json:"ip"`           // IP address related to the command (e.g., user IP)
}
// Define payload structs
type MikrotikCommandPayload struct {
	DeviceID    string   `json:"device_id"`    // ID of the MikroTik device
	Command     string   `json:"command"`      // Command to execute on the device
	Args        []string `json:"args"`         // Arguments for the command
	Ip          string   `json:"ip"`           // IP address related to the command (e.g., user IP)
	CallbackURL string   `json:"callback_url"` // Optional URL for callback after command execution
}


type DatabaseOperationPayload struct {
	Ip      string          `json:"ip"`
	System  string          `json:"system"`
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/websocket"
)

// MikrotikHandler handles MikroTik related tasks.
type MikrotikHandler struct {
	manager *mikrotik.Manager
	wsHub   *websocket.Hub
}

func NewMikrotikHandler(manager *mikrotik.Manager, wsHub *websocket.Hub) *MikrotikHandler {
	return &MikrotikHandler{
		manager: manager,
		wsHub:   wsHub,
	}
}

func (h *MikrotikHandler) HandleTask(ctx context.Context, task *asynq.Task) error {
	var wrapper GenericTaskPayload
	if err := json.Unmarshal(task.Payload(), &wrapper); err != nil {
		return fmt.Errorf("failed to unmarshal generic task payload: %w", err)
	}

	if wrapper.System != "mikrotik" {
		return fmt.Errorf("invalid system for MikrotikHandler: %s", wrapper.System)
	}

	var cmdPayload MikrotikCommandPayload
	if err := json.Unmarshal(wrapper.Payload, &cmdPayload); err != nil {
		return fmt.Errorf("failed to unmarshal mikrotik command payload: %w", err)
	}

	err := h.executeCommand(ctx, &cmdPayload)
	if err != nil {
		if ShouldNotRetryError(err) {
			return asynq.SkipRetry
		}
		return err
	}

	return nil
}


func (h *MikrotikHandler) executeCommand(ctx context.Context, payload *MikrotikCommandPayload) error {
	log.Printf("Executing MikroTik command: %s on device: %s", payload.Command, payload.DeviceID)

	pool, err := h.manager.GetDevice(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	result, err := pool.Execute(payload.Command, payload.Args...)
	if err != nil {
		return fmt.Errorf("%w", err)
	}else{
		h.wsHub.SendToIP(payload.Ip, []byte(fmt.Sprintf(`{"type":"login", "status": "success", "message": %q}`, payload.Command)))
	}

	callbackPayload := map[string]interface{}{
		"deviceID": payload.DeviceID,
		"command":  payload.Command,
		"args":     payload.Args,
		"result":   result,
	}

	jsonPayload, err := json.Marshal(callbackPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	if payload.CallbackURL != "" {
		req, err := http.NewRequest(http.MethodPost, payload.CallbackURL, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("failed to create callback request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("callback request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Callback failed: status=%s, body=%s", resp.Status, string(body))
			return fmt.Errorf("callback failed with status: %s", resp.Status)
		}

		log.Printf("Callback succeeded: %s", payload.CallbackURL)
	}

	
	return nil
}

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/server/dto"
	job "github.com/ortupik/wifigo/server/job"
	"github.com/ortupik/wifigo/websocket"
)

// MikrotikHandler handles MikroTik related tasks.
type MikrotikHandler struct {
	manager *mikrotik.Manager
	wsHub   *websocket.Hub
	actionHandlers map[string]func(ctx context.Context, raw json.RawMessage) error
}

func NewMikrotikHandler(manager *mikrotik.Manager, wsHub *websocket.Hub) *MikrotikHandler {
	h := &MikrotikHandler{
		actionHandlers: make(map[string]func(ctx context.Context, raw json.RawMessage) error),
		wsHub:          wsHub,
		manager: manager,
	}
	h.registerHandlers()
	return h
}

func (h *MikrotikHandler) registerHandlers() {
	h.actionHandlers[ActionMikrotikLoginUser] = h.handleLoginUser
	//h.actionHandlers[ActionMikrotikCommand] = h.handleExecuteCommand
}

func (h *MikrotikHandler) HandleTask(ctx context.Context, task *asynq.Task) error {

	var payload GenericTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	if payload.System != "mikrotik" {
		return fmt.Errorf("invalid system for MikrotikHandler: %s", payload.System)
	}

	log.Printf("Processing MiktoTik operation: %s", payload.Action)

	handlerFunc, ok := h.actionHandlers[payload.Action]
	if !ok {
		return fmt.Errorf("unknown mikrotik action: %s", payload.Action)
	}
	return handlerFunc(ctx, payload.Payload)

}

func (h *MikrotikHandler) handleLoginUser(ctx context.Context, raw json.RawMessage) error {
    var data dto.MikrotikLogin
    if err := json.Unmarshal(raw, &data); err != nil {
        return fmt.Errorf("failed to decode payload: %w", err)
    }

    err := job.LoginHotspotDeviceByAddress(h.manager, data)
    if err != nil {
		h.wsHub.SendToIP(data.Address, []byte(fmt.Sprintf(`{"type":"login", "status": "failed", "message": "Could not log you in!"}`)) )
        if ShouldNotRetryError(err) {
			return asynq.SkipRetry
		}
		return fmt.Errorf("failed to login user: %w", err)
    }else{
		h.wsHub.SendToIP(data.Address,[]byte(fmt.Sprintf(`{"type":"login", "status": "success", "message": "You are now logged in"}`) ))		
      return nil
	}

}


/*func (h *MikrotikHandler) handleExecuteCommand(ctx context.Context, payload *MikrotikCommandPayload) error {
	log.Printf("Executing MikroTik command: %s on device: %s", payload.Command, payload.DeviceID)

	pool, err := h.manager.GetDevice(payload.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	result, err := pool.Execute(payload.Command, payload.Args...)
	if err != nil {
		return fmt.Errorf("%w", err)
	}else{
		h.wsHub.SendToIP(payload.Ip, []byte(fmt.Sprintf(`{"type":"command", "status": "success", "result": %q}`,result)))
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
}*/

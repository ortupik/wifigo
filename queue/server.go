package queue

import (
	"context"
	"fmt"
	"encoding/json"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/ortupik/wifigo/mikrotik"
	"github.com/ortupik/wifigo/websocket"
)

// Server is the server for processing tasks
type Server struct {
	server  *asynq.Server
	manager *mikrotik.Manager
	wsHub   *websocket.Hub
	handlers *Handlers // Use the Handlers struct from the same package
}

// Handler interface
type Handler interface {
	HandleTask(ctx context.Context, task *asynq.Task) error
}

// Handlers struct to hold all handler instances.
type Handlers struct {
	MikrotikHandler   MikrotikHandler // Use the struct directly, not the pointer
	DatabaseHandler DatabaseHandler // Use the struct directly, not the pointer
	// Add other handlers here as needed.
}

// NewServer creates a new queue server
func NewServer(redisAddr string, manager *mikrotik.Manager, wsHub *websocket.Hub, handlers *Handlers) (*Server, error) {
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Configure queue priorities and concurrency
			Queues: map[string]int{
				QueueCritical:  5, // Process 5 critical tasks at a time
				QueueDefault:   3, // Process 3 default tasks at a time
				QueueReporting: 2, // Process 2 reporting tasks at a time
			},
			Concurrency: 10, // Maximum number of concurrent tasks
			ErrorHandler: NewErrorHandler(wsHub),
		},
	)
	if server == nil {
		return nil, fmt.Errorf("failed to create asynq server")
	}
	return &Server{
		server:  server,
		manager: manager,
		wsHub:   wsHub,
		handlers: handlers,
	}, nil
}

// Start starts the queue server
func (s *Server) Start() error {
	mux := asynq.NewServeMux()

	// Register task handlers, using the Handlers struct.
	if s.handlers == nil {
		return fmt.Errorf("handlers is nil") //check if handlers are nil
	}
	
	mux.HandleFunc(TypeMikrotikCommand, s.handlers.MikrotikHandler.HandleTask)
	mux.HandleFunc(TypeDatabaseOperation, s.handlers.DatabaseHandler.HandleTask)

	return s.server.Start(mux)
}

// Stop stops the queue server
func (s *Server) Stop() {
	s.server.Stop()
}

// GracefullyShutdown gracefully shuts down the queue server
func (s *Server) GracefullyShutdown() {
	s.server.Shutdown()
}

type ErrorHandlerFunc func(ctx context.Context, task *asynq.Task, err error) (noRetry bool)

func ShouldNotRetryError(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := err.Error()
	return strings.Contains(errMsg, "is already logged in") 
}

func NewErrorHandler(wsHub *websocket.Hub) asynq.ErrorHandlerFunc {
	return func(ctx context.Context, task *asynq.Task, err error) {
		fmt.Printf("❌ Task %s failed: %v\n", task.Type(), err)

		switch task.Type() {
		case TypeMikrotikCommand:
			var payload GenericTaskPayload
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				fmt.Println("Failed to unmarshal MikrotikCommand payload:", err)
				return
			}
			msg := `{"type":"login","status":"failed","message":"` + err.Error() + `"}`

			if ShouldNotRetryError(err) {
				msg = `{"type":"login","status":"success","message":"You are already logged in"}`
			}
			wsHub.SendToIP(payload.Ip, []byte(fmt.Sprintf(msg)))

		case TypeDatabaseOperation:
			var payload GenericTaskPayload
			if err := json.Unmarshal(task.Payload(), &payload); err != nil {
				fmt.Println("Failed to unmarshal DatabaseOperation payload:", err)
				return
			}
			wsHub.SendToIP(payload.Ip, []byte(fmt.Sprintf(`{"type":"payment","status":"failed","message":%q}`, err)))
		}
	}
}

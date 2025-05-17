package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client is the client for enqueuing tasks
type Client struct {
	client *asynq.Client
}

// NewClient creates a new queue client
func NewClient(redisAddr string) (*Client, error) { // Return an error
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	if client == nil {
		return nil, fmt.Errorf("failed to create asynq client") // Return the error
	}
	return &Client{
		client: client,
	}, nil
}

// Close closes the queue client
func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueTask enqueues a generic task
func (c *Client) EnqueueTask(ctx context.Context, taskType string, payload interface{}, priority string, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, data)

	// Configure default task options based on priority
	var defaultOpts []asynq.Option

	switch priority {
	case QueueCritical:
		defaultOpts = []asynq.Option{
			asynq.Queue(QueueCritical),
			asynq.MaxRetry(5),
			asynq.Timeout(30 * time.Second),
			asynq.Retention(2 * time.Hour),
		}
	case QueueReporting:
		defaultOpts = []asynq.Option{
			asynq.Queue(QueueReporting),
			asynq.MaxRetry(3),
			asynq.Timeout(2 * time.Minute),
			asynq.Retention(24 * time.Hour),
		}
	default: // QueueDefault
		defaultOpts = []asynq.Option{
			asynq.Queue(QueueDefault),
			asynq.MaxRetry(3),
			asynq.Timeout(1 * time.Minute),
			asynq.Retention(6 * time.Hour),
		}
	}

	allOpts := append(defaultOpts, opts...)
	return c.client.EnqueueContext(ctx, task, allOpts...)
}

// EnqueueMikrotikCommand enqueues a MikroTik command task
func (c *Client) EnqueueMikrotikCommand(ctx context.Context, payload *MikrotikCommandPayload, priority string) (*asynq.TaskInfo, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MikrotikCommandPayload: %w", err)
	}

	genericPayload := GenericTaskPayload{
		System:  "mikrotik",
		Action:  "command",
		Payload: raw,
		Ip:      payload.Ip,
	}
	return c.EnqueueTask(ctx, TypeMikrotikCommand, genericPayload, priority)
}


// EnqueueDatabaseOperation enqueues a database operation task
func (c *Client) EnqueueDatabaseOperation(ctx context.Context, action string, payload interface{}, priority string) (*asynq.TaskInfo, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal database payload: %w", err)
	}

	genericPayload := GenericTaskPayload{
		System:  "mysql",
		Action:  action,
		Payload: raw,
	}
	return c.EnqueueTask(ctx, TypeDatabaseOperation, genericPayload, priority)
}




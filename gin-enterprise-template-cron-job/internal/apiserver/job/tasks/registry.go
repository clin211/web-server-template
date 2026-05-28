package tasks

import (
	"context"
	"log/slog"

	"github.com/google/wire"

	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

const (
	// TaskTypeNoop is the internal noop task type.
	TaskTypeNoop = "system:noop"
	// TaskTypeClientNoop is the public noop task type for scheduled task tests.
	TaskTypeClientNoop = "client:noop"
)

// ProviderSet contains task registry dependency providers.
var ProviderSet = wire.NewSet(NewRegistry)

// NewRegistry creates a task registry with built-in noop tasks.
func NewRegistry() *genericjob.Registry {
	registry := genericjob.NewRegistry()
	registry.MustRegister(genericjob.TaskDef{
		Type:            TaskTypeNoop,
		Handler:         handleNoop,
		DefaultQueue:    genericjob.DefaultQueue,
		AllowedQueues:   []string{genericjob.DefaultQueue, "low"},
		Visibility:      genericjob.VisibilityInternal,
		MaxPayloadBytes: 1024,
		RetryPolicy:     genericjob.RetryPolicy{MaxRetry: 1},
	})
	registry.MustRegister(genericjob.TaskDef{
		Type:            TaskTypeClientNoop,
		Handler:         handleNoop,
		DefaultQueue:    genericjob.DefaultQueue,
		AllowedQueues:   []string{genericjob.DefaultQueue, "low"},
		Visibility:      genericjob.VisibilityPublic,
		MaxPayloadBytes: 1024,
		RetryPolicy:     genericjob.RetryPolicy{MaxRetry: 1},
	})
	return registry
}

// handleNoop is a noop task handler for testing.
func handleNoop(ctx context.Context, task *genericjob.Task) error {
	slog.InfoContext(ctx, "Noop job task processed", "taskType", task.Type, "queue", task.Queue, "asynqTaskID", task.ID)
	return nil
}

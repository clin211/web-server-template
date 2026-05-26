package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

// ErrUnknownTaskError is used when a task fails without a concrete error.
var ErrUnknownTaskError = errors.New("unknown job task error")

// Worker processes scheduled job tasks from the queue.
type Worker struct {
	enabled  bool
	server   *asynq.Server
	registry *genericjob.Registry
	metrics  *genericjob.Metrics
	recorder genericjob.ExecutionRecorder
	tracer   trace.Tracer
}

// NewWorker creates a new job worker instance.
func NewWorker(rdb *redis.Client, registry *genericjob.Registry, metrics *genericjob.Metrics, recorder genericjob.ExecutionRecorder, opts *genericoptions.JobOptions) *Worker {
	if recorder == nil {
		recorder = genericjob.NoopExecutionRecorder{}
	}
	if opts == nil {
		opts = genericoptions.NewJobOptions()
	}

	cfg := asynq.Config{
		Concurrency:     opts.Async.Worker.Concurrency,
		Queues:          opts.Async.Worker.Queues,
		StrictPriority:  opts.Async.Worker.StrictPriority,
		ShutdownTimeout: opts.Async.Retry.Timeout,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			queue, _ := asynq.GetQueueName(ctx)
			slog.ErrorContext(ctx, "Asynq task failed", "taskType", task.Type(), "queue", queue, "error", err)
		}),
	}

	return &Worker{
		enabled:  opts.Async.Enabled,
		server:   asynq.NewServerFromRedisClient(rdb, cfg),
		registry: registry,
		metrics:  metrics,
		recorder: recorder,
		tracer:   otel.Tracer("github.com/clin211/gin-enterprise-template/internal/apiserver/job/worker"),
	}
}

// Start launches the asynq worker.
func (w *Worker) Start(ctx context.Context) error {
	if w == nil || !w.enabled {
		return nil
	}
	if w.server == nil {
		return fmt.Errorf("job worker server is not initialized")
	}
	if w.registry == nil {
		return fmt.Errorf("job registry is not initialized")
	}

	if err := w.server.Start(asynq.HandlerFunc(w.ProcessTask)); err != nil {
		return fmt.Errorf("start job worker: %w", err)
	}

	slog.InfoContext(ctx, "Job worker started")
	return nil
}

// Shutdown gracefully stops the worker.
func (w *Worker) Shutdown(ctx context.Context) {
	if w == nil || !w.enabled || w.server == nil {
		return
	}
	slog.InfoContext(ctx, "Stopping job worker")
	w.server.Stop()
	w.server.Shutdown()
}

// ProcessTask handles a single job task from the queue.
func (w *Worker) ProcessTask(ctx context.Context, asynqTask *asynq.Task) error {
	startedAt := time.Now()
	envelope, err := genericjob.UnmarshalEnvelope(asynqTask.Payload())
	if err != nil {
		return fmt.Errorf("unmarshal job task payload: %w", err)
	}

	ctx = genericjob.ExtractTraceContext(ctx, envelope.Trace)
	queue, _ := asynq.GetQueueName(ctx)
	taskID, _ := asynq.GetTaskID(ctx)

	ctx, span := w.tracer.Start(ctx, "job.worker.process",
		trace.WithAttributes(
			attribute.String("job.task_type", asynqTask.Type()),
			attribute.String("job.queue", queue),
			attribute.String("job.asynq_task_id", taskID),
			attribute.String("job.execution_id", envelope.Metadata[genericjob.MetadataExecutionID]),
			attribute.String("job.scheduled_task_id", envelope.Metadata[genericjob.MetadataScheduledTaskID]),
		),
	)
	defer span.End()

	task := &genericjob.Task{
		Type:            asynqTask.Type(),
		Payload:         envelope.Payload,
		ID:              taskID,
		Queue:           queue,
		ExecutionID:     envelope.Metadata[genericjob.MetadataExecutionID],
		ScheduledTaskID: envelope.Metadata[genericjob.MetadataScheduledTaskID],
		Metadata:        envelope.Metadata,
	}

	def, ok := w.registry.Get(asynqTask.Type())
	if !ok {
		err := fmt.Errorf("job task type %q is not registered", asynqTask.Type())
		w.recordFailed(ctx, task, startedAt, span, err)
		return err
	}

	if def.PayloadValidator != nil {
		if err := def.PayloadValidator(ctx, envelope.Payload); err != nil {
			err = fmt.Errorf("job task %q payload validation failed: %w", asynqTask.Type(), err)
			w.recordFailed(ctx, task, startedAt, span, err)
			return err
		}
	}

	if err := w.recorder.MarkRunning(ctx, task); err != nil {
		slog.WarnContext(ctx, "Failed to mark job task running", "taskType", asynqTask.Type(), "executionID", task.ExecutionID, "error", err)
	}
	if err := def.Handler(ctx, task); err != nil {
		w.recordFailed(ctx, task, startedAt, span, err)
		return err
	}

	duration := time.Since(startedAt)
	if err := w.recorder.MarkSucceeded(ctx, task, duration); err != nil {
		slog.WarnContext(ctx, "Failed to mark job task succeeded", "taskType", asynqTask.Type(), "executionID", task.ExecutionID, "error", err)
	}
	w.metrics.RecordProcessed(ctx, asynqTask.Type(), queue, "succeeded", duration)
	span.SetStatus(codes.Ok, "")
	slog.InfoContext(ctx, "Job task processed", "taskType", asynqTask.Type(), "queue", queue, "asynqTaskID", taskID, "duration", time.Since(startedAt))
	return nil
}

// recordFailed handles task failure: updates execution record, metrics, and span.
func (w *Worker) recordFailed(ctx context.Context, task *genericjob.Task, startedAt time.Time, span trace.Span, err error) {
	if err == nil {
		err = ErrUnknownTaskError
	}
	duration := time.Since(startedAt)
	if task != nil {
		if recordErr := w.recorder.MarkFailed(ctx, task, err, duration); recordErr != nil {
			slog.WarnContext(ctx, "Failed to mark job task failed", "taskType", task.Type, "executionID", task.ExecutionID, "error", recordErr)
		}
		w.metrics.RecordProcessed(ctx, task.Type, task.Queue, "failed", duration)
		slog.ErrorContext(ctx, "Job task failed", "taskType", task.Type, "queue", task.Queue, "error", err, "duration", duration)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

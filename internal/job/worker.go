package job

import (
	"context"
	"errors"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"sync"
	"time"
)

var errInvalidWorkerOptions = errors.New("invalid worker options")
var errWorkerDrainExceededUnfinishedLeasesWillExpire = errors.New("worker drain exceeded; unfinished leases will expire")
var errHandlerPanic = errors.New("handler panic")
var errResultTooLarge = errors.New("result too large")

// Repository is the worker's fenced persistence consumer contract.
type Repository interface {
	Claim(ctx context.Context, worker string, lease time.Duration) (*model.TaskM, error)
	Heartbeat(ctx context.Context, task *model.TaskM, lease time.Duration) (bool, error)
	Finish(ctx context.Context, task *model.TaskM, result Result, runErr error) error
}

// Observer records bounded local execution metrics.
type Observer interface {
	RecordExecution(ctx context.Context, outcome string, duration time.Duration)
	RecordLeaseLost(ctx context.Context)
}

// WorkerOptions sets independent execution and drain limits.
type WorkerOptions struct {
	Observer                      Observer
	ID                            string
	Concurrency                   int
	Poll, Lease, Heartbeat, Drain time.Duration
}

// Worker executes only as many leased tasks as it has available slots.
type Worker struct {
	repo     Repository
	registry *Registry
	options  WorkerOptions
}

// NewWorker rejects unsafe heartbeat, lease and capacity settings.
func NewWorker(repo Repository, r *Registry, o WorkerOptions) (*Worker, error) {
	if repo == nil || r == nil {
		return nil, errInvalidWorkerOptions
	}
	if err := validateWorkerOptions(o); err != nil {
		return nil, err
	}
	return &Worker{repo, r, o}, nil
}

// Run stops intake first, then drains while continuing heartbeats. Shutdown is bounded
// even for a handler that violates the cooperative context contract; its lease then expires.
func (w *Worker) Run(ctx context.Context) error {
	workCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	var wg sync.WaitGroup
	slots := make(chan struct{}, w.options.Concurrency)
	w.intake(ctx, workCtx, &wg, slots)
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	timer := time.NewTimer(w.options.Drain)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		cancel()
	}
	timer.Reset(2 * time.Second)
	select {
	case <-done:
		return nil
	case <-timer.C:
		return errWorkerDrainExceededUnfinishedLeasesWillExpire
	}
}
func (w *Worker) execute(root context.Context, t *model.TaskM) {
	started := time.Now()
	if t.TraceParent != nil {
		root = propagation.TraceContext{}.Extract(root, propagation.MapCarrier{"traceparent": *t.TraceParent})
	}
	root, span := otel.Tracer("tasks.worker").Start(root, "task.execute", trace.WithAttributes(attribute.String("task_id", t.TaskID), attribute.String("task_type", t.TaskType), attribute.Int("attempt", t.Attempts)))
	defer span.End()
	if span.SpanContext().IsValid() {
		id := span.SpanContext().TraceID().String()
		t.TraceID = &id
	}
	root = contextx.WithTraceID(root, span.SpanContext().TraceID().String())
	root = contextx.WithUserID(root, t.OwnerID)
	root = executionContext(root, t, span)
	ctx, cancel := context.WithTimeout(root, time.Duration(t.TimeoutSeconds)*time.Second)
	defer cancel()
	hbCtx, stopHB := context.WithCancel(context.WithoutCancel(root))
	hbDone := make(chan error, 1)
	go w.heartbeat(ctx, hbCtx, cancel, t, hbDone)
	result, runErr := w.runHandler(ctx, t)
	stopHB()
	leaseErr := <-hbDone
	if leaseErr != nil {
		if w.options.Observer != nil {
			w.options.Observer.RecordLeaseLost(root)
		}
		slog.WarnContext(root, "task lease lost", "task_id", t.TaskID, "attempt", t.Attempts)
		return
	}
	finishCtx, c := context.WithTimeout(context.WithoutCancel(root), 10*time.Second)
	defer c()
	if err := w.repo.Finish(finishCtx, t, result, runErr); err != nil {
		slog.ErrorContext(finishCtx, "task completion unconfirmed", "task_id", t.TaskID, "attempt", t.Attempts, "error_code", "commit_failed")
	} else if w.options.Observer != nil {
		outcome := "succeeded"
		if runErr != nil {
			outcome = "failed"
		}
		w.options.Observer.RecordExecution(finishCtx, outcome, time.Since(started))
	}
}
func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *Worker) intake(ctx, workCtx context.Context, wg *sync.WaitGroup, slots chan struct{}) {
	for ctx.Err() == nil {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		task, err := w.repo.Claim(ctx, w.options.ID, w.options.Lease)
		if err != nil || task == nil {
			<-slots
			if !w.claimBackoff(ctx, err) {
				return
			}
			continue
		}
		wg.Add(1)
		go func(t *model.TaskM) { defer wg.Done(); defer func() { <-slots }(); w.execute(workCtx, t) }(task)
	}
}

func (w *Worker) heartbeat(ctx, hbCtx context.Context, cancel context.CancelFunc, t *model.TaskM, hbDone chan<- error) {
	ticker := time.NewTicker(w.options.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			hbDone <- nil
			return
		case <-hbCtx.Done():
			hbDone <- nil
			return
		case <-ticker.C:
			check, c := context.WithTimeout(hbCtx, w.options.Heartbeat)
			canceled, err := w.repo.Heartbeat(check, t, w.options.Lease)
			c()
			if err != nil {
				cancel()
				hbDone <- err
				return
			}
			if canceled {
				cancel()
			}
		}
	}
}

func (w *Worker) runHandler(ctx context.Context, t *model.TaskM) (Result, error) {
	var result Result
	var runErr error
	func() {
		defer func() {
			if recover() != nil {
				runErr = errHandlerPanic
			}
		}()
		h, err := w.registry.Get(t.TaskType, t.ParamsVersion)
		if err != nil {
			runErr = err
			return
		}
		result, runErr = h.Execute(ctx, t.Params)
	}()
	if len(result.Content) > 10<<20 {
		runErr = Permanent(errResultTooLarge)
	}
	if ctx.Err() != nil {
		runErr = ctx.Err()
	}
	return result, runErr
}

func validateWorkerOptions(o WorkerOptions) error {
	if o.ID == "" || o.Concurrency < 1 || o.Concurrency > 128 {
		return errInvalidWorkerOptions
	}
	if o.Poll <= 0 || o.Drain <= 0 {
		return errInvalidWorkerOptions
	}
	if o.Lease <= 0 || o.Heartbeat <= 0 || o.Heartbeat*3 > o.Lease {
		return errInvalidWorkerOptions
	}
	return nil
}

func (w *Worker) claimBackoff(ctx context.Context, err error) bool {
	if err != nil {
		slog.WarnContext(ctx, "task claim unavailable", "error_code", "database_unavailable")
	}
	return wait(ctx, w.options.Poll)
}

func executionContext(root context.Context, t *model.TaskM, span trace.Span) context.Context {
	if t.RequestID != nil {
		root = contextx.WithRequestID(root, *t.RequestID)
	}
	if !span.SpanContext().IsValid() && t.TraceID != nil {
		root = contextx.WithTraceID(root, *t.TraceID)
	}
	return root
}

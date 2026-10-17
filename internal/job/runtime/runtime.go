// Package runtime assembles independently deployable Worker and Scheduler lifecycles.
package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver"
	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/metrics"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/pkg/dbschema"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
)

var errUnknownJobRole = errors.New("unknown job role")
var errJobRunnerStoppedUnexpectedly = errors.New("job runner stopped unexpectedly")

type runtime struct {
	cfg      *apiserver.Config
	core     store.IStore
	raw      *sql.DB
	role, id string
	metrics  *metrics.QueueMetrics
	worker   *job.Worker
	async    *apiserver.AsyncRuntime
	ready    atomic.Bool
}

// Run constructs dependencies before launching any goroutines and joins cleanup errors.
func Run(ctx context.Context, cfg *apiserver.Config, role string) error {
	if role != "worker" && role != "scheduler" {
		return errUnknownJobRole
	}
	db, err := cfg.PostgreSQLOptions.NewDB()
	if err != nil {
		return fmt.Errorf("open job database: %w", err)
	}
	raw, err := db.DB()
	if err != nil {
		return err
	}
	r := &runtime{cfg: cfg, core: store.NewStore(db), raw: raw, role: role, id: role + "-" + uuid.NewString()}
	err = r.initialize(ctx)
	if err == nil {
		err = r.serve(ctx)
	}
	if r.async != nil {
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 12*time.Second)
		err = errors.Join(err, r.async.Shutdown(shutdown))
		cancel()
	}
	return errors.Join(err, raw.Close())
}
func (r *runtime) initialize(ctx context.Context) error {
	if err := dbschema.Check(ctx, r.raw); err != nil {
		return err
	}
	m, err := metrics.NewQueueMetrics(r.core.DB(ctx), otel.Meter("background.tasks"))
	if err != nil {
		return err
	}
	r.metrics = m
	if r.role != "worker" {
		return nil
	}
	service := taskbiz.New(r.core)
	registry, err := service.Registry()
	if err != nil {
		return err
	}
	opts := r.cfg.TaskOptions
	if opts.Enabled {
		r.worker, err = job.NewWorker(store.NewTaskStore(r.core), registry, job.WorkerOptions{Observer: m, ID: r.id, Concurrency: opts.Concurrency, Poll: opts.PollInterval, Lease: opts.LeaseDuration, Heartbeat: opts.HeartbeatInterval, Drain: opts.DrainTimeout})
		if err != nil {
			return err
		}
	}
	return r.initializeMessaging(service)
}
func (r *runtime) healthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) {
		probe, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := r.check(probe); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
func (r *runtime) check(ctx context.Context) error {
	if !r.ready.Load() {
		return errJobRunnerStoppedUnexpectedly
	}
	if err := r.raw.PingContext(ctx); err != nil {
		return err
	}
	if r.async != nil {
		if err := r.async.Check(ctx); err != nil {
			return err
		}
		return apiserver.CheckKafka(ctx, r.cfg.KafkaOptions)
	}
	return nil
}
func (r *runtime) serve(ctx context.Context) error {
	addr := r.cfg.TaskOptions.WorkerAddr
	if r.role == "scheduler" {
		addr = r.cfg.TaskOptions.SchedulerAddr
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("background health listener: %w", err)
	}
	server := &http.Server{Handler: r.healthMux(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.ready.Store(true)
	results := make(chan error, 4)
	runners := 0
	launch := func(run func(context.Context) error) { runners++; go func() { results <- run(runCtx) }() }
	launch(func(context.Context) error { return server.Serve(listener) })
	launch(r.heartbeat)
	r.launchRole(runCtx, launch)
	remaining, runErr := awaitStop(ctx, results, runners)
	r.ready.Store(false)
	cancel()
	shutdown, stop := context.WithTimeout(context.WithoutCancel(ctx), r.cfg.TaskOptions.DrainTimeout+12*time.Second)
	defer stop()
	runErr = errors.Join(runErr, server.Shutdown(shutdown))
	if r.async != nil {
		runErr = errors.Join(runErr, r.async.Shutdown(shutdown))
	}
	return errors.Join(runErr, drainRunners(shutdown, results, remaining))
}
func (r *runtime) launchRole(ctx context.Context, launch func(func(context.Context) error)) {
	if r.role == "scheduler" {
		launch(r.schedule)
		return
	}
	if r.worker != nil {
		launch(r.worker.Run)
	}
	if r.async != nil {
		r.async.Start(ctx)
		launch(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return nil
			case <-r.async.Done():
				return r.async.Err()
			}
		})
	}
}
func awaitStop(ctx context.Context, results <-chan error, runners int) (int, error) {
	select {
	case <-ctx.Done():
		return runners, nil
	case err := <-results:
		if err == nil {
			return runners - 1, errJobRunnerStoppedUnexpectedly
		}
		return runners - 1, runnerError(err)
	}
}
func runnerError(err error) error {
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
func drainRunners(ctx context.Context, results <-chan error, runners int) error {
	var result error
	for range runners {
		select {
		case err := <-results:
			result = errors.Join(result, runnerError(err))
		case <-ctx.Done():
			return errors.Join(result, ctx.Err())
		}
	}
	return result
}
func (r *runtime) heartbeat(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	lastRefresh := time.Time{}
	for {
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := r.core.DB(probe).WithContext(probe).Exec("INSERT INTO runtime_heartbeat(instance_id,role) VALUES(?,?) ON CONFLICT(instance_id) DO UPDATE SET updated_at=clock_timestamp()", r.id, r.role).Error
		if err == nil {
			err = r.refreshMetrics(probe, &lastRefresh)
		}
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "runtime heartbeat unavailable", "role", r.role)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (r *runtime) schedule(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.TaskOptions.PollInterval)
	defer ticker.Stop()
	plans := store.NewScheduleStore(r.core)
	lastCleanup := time.Time{}
	for {
		probe, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := plans.Tick(probe)
		if err == nil {
			err = r.cleanup(probe, &lastCleanup)
		}
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "schedule tick failed", "error_code", "database_unavailable")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *runtime) initializeMessaging(service *taskbiz.Service) error {
	var err error
	if r.cfg.ConsumerOptions.Enabled || r.cfg.OutboxOptions.Enabled {
		router, e := service.EventRouter()
		if e != nil {
			return e
		}
		r.async, err = apiserver.NewAsyncRuntimeWithRouter(r.cfg, r.core, router)
	}
	return err
}

func (r *runtime) refreshMetrics(ctx context.Context, last *time.Time) error {
	if time.Since(*last) < 15*time.Second {
		return nil
	}
	if err := r.metrics.Refresh(ctx); err != nil {
		return err
	}
	*last = time.Now()
	return nil
}
func (r *runtime) cleanup(ctx context.Context, last *time.Time) error {
	if time.Since(*last) < time.Hour {
		return nil
	}
	if err := store.Cleanup(ctx, r.core); err != nil {
		return err
	}
	*last = time.Now()
	return nil
}

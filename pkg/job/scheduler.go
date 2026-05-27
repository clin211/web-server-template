package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

const (
	schedulerLockPrefix = "job:scheduler"

	// TriggerTypeCron 表示任务由 Cron 定时器触发。
	TriggerTypeCron = "cron"
	// DispatchStatusPending 表示调度记录等待入队。
	DispatchStatusPending = "pending"
	// DispatchStatusEnqueued 表示调度记录已成功入队。
	DispatchStatusEnqueued = "enqueued"
	// DispatchStatusEnqueueFailed 表示调度记录入队失败。
	DispatchStatusEnqueueFailed = "enqueue_failed"
	// ProcessStatusPending 表示任务尚未开始处理。
	ProcessStatusPending = "pending"
)

// Scheduler 管理系统定时任务和客户端定时任务的注册、同步与触发。
type Scheduler struct {
	enabled bool
	cron    *cron.Cron
	parser  cron.Parser
	lockTTL time.Duration

	registry  *Registry
	producer  Producer
	lock      Locker
	metrics   *Metrics
	taskStore SchedulerTaskStore
	options   *genericoptions.JobOptions

	mu      sync.RWMutex
	entries map[string]schedulerEntry
	cancel  context.CancelFunc
}

type schedulerEntry struct {
	id          cron.EntryID
	source      string
	fingerprint string
}

// SystemTask 表示调度器可注册的定时任务配置。
type SystemTask struct {
	// Name 是定时任务的唯一名称。
	Name string
	// CronExpr 是五段式 Cron 表达式。
	CronExpr string
	// TaskType 是要投递的异步任务类型。
	TaskType string
	// Queue 是任务投递目标队列。
	Queue string
	// Payload 是任务入队时携带的 JSON 负载。
	Payload map[string]any
	// Enabled 表示客户端定时任务是否启用。
	Enabled bool
	// Timezone 是该任务独立使用的时区。
	Timezone string
	// UserID 是客户端定时任务所属用户 ID。
	UserID string
	// UpdatedAt 是任务配置最后更新时间。
	UpdatedAt time.Time
}

// SchedulerExecution 表示一次定时任务调度执行记录。
type SchedulerExecution struct {
	// ExecutionID 是调度执行记录 ID。
	ExecutionID string
	// DispatchStatus 是任务入队阶段状态。
	DispatchStatus string
	// ProcessStatus 是任务处理阶段状态。
	ProcessStatus string
	// ScheduledTaskID 是对应的定时任务 ID。
	ScheduledTaskID string
}

// SchedulerTaskStore 定义客户端定时任务和执行记录的存储接口。
type SchedulerTaskStore interface {
	// ListEnabledTasks 返回当前启用的客户端定时任务。
	ListEnabledTasks(context.Context) ([]SystemTask, error)
	// CreateExecutionIfAbsent 创建指定调度时间的执行记录，已存在时返回 created=false。
	CreateExecutionIfAbsent(context.Context, SystemTask, time.Time) (*SchedulerExecution, bool, error)
	// MarkExecutionEnqueued 标记执行记录已成功入队。
	MarkExecutionEnqueued(context.Context, string, string, time.Time) error
	// MarkExecutionEnqueueFailed 标记执行记录入队失败。
	MarkExecutionEnqueueFailed(context.Context, string, error) error
	// UpdateTaskScheduleState 更新定时任务最近调度状态。
	UpdateTaskScheduleState(context.Context, SystemTask, time.Time, *time.Time, string, error) error
}

// NewScheduler 创建定时任务调度器。
func NewScheduler(registry *Registry, producer Producer, lock Locker, metrics *Metrics, taskStore SchedulerTaskStore, opts *genericoptions.JobOptions) (*Scheduler, error) {
	if opts == nil {
		opts = genericoptions.NewJobOptions()
	}

	location, err := time.LoadLocation(opts.Scheduler.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load scheduler timezone %q: %w", opts.Scheduler.Timezone, err)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	return &Scheduler{
		enabled:   opts.Scheduler.Enabled,
		cron:      cron.New(cron.WithParser(parser), cron.WithLocation(location)),
		parser:    parser,
		lockTTL:   opts.Scheduler.LockTTL,
		registry:  registry,
		producer:  producer,
		lock:      lock,
		metrics:   metrics,
		taskStore: taskStore,
		options:   opts,
		entries:   map[string]schedulerEntry{},
	}, nil
}

// Start 注册任务并启动底层 Cron 调度器。
func (s *Scheduler) Start(ctx context.Context) error {
	if s == nil || !s.enabled {
		return nil
	}
	if s.registry == nil {
		return fmt.Errorf("job registry is not initialized")
	}
	if s.producer == nil {
		return fmt.Errorf("job scheduler producer is not initialized")
	}
	if s.lock == nil {
		return fmt.Errorf("job scheduler lock is not initialized")
	}

	for _, cfg := range s.options.SystemTasks {
		if err := s.RegisterSystemTask(ctx, SystemTask{
			Name:     cfg.Name,
			CronExpr: cfg.CronExpr,
			TaskType: cfg.TaskType,
			Queue:    cfg.Queue,
			Payload:  cfg.Payload,
		}); err != nil {
			return err
		}
	}

	if s.clientTaskEnabled() {
		if err := s.loadClientTasks(ctx); err != nil {
			return err
		}
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.cron.Start()
	if s.clientTaskEnabled() {
		s.startReconcile(runCtx)
	}
	slog.InfoContext(ctx, "Job scheduler started", "systemTaskCount", len(s.options.SystemTasks))
	return nil
}

// Stop 停止底层 Cron 调度器并等待正在运行的任务退出。
func (s *Scheduler) Stop(ctx context.Context) {
	if s == nil || !s.enabled || s.cron == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	slog.InfoContext(ctx, "Stopping job scheduler")
	stopped := s.cron.Stop()
	select {
	case <-stopped.Done():
	case <-ctx.Done():
	}
}

// RegisterSystemTask 注册由系统配置驱动的定时任务。
func (s *Scheduler) RegisterSystemTask(ctx context.Context, task SystemTask) error {
	return s.registerTask(ctx, task, "system")
}

// RegisterClientTask 注册由客户端 API 管理的定时任务。
func (s *Scheduler) RegisterClientTask(ctx context.Context, task SystemTask) error {
	task.Enabled = true
	return s.registerTask(ctx, task, "client")
}

func (s *Scheduler) registerTask(ctx context.Context, task SystemTask, source string) error {
	if task.Name == "" {
		return fmt.Errorf("scheduled task name must not be empty")
	}
	if task.CronExpr == "" {
		return fmt.Errorf("scheduled task %q cron expr must not be empty", task.Name)
	}
	if task.TaskType == "" {
		return fmt.Errorf("scheduled task %q task type must not be empty", task.Name)
	}
	if source == "client" && !task.Enabled {
		s.UnregisterTask(ctx, task.Name)
		return nil
	}

	payload, err := marshalTaskPayload(task.Payload)
	if err != nil {
		return fmt.Errorf("marshal scheduled task %q payload: %w", task.Name, err)
	}
	def, queue, err := s.registry.ValidateEnqueue(ctx, task.TaskType, payload, task.Queue)
	if err != nil {
		return err
	}
	if source == "client" && def.Visibility != VisibilityPublic {
		return fmt.Errorf("job task type %q is not public", task.TaskType)
	}
	task.Queue = queue

	cronSpec := s.cronSpec(task)
	if _, err := s.parser.Parse(cronSpec); err != nil {
		return fmt.Errorf("parse scheduled task %q cron expr: %w", task.Name, err)
	}

	fingerprint := taskFingerprint(task, payload)

	var removedSource string
	s.mu.Lock()
	old, ok := s.entries[task.Name]
	if ok && old.source == source && old.fingerprint == fingerprint {
		s.mu.Unlock()
		return nil
	}

	entryID, err := s.cron.AddFunc(cronSpec, func() {
		s.dispatch(context.WithoutCancel(ctx), task, source)
	})
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("register scheduled task %q: %w", task.Name, err)
	}

	if ok {
		s.cron.Remove(old.id)
		removedSource = old.source
	}
	s.entries[task.Name] = schedulerEntry{id: entryID, source: source, fingerprint: fingerprint}
	s.mu.Unlock()

	if removedSource != "" {
		s.metrics.AddRegisteredTasks(ctx, removedSource, -1)
	}
	s.metrics.AddRegisteredTasks(ctx, source, 1)
	slog.InfoContext(ctx, "Registered scheduled task", "source", source, "name", task.Name, "cronExpr", task.CronExpr, "taskType", task.TaskType)
	return nil
}

// UnregisterTask 从调度器中移除指定名称的任务。
func (s *Scheduler) UnregisterTask(ctx context.Context, name string) {
	if s == nil || name == "" {
		return
	}

	s.mu.Lock()
	entry, ok := s.entries[name]
	if ok {
		delete(s.entries, name)
		s.cron.Remove(entry.id)
	}
	s.mu.Unlock()

	if ok {
		s.metrics.AddRegisteredTasks(ctx, entry.source, -1)
	}
}

func (s *Scheduler) dispatch(ctx context.Context, task SystemTask, source string) {
	scheduledAt := time.Now().UTC().Truncate(time.Minute)
	lockKey := fmt.Sprintf("%s:%s:%d", schedulerLockPrefix, task.Name, scheduledAt.Unix())
	token, err := s.lock.Acquire(ctx, lockKey, s.lockTTL)
	if err != nil {
		if errors.Is(err, ErrLockNotAcquired) {
			s.metrics.RecordSchedulerTick(ctx, task.TaskType, "skipped")
			return
		}
		s.metrics.RecordSchedulerTick(ctx, task.TaskType, "lock_failed")
		slog.ErrorContext(ctx, "Failed to acquire scheduler lock", "task", task.Name, "error", err)
		return
	}
	defer func() {
		if err := s.lock.Release(ctx, lockKey, token); err != nil && !errors.Is(err, ErrLockNotHeld) {
			slog.WarnContext(ctx, "Failed to release scheduler lock", "task", task.Name, "error", err)
		}
	}()

	if source == "client" && s.taskStore != nil {
		s.dispatchClientTask(ctx, task, scheduledAt)
		return
	}
	s.dispatchSystemTask(ctx, task, scheduledAt)
}

func (s *Scheduler) dispatchSystemTask(ctx context.Context, task SystemTask, scheduledAt time.Time) {
	payload, err := marshalTaskPayload(task.Payload)
	if err != nil {
		s.metrics.RecordSchedulerTick(ctx, task.TaskType, "payload_failed")
		slog.ErrorContext(ctx, "Failed to marshal system task payload", "task", task.Name, "error", err)
		return
	}

	result, err := s.producer.Enqueue(ctx, EnqueueRequest{
		TaskType:        task.TaskType,
		Payload:         payload,
		Queue:           task.Queue,
		ExecutionID:     fmt.Sprintf("%s:%d", task.Name, scheduledAt.Unix()),
		ScheduledTaskID: task.Name,
		UniqueTTL:       s.lockTTL,
		Metadata: map[string]string{
			"triggerType": TriggerTypeCron,
			"scheduledAt": scheduledAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		s.metrics.RecordSchedulerTick(ctx, task.TaskType, "enqueue_failed")
		slog.ErrorContext(ctx, "Failed to enqueue system scheduled task", "task", task.Name, "error", err)
		return
	}

	s.metrics.RecordSchedulerTick(ctx, task.TaskType, "enqueued")
	slog.InfoContext(ctx, "System scheduled task enqueued", "task", task.Name, "taskType", task.TaskType, "queue", result.Queue, "asynqTaskID", result.TaskID)
}

func (s *Scheduler) dispatchClientTask(ctx context.Context, task SystemTask, scheduledAt time.Time) {
	execution, created, err := s.taskStore.CreateExecutionIfAbsent(ctx, task, scheduledAt)
	if err != nil {
		s.recordClientDispatchFailure(ctx, task, scheduledAt, "execution_failed", "create scheduled task execution", err, "")
		return
	}
	if !created {
		s.metrics.RecordSchedulerTick(ctx, task.TaskType, "skipped")
		slog.InfoContext(ctx, "Scheduled task execution already exists", "scheduledTaskID", task.Name, "scheduledAt", scheduledAt, "executionID", execution.ExecutionID)
		return
	}

	payload, err := marshalTaskPayload(task.Payload)
	if err != nil {
		_ = s.taskStore.MarkExecutionEnqueueFailed(ctx, execution.ExecutionID, err)
		s.recordClientDispatchFailure(ctx, task, scheduledAt, "payload_failed", "marshal scheduled task payload", err, execution.ExecutionID)
		return
	}

	result, err := s.producer.Enqueue(ctx, EnqueueRequest{
		TaskType:        task.TaskType,
		Payload:         payload,
		Queue:           task.Queue,
		ExecutionID:     execution.ExecutionID,
		ScheduledTaskID: task.Name,
		UniqueTTL:       s.lockTTL,
		Metadata: map[string]string{
			"triggerType": TriggerTypeCron,
			"scheduledAt": scheduledAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		_ = s.taskStore.MarkExecutionEnqueueFailed(ctx, execution.ExecutionID, err)
		s.recordClientDispatchFailure(ctx, task, scheduledAt, "enqueue_failed", "enqueue client scheduled task", err, execution.ExecutionID)
		return
	}

	enqueuedAt := time.Now().UTC()
	if err := s.taskStore.MarkExecutionEnqueued(ctx, execution.ExecutionID, result.TaskID, enqueuedAt); err != nil {
		slog.WarnContext(ctx, "Failed to mark scheduled task execution enqueued", "executionID", execution.ExecutionID, "error", err)
	}
	if err := s.taskStore.UpdateTaskScheduleState(ctx, task, scheduledAt, s.nextRunTime(task), execution.ExecutionID, nil); err != nil {
		slog.WarnContext(ctx, "Failed to update scheduled task dispatch state", "scheduledTaskID", task.Name, "error", err)
	}
	s.metrics.RecordSchedulerTick(ctx, task.TaskType, "enqueued")
	slog.InfoContext(ctx, "Client scheduled task enqueued", "scheduledTaskID", task.Name, "taskType", task.TaskType, "queue", result.Queue, "executionID", execution.ExecutionID, "asynqTaskID", result.TaskID)
}

func (s *Scheduler) recordClientDispatchFailure(ctx context.Context, task SystemTask, scheduledAt time.Time, metricResult string, message string, err error, executionID string) {
	s.metrics.RecordSchedulerTick(ctx, task.TaskType, metricResult)
	if updateErr := s.taskStore.UpdateTaskScheduleState(ctx, task, scheduledAt, s.nextRunTime(task), executionID, err); updateErr != nil {
		slog.WarnContext(ctx, "Failed to update scheduled task dispatch failure", "scheduledTaskID", task.Name, "error", updateErr)
	}
	slog.ErrorContext(ctx, message, "scheduledTaskID", task.Name, "taskType", task.TaskType, "error", err)
}

func (s *Scheduler) loadClientTasks(ctx context.Context) error {
	if s.taskStore == nil {
		slog.WarnContext(ctx, "Client scheduled task store is not initialized")
		return nil
	}
	tasks, err := s.taskStore.ListEnabledTasks(ctx)
	if err != nil {
		return fmt.Errorf("list enabled client scheduled tasks: %w", err)
	}

	failed := 0
	for _, task := range tasks {
		task.Enabled = true
		if err := s.registerTask(ctx, task, "client"); err != nil {
			failed++
			slog.ErrorContext(ctx, "Failed to register client scheduled task", "scheduledTaskID", task.Name, "error", err)
		}
	}
	slog.InfoContext(ctx, "Loaded client scheduled tasks", "count", len(tasks), "failed", failed)
	return nil
}

func (s *Scheduler) startReconcile(ctx context.Context) {
	interval := s.options.Scheduler.ReconcileInterval
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.reconcile(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) reconcile(ctx context.Context) {
	if s.taskStore == nil {
		return
	}
	tasks, err := s.taskStore.ListEnabledTasks(ctx)
	if err != nil {
		s.metrics.RecordSchedulerReconcile(ctx, "failed")
		slog.ErrorContext(ctx, "Failed to list enabled client scheduled tasks", "error", err)
		return
	}

	seen := make(map[string]struct{}, len(tasks))
	failed := 0
	for _, task := range tasks {
		task.Enabled = true
		seen[task.Name] = struct{}{}
		if err := s.registerTask(ctx, task, "client"); err != nil {
			failed++
			slog.ErrorContext(ctx, "Failed to reconcile client scheduled task", "scheduledTaskID", task.Name, "error", err)
		}
	}

	for _, name := range s.clientEntryNames() {
		if _, ok := seen[name]; !ok {
			s.UnregisterTask(ctx, name)
		}
	}

	if failed > 0 {
		s.metrics.RecordSchedulerReconcile(ctx, "partial_failed")
		return
	}
	s.metrics.RecordSchedulerReconcile(ctx, "succeeded")
}

func (s *Scheduler) clientEntryNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range s.entries {
		if entry.source == "client" {
			names = append(names, name)
		}
	}
	return names
}

func (s *Scheduler) clientTaskEnabled() bool {
	return s != nil && s.options != nil && s.options.ClientTask.Enabled && s.taskStore != nil
}

func (s *Scheduler) cronSpec(task SystemTask) string {
	if task.Timezone == "" || task.Timezone == s.options.Scheduler.Timezone {
		return task.CronExpr
	}
	return fmt.Sprintf("CRON_TZ=%s %s", task.Timezone, task.CronExpr)
}

func (s *Scheduler) nextRunTime(task SystemTask) *time.Time {
	schedule, err := s.parser.Parse(s.cronSpec(task))
	if err != nil {
		return nil
	}
	next := schedule.Next(time.Now())
	return &next
}

func marshalTaskPayload(payload map[string]any) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return json.RawMessage("{}"), nil
	}
	return json.RawMessage(data), nil
}

func taskFingerprint(task SystemTask, payload json.RawMessage) string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s", task.CronExpr, task.TaskType, task.Queue, task.Timezone, string(payload))
}

// ValidateCronExpr 校验五段式 Cron 表达式是否合法。
func ValidateCronExpr(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(expr); err != nil {
		return fmt.Errorf("parse cron expr: %w", err)
	}
	return nil
}

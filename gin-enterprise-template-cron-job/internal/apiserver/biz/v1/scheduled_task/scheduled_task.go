package scheduled_task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authz"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

const (
	// DispatchStatusPending means the task execution is waiting to be enqueued.
	DispatchStatusPending = "pending"
	// DispatchStatusEnqueued means the task execution has been enqueued.
	DispatchStatusEnqueued = "enqueued"
	// DispatchStatusEnqueueFailed means enqueueing the execution failed.
	DispatchStatusEnqueueFailed = "enqueue_failed"
	// ProcessStatusPending means the execution has not started processing.
	ProcessStatusPending = "pending"
	// TriggerTypeCron marks an execution triggered by cron scheduling.
	TriggerTypeCron = "cron"
	// TriggerTypeManual marks an execution triggered manually.
	TriggerTypeManual = "manual"
)

// TaskProducer enqueues job tasks to the queue.
type TaskProducer interface {
	Enqueue(context.Context, genericjob.EnqueueRequest) (*genericjob.EnqueueResult, error)
}

// ClientTaskScheduler registers and unregisters scheduled tasks with the scheduler.
type ClientTaskScheduler interface {
	RegisterClientTask(context.Context, genericjob.SystemTask) error
	UnregisterTask(context.Context, string)
}

// TaskRegistry provides task definitions and validation.
type TaskRegistry interface {
	Get(string) (genericjob.TaskDef, bool)
	ValidateEnqueue(context.Context, string, json.RawMessage, string) (genericjob.TaskDef, string, error)
}

// ScheduledTaskBiz defines the scheduled task business logic interface.
type ScheduledTaskBiz interface {
	Create(ctx context.Context, rq *v1.CreateScheduledTaskRequest) (*v1.CreateScheduledTaskResponse, error)
	Update(ctx context.Context, rq *v1.UpdateScheduledTaskRequest) (*v1.UpdateScheduledTaskResponse, error)
	Delete(ctx context.Context, rq *v1.DeleteScheduledTaskRequest) (*v1.DeleteScheduledTaskResponse, error)
	Get(ctx context.Context, rq *v1.GetScheduledTaskRequest) (*v1.GetScheduledTaskResponse, error)
	List(ctx context.Context, rq *v1.ListScheduledTasksRequest) (*v1.ListScheduledTasksResponse, error)
	Toggle(ctx context.Context, rq *v1.ToggleScheduledTaskRequest) (*v1.ToggleScheduledTaskResponse, error)
	Trigger(ctx context.Context, rq *v1.TriggerScheduledTaskRequest) (*v1.TriggerScheduledTaskResponse, error)
	ListExecutions(ctx context.Context, rq *v1.ListScheduledTaskExecutionsRequest) (*v1.ListScheduledTaskExecutionsResponse, error)
}

// scheduledTaskBiz implements ScheduledTaskBiz.
type scheduledTaskBiz struct {
	store     store.IStore
	authz     *authz.Authz
	producer  TaskProducer
	scheduler ClientTaskScheduler
	registry  TaskRegistry
	options   *genericoptions.JobOptions
}

var (
	now                  = time.Now
	_   ScheduledTaskBiz = (*scheduledTaskBiz)(nil)
)

// New creates a new scheduledTaskBiz.
func New(store store.IStore, authz *authz.Authz, producer TaskProducer, scheduler ClientTaskScheduler, registry TaskRegistry, options *genericoptions.JobOptions) *scheduledTaskBiz {
	return &scheduledTaskBiz{store: store, authz: authz, producer: producer, scheduler: scheduler, registry: registry, options: options}
}

// minInterval returns the minimum allowed interval between task executions.
func (b *scheduledTaskBiz) minInterval() time.Duration {
	if b.options == nil || b.options.Scheduler.MinInterval <= 0 {
		return time.Minute
	}
	return b.options.Scheduler.MinInterval
}

// maxTasksPerUser returns the maximum tasks allowed per user.
func (b *scheduledTaskBiz) maxTasksPerUser() int {
	if b.options == nil || b.options.ClientTask.MaxTasksPerUser <= 0 {
		return 100
	}
	return b.options.ClientTask.MaxTasksPerUser
}

// maxPayloadBytes returns the maximum payload size in bytes.
func (b *scheduledTaskBiz) maxPayloadBytes() int {
	if b.options == nil || b.options.ClientTask.MaxPayloadBytes <= 0 {
		return 8192
	}
	return b.options.ClientTask.MaxPayloadBytes
}

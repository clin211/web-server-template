package scheduled_task

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/datatypes"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// checkTaskDefinition validates task type, payload, and queue.
func (b *scheduledTaskBiz) checkTaskDefinition(ctx context.Context, taskType string, payload json.RawMessage, queue string) (string, error) {
	def, ok := b.registry.Get(taskType)
	if !ok || def.Visibility != "public" {
		return "", errno.ErrScheduledTaskTaskTypeNotSupported
	}
	_, queue, err := b.registry.ValidateEnqueue(ctx, taskType, payload, queue)
	if err != nil {
		return "", errno.ErrScheduledTaskInvalidPayload.WithMessage("%s", err.Error())
	}
	if b.options != nil && len(b.options.ClientTask.AllowedQueues) > 0 && !slices.Contains(b.options.ClientTask.AllowedQueues, queue) {
		return "", errno.ErrScheduledTaskQueueNotAllowed
	}
	return queue, nil
}

// validateQuota checks if the user has reached the max task limit.
func (b *scheduledTaskBiz) validateQuota(ctx context.Context) error {
	if contextx.Username(ctx) == known.AdminUsername {
		return nil
	}
	count, _, err := b.store.ScheduledTask().List(ctx, where.F("user_id", contextx.UserID(ctx)))
	if err != nil {
		return fmt.Errorf("list scheduled tasks for quota: %w", err)
	}
	if int(count) >= b.maxTasksPerUser() {
		return errno.ErrScheduledTaskQuotaExceeded
	}
	return nil
}

// nextRunTime computes the next execution time for a cron expression.
func nextRunTime(cronExpr string, timezone string, from time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone: %w", err)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return nil, errno.ErrScheduledTaskInvalidCronExpr.WithMessage("%s", err.Error())
	}
	next := schedule.Next(from.In(location))
	return &next, nil
}

// ensureMinInterval validates that cron expression has at least minInterval between runs.
func ensureMinInterval(cronExpr string, timezone string, minInterval time.Duration) error {
	next, err := nextRunTime(cronExpr, timezone, time.Now())
	if err != nil {
		return err
	}
	afterNext, err := nextRunTime(cronExpr, timezone, *next)
	if err != nil {
		return err
	}
	if afterNext.Sub(*next) < minInterval {
		return errno.ErrScheduledTaskInvalidCronExpr.WithMessage("cron interval is too short")
	}
	return nil
}

// newScheduledTaskModel creates a new scheduled task model from parameters.
func newScheduledTaskModel(ctx context.Context, name string, taskType string, payload []byte, cronExpr string, queue string, enabled bool, timezone string, nextRun *time.Time) *model.ScheduledTaskM {
	return &model.ScheduledTaskM{
		ScheduledTaskID: uuid.New().String(),
		Name:            name,
		TaskType:        taskType,
		Payload:         datatypes.JSON(payload),
		CronExpr:        cronExpr,
		Queue:           queue,
		Enabled:         enabled,
		Timezone:        timezone,
		UserID:          contextx.UserID(ctx),
		NextRunTime:     nextRun,
	}
}

// ownerWhere returns a filter for the user's own tasks only.
func (b *scheduledTaskBiz) ownerWhere(ctx context.Context) *where.Options {
	if contextx.Username(ctx) == known.AdminUsername {
		return where.NewWhere()
	}
	return where.F("user_id", contextx.UserID(ctx))
}

// registerSchedulerTask registers the task with the scheduler if enabled.
func registerSchedulerTask(ctx context.Context, scheduler ClientTaskScheduler, task *model.ScheduledTaskM) error {
	if scheduler == nil || task == nil || !task.Enabled {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal scheduled task payload: %w", err)
	}
	return scheduler.RegisterClientTask(ctx, genericjob.SystemTask{
		Name:      task.ScheduledTaskID,
		CronExpr:  task.CronExpr,
		TaskType:  task.TaskType,
		Queue:     task.Queue,
		Payload:   payload,
		Enabled:   task.Enabled,
		Timezone:  task.Timezone,
		UserID:    task.UserID,
		UpdatedAt: task.UpdatedAt,
	})
}

// checkPermission checks if the subject has permission for the action.
func (b *scheduledTaskBiz) checkPermission(ctx context.Context, sub, obj, act string) error {
	allowed, err := b.authz.Authorize(sub, obj, act)
	if err != nil {
		return fmt.Errorf("check permission %s %s %s: %w", sub, obj, act, err)
	}
	if !allowed {
		return errno.ErrScheduledTaskPermissionDenied
	}
	return nil
}

// canAccessTask checks if the current user can perform the action on the task.
func (b *scheduledTaskBiz) canAccessTask(ctx context.Context, task *model.ScheduledTaskM, act string) error {
	if contextx.Username(ctx) == known.AdminUsername {
		return nil
	}
	if task.UserID != contextx.UserID(ctx) {
		return errno.ErrScheduledTaskPermissionDenied
	}
	return b.checkPermission(ctx, contextx.Username(ctx), "/scheduled-tasks", act)
}

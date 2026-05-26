package scheduled_task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Update updates an existing scheduled task.
func (b *scheduledTaskBiz) Update(ctx context.Context, rq *v1.UpdateScheduledTaskRequest) (*v1.UpdateScheduledTaskResponse, error) {
	task, err := b.store.ScheduledTask().GetByScheduledTaskID(ctx, rq.GetScheduledTaskID())
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "update"); err != nil {
		return nil, err
	}
	cronChanged := false
	if rq.Name != nil {
		task.Name = rq.GetName()
	}
	if rq.Payload != nil {
		payload := conversion.StructToJSON(rq.GetPayload())
		if len(payload) > b.maxPayloadBytes() {
			return nil, errno.ErrScheduledTaskInvalidPayload.WithMessage("payload is too large")
		}
		task.Payload = payload
	}
	if rq.CronExpr != nil {
		if err := ensureMinInterval(rq.GetCronExpr(), task.Timezone, b.minInterval()); err != nil {
			return nil, err
		}
		task.CronExpr = rq.GetCronExpr()
		cronChanged = true
	}
	if rq.Queue != nil {
		task.Queue = rq.GetQueue()
	}
	if rq.Enabled != nil {
		task.Enabled = rq.GetEnabled()
	}
	if rq.Timezone != nil {
		task.Timezone = rq.GetTimezone()
		cronChanged = true
	}
	queue, err := b.checkTaskDefinition(ctx, task.TaskType, json.RawMessage(task.Payload), task.Queue)
	if err != nil {
		return nil, err
	}
	task.Queue = queue
	if cronChanged {
		nextRun, err := nextRunTime(task.CronExpr, task.Timezone, now())
		if err != nil {
			return nil, err
		}
		task.NextRunTime = nextRun
	}

	if err := b.store.ScheduledTask().Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update scheduled task: %w", err)
	}
	if b.scheduler != nil {
		b.scheduler.UnregisterTask(ctx, task.ScheduledTaskID)
	}
	if err := registerSchedulerTask(ctx, b.scheduler, task); err != nil {
		return nil, fmt.Errorf("register scheduled task: %w", err)
	}
	return &v1.UpdateScheduledTaskResponse{}, nil
}

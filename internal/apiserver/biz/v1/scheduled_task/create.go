package scheduled_task

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Create creates a new scheduled task.
func (b *scheduledTaskBiz) Create(ctx context.Context, rq *v1.CreateScheduledTaskRequest) (*v1.CreateScheduledTaskResponse, error) {
	if err := b.checkPermission(ctx, contextx.Username(ctx), "/scheduled-tasks", "create"); err != nil {
		return nil, err
	}
	if err := b.validateQuota(ctx); err != nil {
		return nil, err
	}
	payload := conversion.StructToJSON(rq.GetPayload())
	if len(payload) > b.maxPayloadBytes() {
		return nil, errno.ErrScheduledTaskInvalidPayload.WithMessage("payload is too large")
	}
	if err := ensureMinInterval(rq.GetCronExpr(), rq.GetTimezone(), b.minInterval()); err != nil {
		return nil, err
	}
	queue, err := b.checkTaskDefinition(ctx, rq.GetTaskType(), payload, rq.GetQueue())
	if err != nil {
		return nil, err
	}
	nextRun, err := nextRunTime(rq.GetCronExpr(), rq.GetTimezone(), now())
	if err != nil {
		return nil, err
	}
	task := newScheduledTaskModel(ctx, rq.GetName(), rq.GetTaskType(), payload, rq.GetCronExpr(), queue, rq.GetEnabled(), rq.GetTimezone(), nextRun)
	if err := b.store.ScheduledTask().Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create scheduled task: %w", err)
	}
	if err := registerSchedulerTask(ctx, b.scheduler, task); err != nil {
		return nil, fmt.Errorf("register scheduled task: %w", err)
	}
	return &v1.CreateScheduledTaskResponse{ScheduledTaskID: task.ScheduledTaskID}, nil
}

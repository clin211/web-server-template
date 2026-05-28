package scheduled_task

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Delete deletes a scheduled task.
func (b *scheduledTaskBiz) Delete(ctx context.Context, rq *v1.DeleteScheduledTaskRequest) (*v1.DeleteScheduledTaskResponse, error) {
	task, err := b.store.ScheduledTask().GetByScheduledTaskID(ctx, rq.GetScheduledTaskID())
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "delete"); err != nil {
		return nil, err
	}
	if err := b.store.ScheduledTask().Delete(ctx, b.ownerWhere(ctx).F("scheduled_task_id", rq.GetScheduledTaskID())); err != nil {
		return nil, fmt.Errorf("delete scheduled task: %w", err)
	}
	if b.scheduler != nil {
		b.scheduler.UnregisterTask(ctx, rq.GetScheduledTaskID())
	}
	return &v1.DeleteScheduledTaskResponse{}, nil
}

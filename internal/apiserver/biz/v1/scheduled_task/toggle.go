package scheduled_task

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Toggle enables or disables a scheduled task.
func (b *scheduledTaskBiz) Toggle(ctx context.Context, rq *v1.ToggleScheduledTaskRequest) (*v1.ToggleScheduledTaskResponse, error) {
	task, err := b.store.ScheduledTask().Get(ctx, where.F("scheduled_task_id", rq.GetScheduledTaskID()))
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "update"); err != nil {
		return nil, err
	}
	task.Enabled = rq.GetEnabled()
	nextRun, err := nextRunTime(task.CronExpr, task.Timezone, now())
	if err != nil {
		return nil, err
	}
	task.NextRunTime = nextRun
	if err := b.store.ScheduledTask().Update(ctx, task); err != nil {
		return nil, fmt.Errorf("toggle scheduled task: %w", err)
	}
	if b.scheduler != nil {
		b.scheduler.UnregisterTask(ctx, task.ScheduledTaskID)
	}
	if err := registerSchedulerTask(ctx, b.scheduler, task); err != nil {
		return nil, fmt.Errorf("register scheduled task: %w", err)
	}
	return &v1.ToggleScheduledTaskResponse{ScheduledTask: conversion.ScheduledTaskModelToScheduledTaskV1(task)}, nil
}

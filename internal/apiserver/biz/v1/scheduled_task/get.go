package scheduled_task

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Get returns a scheduled task by ID.
func (b *scheduledTaskBiz) Get(ctx context.Context, rq *v1.GetScheduledTaskRequest) (*v1.GetScheduledTaskResponse, error) {
	task, err := b.store.ScheduledTask().GetByScheduledTaskID(ctx, rq.GetScheduledTaskID())
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "get"); err != nil {
		return nil, err
	}
	return &v1.GetScheduledTaskResponse{ScheduledTask: conversion.ScheduledTaskModelToScheduledTaskV1(task)}, nil
}

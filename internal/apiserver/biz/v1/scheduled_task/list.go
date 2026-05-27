package scheduled_task

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// List returns a paginated list of scheduled tasks.
func (b *scheduledTaskBiz) List(ctx context.Context, rq *v1.ListScheduledTasksRequest) (*v1.ListScheduledTasksResponse, error) {
	if err := b.checkPermission(ctx, contextx.Username(ctx), "/scheduled-tasks", "list"); err != nil {
		return nil, err
	}
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())
	opts := b.ownerWhere(ctx).L(pageSize)
	if cursor := decodePageCursor(rq.GetPageToken()); cursor != nil {
		opts.Cursor = cursor
	}
	if rq.Enabled != nil {
		opts.F("enabled", rq.GetEnabled())
	}
	if rq.TaskType != nil {
		opts.F("task_type", rq.GetTaskType())
	}

	total, tasks, err := b.store.ScheduledTask().List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &v1.ListScheduledTasksResponse{
		TotalCount:     total,
		ScheduledTasks: conversion.ScheduledTaskModelListToScheduledTaskV1List(tasks),
		PageToken: pagination.NextPageToken(len(tasks), pageSize, func() int64 {
			return tasks[len(tasks)-1].ID
		}),
	}, nil
}

// decodePageCursor extracts the cursor ID from a page token string.
func decodePageCursor(pageToken string) *int64 {
	if pageToken == "" {
		return nil
	}
	decoded, err := pagination.DecodeCursor(pageToken)
	if err != nil {
		return nil
	}
	id, ok := decoded.GetInt64("id")
	if !ok {
		return nil
	}
	return &id
}

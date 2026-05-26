package scheduled_task

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// ListExecutions returns a paginated list of task execution records.
func (b *scheduledTaskBiz) ListExecutions(ctx context.Context, rq *v1.ListScheduledTaskExecutionsRequest) (*v1.ListScheduledTaskExecutionsResponse, error) {
	task, err := b.store.ScheduledTask().GetByScheduledTaskID(ctx, rq.GetScheduledTaskID())
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "execution:list"); err != nil {
		return nil, err
	}
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())
	opts := where.F("scheduled_task_id", rq.GetScheduledTaskID()).L(pageSize)
	if cursor := decodePageCursor(rq.GetPageToken()); cursor != nil {
		opts.Cursor = cursor
	}
	if rq.DispatchStatus != nil {
		opts.F("dispatch_status", rq.GetDispatchStatus())
	}
	if rq.ProcessStatus != nil {
		opts.F("process_status", rq.GetProcessStatus())
	}
	total, executions, err := b.store.ScheduledTaskExecution().List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &v1.ListScheduledTaskExecutionsResponse{
		TotalCount: total,
		Executions: conversion.ScheduledTaskExecutionModelListToExecutionV1List(executions),
		PageToken:  nextPageToken(len(executions), pageSize, func() int64 { return executions[len(executions)-1].ID }),
	}, nil
}

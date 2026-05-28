package scheduled_task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

// Trigger manually triggers a scheduled task execution.
func (b *scheduledTaskBiz) Trigger(ctx context.Context, rq *v1.TriggerScheduledTaskRequest) (*v1.TriggerScheduledTaskResponse, error) {
	task, err := b.store.ScheduledTask().GetByScheduledTaskID(ctx, rq.GetScheduledTaskID())
	if err != nil {
		return nil, errno.ErrScheduledTaskNotFound
	}
	if err := b.canAccessTask(ctx, task, "trigger"); err != nil {
		return nil, err
	}
	if _, err := b.checkTaskDefinition(ctx, task.TaskType, json.RawMessage(task.Payload), task.Queue); err != nil {
		return nil, err
	}

	triggeredAt := now().UTC()
	execution := &model.ScheduledTaskExecutionM{
		ExecutionID:     uuid.New().String(),
		ScheduledTaskID: task.ScheduledTaskID,
		UserID:          task.UserID,
		TriggerType:     TriggerTypeManual,
		ScheduledAt:     triggeredAt,
		DispatchStatus:  DispatchStatusPending,
		ProcessStatus:   ProcessStatusPending,
	}

	if err := b.store.ScheduledTaskExecution().Create(ctx, execution); err != nil {
		return nil, fmt.Errorf("create scheduled task execution: %w", err)
	}
	result, err := b.producer.Enqueue(ctx, genericjob.EnqueueRequest{
		TaskType:        task.TaskType,
		Payload:         json.RawMessage(task.Payload),
		Queue:           task.Queue,
		ExecutionID:     execution.ExecutionID,
		ScheduledTaskID: task.ScheduledTaskID,
		Metadata: map[string]string{
			"triggerType": TriggerTypeManual,
			"traceID":     contextx.TraceID(ctx),
		},
	})
	if err != nil {
		errorMsg := err.Error()
		execution.DispatchStatus = DispatchStatusEnqueueFailed
		execution.ErrorMsg = &errorMsg
		_ = b.store.ScheduledTaskExecution().Update(ctx, execution)
		return nil, errno.ErrScheduledTaskEnqueueFailed.WithMessage("%s", err.Error())
	}

	enqueuedAt := now().UTC()
	execution.EnqueuedAt = &enqueuedAt
	execution.DispatchStatus = DispatchStatusEnqueued
	execution.AsynqTaskID = &result.TaskID
	if err := b.store.ScheduledTaskExecution().Update(ctx, execution); err != nil {
		return nil, fmt.Errorf("update scheduled task execution: %w", err)
	}
	return &v1.TriggerScheduledTaskResponse{ExecutionID: execution.ExecutionID}, nil
}

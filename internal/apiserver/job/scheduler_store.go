package job

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

// schedulerTaskStore adapts the scheduled task store to genericjob.SchedulerTaskStore.
type schedulerTaskStore struct {
	store store.IStore
}

// NewSchedulerTaskStore creates the scheduled task store adapter for pkg/job.
func NewSchedulerTaskStore(store store.IStore) genericjob.SchedulerTaskStore {
	return &schedulerTaskStore{store: store}
}

// ListEnabledTasks returns all enabled scheduled tasks for the scheduler.
func (s *schedulerTaskStore) ListEnabledTasks(ctx context.Context) ([]genericjob.SystemTask, error) {
	tasks, err := s.store.ScheduledTask().ListEnabledTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled scheduled tasks: %w", err)
	}

	result := make([]genericjob.SystemTask, 0, len(tasks))
	for _, task := range tasks {
		value, err := modelToSystemTask(task)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to decode scheduled task", "scheduledTaskID", task.ScheduledTaskID, "error", err)
			continue
		}
		result = append(result, value)
	}
	return result, nil
}

// CreateExecutionIfAbsent creates a new execution record if one doesn't already exist for the scheduled time.
func (s *schedulerTaskStore) CreateExecutionIfAbsent(ctx context.Context, task genericjob.SystemTask, scheduledAt time.Time) (*genericjob.SchedulerExecution, bool, error) {
	execution := &model.ScheduledTaskExecutionM{
		ExecutionID:     uuid.New().String(),
		ScheduledTaskID: task.Name,
		UserID:          task.UserID,
		TriggerType:     genericjob.TriggerTypeCron,
		ScheduledAt:     scheduledAt,
		DispatchStatus:  genericjob.DispatchStatusPending,
		ProcessStatus:   genericjob.ProcessStatusPending,
	}
	createdExecution, created, err := s.store.ScheduledTaskExecution().CreateExecutionIfAbsent(ctx, execution)
	if err != nil {
		return nil, false, fmt.Errorf("create scheduled task execution if absent: %w", err)
	}
	return &genericjob.SchedulerExecution{
		ExecutionID:     createdExecution.ExecutionID,
		DispatchStatus:  createdExecution.DispatchStatus,
		ProcessStatus:   createdExecution.ProcessStatus,
		ScheduledTaskID: createdExecution.ScheduledTaskID,
	}, created, nil
}

// MarkExecutionEnqueued records that an execution was successfully enqueued.
func (s *schedulerTaskStore) MarkExecutionEnqueued(ctx context.Context, executionID string, asynqTaskID string, enqueuedAt time.Time) error {
	result := s.store.DB(ctx).Model(&model.ScheduledTaskExecutionM{}).
		Where("execution_id = ?", executionID).
		Updates(map[string]any{
			"dispatch_status": genericjob.DispatchStatusEnqueued,
			"enqueued_at":     enqueuedAt,
			"asynq_task_id":   asynqTaskID,
			"error_msg":       nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("scheduled task execution %q not found", executionID)
	}
	return nil
}

// MarkExecutionEnqueueFailed records that enqueueing failed for an execution.
func (s *schedulerTaskStore) MarkExecutionEnqueueFailed(ctx context.Context, executionID string, err error) error {
	errorMsg := errorString(err)
	result := s.store.DB(ctx).Model(&model.ScheduledTaskExecutionM{}).
		Where("execution_id = ?", executionID).
		Updates(map[string]any{
			"dispatch_status": genericjob.DispatchStatusEnqueueFailed,
			"error_msg":       errorMsg,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("scheduled task execution %q not found", executionID)
	}
	return nil
}

// UpdateTaskScheduleState updates the task's next run time and last execution info.
func (s *schedulerTaskStore) UpdateTaskScheduleState(ctx context.Context, task genericjob.SystemTask, scheduledAt time.Time, nextRunTime *time.Time, executionID string, err error) error {
	values := map[string]any{
		"last_scheduled_at": scheduledAt,
		"next_run_time":     nextRunTime,
		"last_error":        nil,
	}
	if executionID != "" {
		values["last_execution_id"] = executionID
	}
	if err != nil {
		values["last_error"] = errorString(err)
	}

	result := s.store.DB(ctx).Model(&model.ScheduledTaskM{}).
		Where("scheduled_task_id = ?", task.Name).
		Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("scheduled task %q not found", task.Name)
	}
	return nil
}

// modelToSystemTask converts a database model to a genericjob.SystemTask.
func modelToSystemTask(task *model.ScheduledTaskM) (genericjob.SystemTask, error) {
	if task == nil {
		return genericjob.SystemTask{}, fmt.Errorf("scheduled task is nil")
	}
	payload, err := jsonPayload(task.Payload)
	if err != nil {
		return genericjob.SystemTask{}, fmt.Errorf("decode scheduled task %q payload: %w", task.ScheduledTaskID, err)
	}
	return genericjob.SystemTask{
		Name:      task.ScheduledTaskID,
		CronExpr:  task.CronExpr,
		TaskType:  task.TaskType,
		Queue:     task.Queue,
		Payload:   payload,
		Enabled:   task.Enabled,
		Timezone:  task.Timezone,
		UserID:    task.UserID,
		UpdatedAt: task.UpdatedAt,
	}, nil
}

// jsonPayload converts database JSON column to map.
func jsonPayload(payload string) (map[string]any, error) {
	if len(payload) == 0 {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return nil, err
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 512 {
		return msg[:512]
	}
	return msg
}

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

// ListEnabledTasks 返回所有 enabled=true 的客户端定时任务。
//
// 并发安全说明：
// 1. 使用直接的数据库查询，不依赖 Store 接口的 List 方法
// 2. 返回所有记录而不受分页限制，确保调度器能加载全部启用的任务
// 3. 调度器通过 reconcile 周期与数据库状态保持同步
func (s *schedulerTaskStore) ListEnabledTasks(ctx context.Context) ([]genericjob.SystemTask, error) {
	var tasks []*model.ScheduledTaskM
	if err := s.store.DB(ctx).
		Where("enabled = ?", true).
		Find(&tasks).Error; err != nil {
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

// CreateExecutionIfAbsent 创建一个新的执行记录（如果不存在）。
//
// 并发安全保证：
// 1. 使用 PostgreSQL 的唯一约束 (scheduled_task_id, scheduled_at) 作为幂等保证
// 2. 即使多个 scheduler 实例同时调用，也只有一个会成功创建记录
// 3. 另一个调用会收到 created=false，返回已存在的记录
//
// 注意：返回的 execution.ExecutionID 可能是新创建的或已存在的，调用方不应假设是新创建的。
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

// MarkExecutionEnqueued 标记执行记录已成功入队。
//
// 并发安全说明：
// 1. 使用 executionID 作为更新条件，确保只更新对应的记录
// 2. RowsAffected == 0 表示记录不存在，返回明确错误
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

// MarkExecutionEnqueueFailed 标记执行记录入队失败。
//
// 并发安全说明：
// 1. 使用 executionID 作为更新条件，确保只更新对应的记录
// 2. 错误信息会被截断到 512 字符以避免存储过大
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

// UpdateTaskScheduleState 更新任务的调度状态和最近执行信息。
//
// 并发安全说明：
// 1. 使用任务名称作为更新条件
// 2. nextRunTime、lastScheduledAt、lastError 都会被更新
// 3. 如果执行失败，lastError 会记录失败原因
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

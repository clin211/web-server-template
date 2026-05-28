package job

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

const (
	processStatusRunning   = "running"
	processStatusRetrying  = "retrying"
	processStatusSucceeded = "succeeded"
	processStatusFailed    = "failed"
	processStatusDead      = "dead"
)

// executionRecorder tracks job execution state in the database.
type executionRecorder struct {
	store store.IStore
}

// NewExecutionRecorder creates a database-backed job execution recorder.
func NewExecutionRecorder(store store.IStore) genericjob.ExecutionRecorder {
	return &executionRecorder{store: store}
}

// MarkRunning marks an execution as running.
func (r *executionRecorder) MarkRunning(ctx context.Context, task *genericjob.Task) error {
	if r.skip(task) {
		return nil
	}

	now := time.Now().UTC()
	return r.update(ctx, task.ExecutionID, map[string]any{
		"process_status": processStatusRunning,
		"started_at":     now,
	})
}

// MarkSucceeded marks an execution as succeeded.
func (r *executionRecorder) MarkSucceeded(ctx context.Context, task *genericjob.Task, duration time.Duration) error {
	if r.skip(task) {
		return nil
	}

	now := time.Now().UTC()
	return r.update(ctx, task.ExecutionID, map[string]any{
		"process_status": processStatusSucceeded,
		"finished_at":    now,
		"duration_ms":    duration.Milliseconds(),
		"error_msg":      nil,
	})
}

// MarkFailed marks an execution as failed.
func (r *executionRecorder) MarkFailed(ctx context.Context, task *genericjob.Task, err error, duration time.Duration) error {
	if r.skip(task) {
		return nil
	}

	errorMsg := "unknown job task error"
	if err != nil {
		errorMsg = err.Error()
	}
	if len(errorMsg) > 1024 {
		errorMsg = errorMsg[:1024]
	}

	now := time.Now().UTC()
	return r.update(ctx, task.ExecutionID, map[string]any{
		"process_status": processStatusFailed,
		"finished_at":    now,
		"duration_ms":    duration.Milliseconds(),
		"error_msg":      errorMsg,
		"attempt":        gorm.Expr("attempt + ?", 1),
	})
}

// MarkRetrying marks an execution as retrying.
func (r *executionRecorder) MarkRetrying(ctx context.Context, task *genericjob.Task, err error) error {
	if r.skip(task) {
		return nil
	}

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	if len(errorMsg) > 1024 {
		errorMsg = errorMsg[:1024]
	}

	return r.update(ctx, task.ExecutionID, map[string]any{
		"process_status": processStatusRetrying,
		"error_msg":      errorMsg,
		"attempt":        gorm.Expr("attempt + ?", 1),
	})
}

// MarkDead marks an execution as dead (max retries exceeded).
func (r *executionRecorder) MarkDead(ctx context.Context, task *genericjob.Task, err error, duration time.Duration) error {
	if r.skip(task) {
		return nil
	}

	errorMsg := "unknown job task error"
	if err != nil {
		errorMsg = err.Error()
	}
	if len(errorMsg) > 1024 {
		errorMsg = errorMsg[:1024]
	}

	now := time.Now().UTC()
	return r.update(ctx, task.ExecutionID, map[string]any{
		"process_status": processStatusDead,
		"finished_at":    now,
		"duration_ms":    duration.Milliseconds(),
		"error_msg":      errorMsg,
		"attempt":        gorm.Expr("attempt + ?", 1),
	})
}

// skip returns true if recording should be skipped for this task.
func (r *executionRecorder) skip(task *genericjob.Task) bool {
	return r == nil || r.store == nil || task == nil || task.ExecutionID == ""
}

// update persists the given values to the execution record.
func (r *executionRecorder) update(ctx context.Context, executionID string, values map[string]any) error {
	result := r.store.DB(ctx).
		Model(&model.ScheduledTaskExecutionM{}).
		Where("execution_id = ?", executionID).
		Where("process_status NOT IN ?", []string{processStatusSucceeded, processStatusDead}).
		Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var execution model.ScheduledTaskExecutionM
	err := r.store.DB(ctx).Where("execution_id = ?", executionID).First(&execution).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		slog.WarnContext(ctx, "Job execution record not found", "executionID", executionID)
		return nil
	}
	return err
}

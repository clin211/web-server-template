package store

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"
)

// ScheduledTaskStore 定义了 scheduled_task 模块在 store 层所实现的方法.
type ScheduledTaskStore interface {
	Create(ctx context.Context, obj *model.ScheduledTaskM) error
	Update(ctx context.Context, obj *model.ScheduledTaskM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.ScheduledTaskM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.ScheduledTaskM, error)

	ScheduledTaskExpansion
}

// ScheduledTaskExpansion 定义了定时任务操作的附加方法.
// 仅包含有特殊业务逻辑或数据库语义的操作，不包含简单的 By 查询（应使用 Get/List + where.Options）。
type ScheduledTaskExpansion interface {
	// UpdateNextRunTime 更新任务的下次执行时间
	UpdateNextRunTime(ctx context.Context, scheduledTaskID string, nextRunTime int64) error
	// UpdateLastExecution 更新任务的上次执行信息
	UpdateLastExecution(ctx context.Context, scheduledTaskID string, executionID string, lastError *string) error
}

// scheduledTaskStore 是 ScheduledTaskStore 接口的实现.
type scheduledTaskStore struct {
	*genericstore.Store[model.ScheduledTaskM]
	store *datastore
}

// 确保 scheduledTaskStore 实现了 ScheduledTaskStore 接口。
var _ ScheduledTaskStore = (*scheduledTaskStore)(nil)

// newScheduledTaskStore 创建 scheduledTaskStore 的实例.
func newScheduledTaskStore(store *datastore) *scheduledTaskStore {
	return &scheduledTaskStore{
		Store: genericstore.NewStore[model.ScheduledTaskM](store, storelogger.NewLogger()),
		store: store,
	}
}


// UpdateNextRunTime 更新任务的下次执行时间.
// 使用 Select 指定要更新的字段，确保即使值为 0 也会更新。
func (s *scheduledTaskStore) UpdateNextRunTime(ctx context.Context, scheduledTaskID string, nextRunTime int64) error {
	if scheduledTaskID == "" {
		return fmt.Errorf("scheduledTaskID cannot be empty")
	}
	result := s.store.DB(ctx).Model(&model.ScheduledTaskM{}).
		Select("next_run_time").
		Where("scheduled_task_id = ?", scheduledTaskID).
		Updates(map[string]interface{}{"next_run_time": nextRunTime})
	if result.Error != nil {
		return fmt.Errorf("update next_run_time: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateLastExecution 更新任务的上次执行信息.
// 使用 Select 指定要更新的字段，确保正确处理 last_error 为 nil 的情况。
func (s *scheduledTaskStore) UpdateLastExecution(ctx context.Context, scheduledTaskID string, executionID string, lastError *string) error {
	if scheduledTaskID == "" {
		return fmt.Errorf("scheduledTaskID cannot be empty")
	}
	updates := map[string]interface{}{
		"last_execution_id": executionID,
	}
	if lastError != nil {
		updates["last_error"] = *lastError
	}
	result := s.store.DB(ctx).Model(&model.ScheduledTaskM{}).
		Select("last_execution_id", "last_error").
		Where("scheduled_task_id = ?", scheduledTaskID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update last execution: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}


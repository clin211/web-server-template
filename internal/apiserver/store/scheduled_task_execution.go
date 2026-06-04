package store

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScheduledTaskExecutionStore 定义了 scheduled_task_execution 模块在 store 层所实现的方法.
type ScheduledTaskExecutionStore interface {
	Create(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
	Update(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.ScheduledTaskExecutionM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.ScheduledTaskExecutionM, error)

	ScheduledTaskExecutionExpansion
}

// ScheduledTaskExecutionExpansion 定义了定时任务执行记录操作的附加方法.
// 仅包含有特殊业务逻辑或数据库语义的操作，不包含简单的 By 查询（应使用 Get/List + where.Options）。
type ScheduledTaskExecutionExpansion interface {
	// CreateExecutionIfAbsent 插入执行记录（如果不存在则返回现有记录）
	// 使用 PostgreSQL ON CONFLICT DO NOTHING 确保幂等
	CreateExecutionIfAbsent(ctx context.Context, obj *model.ScheduledTaskExecutionM) (*model.ScheduledTaskExecutionM, bool, error)
	// UpdateExecutionStatus 更新执行记录状态
	UpdateExecutionStatus(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
	// UpdateDispatchStatus 更新调度状态
	UpdateDispatchStatus(ctx context.Context, executionID string, dispatchStatus string, asynqTaskID *string, enqueuedAt *int64) error
	// UpdateProcessStatus 更新处理状态
	UpdateProcessStatus(ctx context.Context, executionID string, processStatus string, attempt int32, errorMsg *string, startedAt *int64, finishedAt *int64, durationMs int64) error
}

// scheduledTaskExecutionStore 是 ScheduledTaskExecutionStore 接口的实现.
type scheduledTaskExecutionStore struct {
	*genericstore.Store[model.ScheduledTaskExecutionM]
	store *datastore
}

// 确保 scheduledTaskExecutionStore 实现了 ScheduledTaskExecutionStore 接口。
var _ ScheduledTaskExecutionStore = (*scheduledTaskExecutionStore)(nil)

// newScheduledTaskExecutionStore 创建 scheduledTaskExecutionStore 的实例.
func newScheduledTaskExecutionStore(store *datastore) *scheduledTaskExecutionStore {
	return &scheduledTaskExecutionStore{
		Store: genericstore.NewStore[model.ScheduledTaskExecutionM](store, storelogger.NewLogger()),
		store: store,
	}
}

// CreateExecutionIfAbsent 插入执行记录，如果已存在则返回现有记录.
// 使用 scheduled_task_id + scheduled_at 作为唯一约束，避免重复创建.
func (s *scheduledTaskExecutionStore) CreateExecutionIfAbsent(ctx context.Context, obj *model.ScheduledTaskExecutionM) (*model.ScheduledTaskExecutionM, bool, error) {
	db := s.store.DB(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scheduled_task_id"}, {Name: "scheduled_at"}}, DoNothing: true}).Create(obj)
	if db.Error != nil {
		return nil, false, db.Error
	}
	if db.RowsAffected > 0 {
		return obj, true, nil
	}

	// 如果没有插入新记录，说明已存在，查询返回现有记录
	var existing model.ScheduledTaskExecutionM
	err := s.store.DB(ctx).Where("scheduled_task_id = ? AND scheduled_at = ?", obj.ScheduledTaskID, obj.ScheduledAt).First(&existing).Error
	if err != nil {
		return nil, false, err
	}
	return &existing, false, nil
}

// UpdateExecutionStatus 更新执行记录状态字段.
func (s *scheduledTaskExecutionStore) UpdateExecutionStatus(ctx context.Context, obj *model.ScheduledTaskExecutionM) error {
	if obj.ExecutionID == "" {
		return fmt.Errorf("executionID cannot be empty")
	}
	result := s.store.DB(ctx).Where("execution_id = ?", obj.ExecutionID).Updates(obj)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}


// UpdateDispatchStatus 更新调度状态.
func (s *scheduledTaskExecutionStore) UpdateDispatchStatus(ctx context.Context, executionID string, dispatchStatus string, asynqTaskID *string, enqueuedAt *int64) error {
	if executionID == "" {
		return fmt.Errorf("executionID cannot be empty")
	}

	updates := map[string]any{
		"dispatch_status": dispatchStatus,
	}
	if asynqTaskID != nil {
		updates["asynq_task_id"] = *asynqTaskID
	}
	if enqueuedAt != nil {
		updates["enqueued_at"] = *enqueuedAt
	}

	result := s.store.DB(ctx).Model(&model.ScheduledTaskExecutionM{}).
		Where("execution_id = ?", executionID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update dispatch status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateProcessStatus 更新处理状态.
func (s *scheduledTaskExecutionStore) UpdateProcessStatus(ctx context.Context, executionID string, processStatus string, attempt int32, errorMsg *string, startedAt *int64, finishedAt *int64, durationMs int64) error {
	if executionID == "" {
		return fmt.Errorf("executionID cannot be empty")
	}

	updates := map[string]any{
		"process_status": processStatus,
		"attempt":        attempt,
		"duration_ms":    durationMs,
	}
	if errorMsg != nil {
		updates["error_msg"] = *errorMsg
	}
	if startedAt != nil {
		updates["started_at"] = *startedAt
	}
	if finishedAt != nil {
		updates["finished_at"] = *finishedAt
	}

	result := s.store.DB(ctx).Model(&model.ScheduledTaskExecutionM{}).
		Where("execution_id = ?", executionID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update process status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

package store

import (
	"context"
	"errors"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ScheduledTaskExecutionStore defines the data access operations for task executions.
type ScheduledTaskExecutionStore interface {
	Create(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
	Update(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.ScheduledTaskExecutionM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.ScheduledTaskExecutionM, error)
	CreateExecutionIfAbsent(ctx context.Context, obj *model.ScheduledTaskExecutionM) (*model.ScheduledTaskExecutionM, bool, error)
	UpdateExecutionStatus(ctx context.Context, obj *model.ScheduledTaskExecutionM) error
}

// scheduledTaskExecutionStore implements ScheduledTaskExecutionStore.
type scheduledTaskExecutionStore struct {
	*genericstore.Store[model.ScheduledTaskExecutionM]
	store *datastore
}

var (
	// ErrExecutionIDRequired means updating an execution requires executionID.
	ErrExecutionIDRequired                             = errors.New("executionID cannot be empty")
	_                      ScheduledTaskExecutionStore = (*scheduledTaskExecutionStore)(nil)
)

// newScheduledTaskExecutionStore creates a scheduled task execution store.
func newScheduledTaskExecutionStore(store *datastore) *scheduledTaskExecutionStore {
	return &scheduledTaskExecutionStore{
		Store: genericstore.NewStore[model.ScheduledTaskExecutionM](store, storelogger.NewLogger()),
		store: store,
	}
}

// CreateExecutionIfAbsent inserts a new execution record if one doesn't already exist.
func (s *scheduledTaskExecutionStore) CreateExecutionIfAbsent(ctx context.Context, obj *model.ScheduledTaskExecutionM) (*model.ScheduledTaskExecutionM, bool, error) {
	db := s.store.DB(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scheduled_task_id"}, {Name: "scheduled_at"}}, DoNothing: true}).Create(obj)
	if db.Error != nil {
		return nil, false, db.Error
	}
	if db.RowsAffected > 0 {
		return obj, true, nil
	}

	var existing model.ScheduledTaskExecutionM
	err := s.store.DB(ctx).Where("scheduled_task_id = ? AND scheduled_at = ?", obj.ScheduledTaskID, obj.ScheduledAt).First(&existing).Error
	if err != nil {
		return nil, false, err
	}
	return &existing, false, nil
}

// UpdateExecutionStatus updates the status fields of an execution record.
func (s *scheduledTaskExecutionStore) UpdateExecutionStatus(ctx context.Context, obj *model.ScheduledTaskExecutionM) error {
	if obj.ExecutionID == "" {
		return ErrExecutionIDRequired
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

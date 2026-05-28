package store

import (
	"context"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// ScheduledTaskStore defines the data access operations for scheduled tasks.
type ScheduledTaskStore interface {
	Create(ctx context.Context, obj *model.ScheduledTaskM) error
	Update(ctx context.Context, obj *model.ScheduledTaskM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.ScheduledTaskM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.ScheduledTaskM, error)
	GetByScheduledTaskID(ctx context.Context, scheduledTaskID string, wheres ...where.Where) (*model.ScheduledTaskM, error)
	ListEnabledTasks(ctx context.Context) ([]*model.ScheduledTaskM, error)
	ListChangedSince(ctx context.Context, since time.Time) ([]*model.ScheduledTaskM, error)
}

// scheduledTaskStore implements ScheduledTaskStore.
type scheduledTaskStore struct {
	*genericstore.Store[model.ScheduledTaskM]
	store *datastore
}

var _ ScheduledTaskStore = (*scheduledTaskStore)(nil)

// newScheduledTaskStore creates a scheduled task store.
func newScheduledTaskStore(store *datastore) *scheduledTaskStore {
	return &scheduledTaskStore{
		Store: genericstore.NewStore[model.ScheduledTaskM](store, storelogger.NewLogger()),
		store: store,
	}
}

// GetByScheduledTaskID retrieves a task by its scheduled task ID.
func (s *scheduledTaskStore) GetByScheduledTaskID(ctx context.Context, scheduledTaskID string, wheres ...where.Where) (*model.ScheduledTaskM, error) {
	var task model.ScheduledTaskM
	if err := s.store.DB(ctx, wheres...).Where("scheduled_task_id = ?", scheduledTaskID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListEnabledTasks returns all tasks that are enabled.
func (s *scheduledTaskStore) ListEnabledTasks(ctx context.Context) ([]*model.ScheduledTaskM, error) {
	var tasks []*model.ScheduledTaskM
	err := s.store.DB(ctx).Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// ListChangedSince returns all tasks modified after the given time.
func (s *scheduledTaskStore) ListChangedSince(ctx context.Context, since time.Time) ([]*model.ScheduledTaskM, error) {
	var tasks []*model.ScheduledTaskM
	err := s.store.DB(ctx).Where("updated_at >= ?", since).Find(&tasks).Error
	return tasks, err
}

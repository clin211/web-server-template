// Package store 实现 API 资源的持久化。
//
//nolint:dupl
package store

import (
	"context"
	"time"

	"gorm.io/gorm"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// UserStore 定义了 user 模块在 store 层所实现的方法.
type UserStore interface {
	UserReader
	UserWriter
	UserExpansion
}

// UserReader 定义用户读取操作。
type UserReader interface {
	Get(ctx context.Context, opts *where.Options) (*model.UserM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.UserM, error)
}

// UserWriter 定义用户写入操作。
type UserWriter interface {
	Create(ctx context.Context, obj *model.UserM) error
	Update(ctx context.Context, userID string, updates map[string]any) error
	Delete(ctx context.Context, opts *where.Options) error
}

// Update 只更新调用方明确给出的字段，避免 Save 把读取后已过期的整行快照写回数据库。
func (s *userStore) Update(ctx context.Context, userID string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	result := s.core.DB(ctx).Model(&model.UserM{}).Where("user_id = ?", userID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UserExpansion 定义了用户操作的附加方法.
// nolint: iface
type UserExpansion interface {
	UpdateLastLoginAt(ctx context.Context, userID string, t time.Time) error
}

// userStore 是 UserStore 接口的实现。
type userStore struct {
	*genericstore.Store[model.UserM]

	core *datastore
}

// 确保 userStore 实现了 UserStore 接口。
var _ UserStore = (*userStore)(nil)

// newUserStore 创建 userStore 的实例。
func newUserStore(store *datastore) *userStore {
	return &userStore{
		Store: genericstore.NewStore[model.UserM](store, storelogger.NewLogger()),
		core:  store,
	}
}

func (s *userStore) UpdateLastLoginAt(ctx context.Context, userID string, t time.Time) error {
	return s.core.DB(ctx).Model(&model.UserM{}).Where("user_id = ?", userID).Update("last_login_at", t).Error
}

// Create atomically assigns the minimal role with account registration.
func (s *userStore) Create(ctx context.Context, user *model.UserM) error {
	return s.core.TX(ctx, func(tx context.Context) error {
		if err := s.core.DB(tx).WithContext(tx).Create(user).Error; err != nil {
			return err
		}
		return NewAuthorizationStore(s.core).AssignDefaultRole(tx, user.UserID)
	})
}

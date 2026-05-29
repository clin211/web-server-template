// nolint: dupl
package store

import (
	"context"
	"time"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// UserStore 定义了 user 模块在 store 层所实现的方法.
type UserStore interface {
	Create(ctx context.Context, obj *model.UserM) error
	Update(ctx context.Context, obj *model.UserM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.UserM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.UserM, error)

	UserExpansion
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

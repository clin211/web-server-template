package store

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// UserLoginLogStore 定义登录安全审计的最小写入接口。
type UserLoginLogStore interface {
	Create(ctx context.Context, record *model.UserLoginLogM) error
}

type userLoginLogStore struct{ core *datastore }

func newUserLoginLogStore(store *datastore) *userLoginLogStore {
	return &userLoginLogStore{core: store}
}

func (s *userLoginLogStore) Create(ctx context.Context, record *model.UserLoginLogM) error {
	if err := s.core.DB(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("create user login log: %w", err)
	}
	return nil
}

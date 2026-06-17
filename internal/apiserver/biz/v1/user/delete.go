package user

import (
	"context"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Delete 实现 UserBiz 接口中的 Delete 方法.
func (b *userBiz) Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	if err := b.store.User().Delete(ctx, where.F("user_id", rq.GetUserID())); err != nil {
		return nil, err
	}

	return &v1.DeleteUserResponse{}, nil
}

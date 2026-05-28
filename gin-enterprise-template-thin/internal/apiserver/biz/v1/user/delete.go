package user

import (
	"context"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Delete 实现 UserBiz 接口中的 Delete 方法.
func (b *userBiz) Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	// 只有 `root` 用户可以删除用户，并且可以删除其他用户
	// 所以这里不用 where.T()，因为 where.T() 会查询 `root` 用户自己
	if err := b.store.User().Delete(ctx, where.F("user_id", rq.GetUserID())); err != nil {
		return nil, err
	}

	if _, err := b.authz.RemoveGroupingPolicy(rq.GetUserID(), known.RoleUser); err != nil {
		slog.ErrorContext(ctx, "Failed to remove grouping policy for user", "user", rq.GetUserID(), "role", known.RoleUser, "error", err)
		return nil, errno.ErrRemoveRole.WithMessage(err.Error())
	}

	return &v1.DeleteUserResponse{}, nil
}

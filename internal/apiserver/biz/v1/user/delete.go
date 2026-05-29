package user

import (
	"context"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Delete 实现 UserBiz 接口中的 Delete 方法.
func (b *userBiz) Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	roleCodes := make([]string, 0)
	if err := b.store.TX(ctx, func(txCtx context.Context) error {
		roles, err := b.store.UserRole().GetUserRoles(txCtx, rq.GetUserID())
		if err != nil {
			return err
		}

		roleCodes = make([]string, 0, len(roles))
		for _, role := range roles {
			roleCodes = append(roleCodes, "role::"+role.RoleCode)
		}

		// 只有 `root` 用户可以删除用户，并且可以删除其他用户
		// 所以这里不用 where.T()，因为 where.T() 会查询 `root` 用户自己
		if err := b.store.User().Delete(txCtx, where.F("user_id", rq.GetUserID())); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	for _, roleCode := range roleCodes {
		if _, err := b.authz.RemoveGroupingPolicy(rq.GetUserID(), roleCode); err != nil {
			slog.WarnContext(ctx, "Failed to remove grouping policy for deleted user", "user", rq.GetUserID(), "role", roleCode, "error", err)
		}
	}

	return &v1.DeleteUserResponse{}, nil
}

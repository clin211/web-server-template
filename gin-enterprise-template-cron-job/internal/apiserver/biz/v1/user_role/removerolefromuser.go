package user_role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"log/slog"
)

// RemoveRoleFromUser 从用户移除角色.
func (b *userRoleBiz) RemoveRoleFromUser(ctx context.Context, rq *v1.RemoveRoleFromUserRequest) (*v1.RemoveRoleFromUserResponse, error) {
	userID := rq.GetUserID()
	roleID := rq.GetRoleID()

	// 获取角色信息
	roleM, err := b.store.Role().Get(ctx, where.F("role_id", roleID).L(1))
	if err != nil {
		return nil, errno.ErrRoleNotFound
	}

	// 从数据库中移除用户-角色关系
	if err := b.store.UserRole().RemoveRole(ctx, userID, roleID); err != nil {
		return nil, err
	}

	// 从 Casbin 中删除用户-角色关系
	casbinRole := "role::" + roleM.RoleCode
	if _, err := b.authz.RemoveGroupingPolicy(userID, casbinRole); err != nil {
		slog.WarnContext(ctx, "Failed to remove grouping policy from Casbin", "userID", userID, "role", casbinRole, "error", err)
	}

	return &v1.RemoveRoleFromUserResponse{}, nil
}

package user_role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"log/slog"
)

// GetUserRoles 获取用户的角色和权限.
func (b *userRoleBiz) GetUserRoles(ctx context.Context, rq *v1.GetUserRolesRequest) (*v1.GetUserRolesResponse, error) {
	userID := rq.GetUserID()

	// 获取用户的角色列表
	roles, err := b.store.UserRole().GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取用户的所有权限编码
	permissionCodes, err := b.store.UserRole().GetUserPermissions(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get user permissions", "userID", userID, "error", err)
		permissionCodes = []string{}
	}

	return &v1.GetUserRolesResponse{
		Roles:          conversion.RoleModelListToRoleV1List(roles),
		PermissionCodes: permissionCodes,
	}, nil
}

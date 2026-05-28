package user_role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"log/slog"
)

// AssignRolesToUser 为用户分配角色（覆盖模式）.
func (b *userRoleBiz) AssignRolesToUser(ctx context.Context, rq *v1.AssignRolesToUserRequest) (*v1.AssignRolesToUserResponse, error) {
	userID := rq.GetUserID()

	// 验证用户是否存在
	if _, err := b.store.User().Get(ctx, where.F("user_id", userID).L(1)); err != nil {
		return nil, errno.ErrUserNotFound
	}

	// 验证所有角色是否存在
	for _, roleID := range rq.GetRoleIDs() {
		if _, err := b.store.Role().Get(ctx, where.F("role_id", roleID).L(1)); err != nil {
			slog.WarnContext(ctx, "Role not found", "roleID", roleID)
			return nil, errno.ErrRoleNotFound
		}
	}

	// 获取用户当前角色列表
	oldRoles, _ := b.store.UserRole().GetUserRoles(ctx, userID)
	oldRoleCodes := make([]string, 0, len(oldRoles))
	for _, r := range oldRoles {
		oldRoleCodes = append(oldRoleCodes, "role::"+r.RoleCode)
	}

	// 分配新角色
	if err := b.store.UserRole().AssignRoles(ctx, userID, rq.GetRoleIDs()); err != nil {
		return nil, err
	}

	// 同步到 Casbin：删除旧的用户-角色关系
	for _, oldRole := range oldRoleCodes {
		if _, err := b.authz.RemoveGroupingPolicy(userID, oldRole); err != nil {
			slog.WarnContext(ctx, "Failed to remove old grouping policy", "userID", userID, "role", oldRole, "error", err)
		}
	}

	// 同步到 Casbin：添加新的用户-角色关系
	for _, roleID := range rq.GetRoleIDs() {
		roleM, err := b.store.Role().Get(ctx, where.F("role_id", roleID).L(1))
		if err != nil {
			continue
		}
		casbinRole := "role::" + roleM.RoleCode
		if _, err := b.authz.AddGroupingPolicy(userID, casbinRole); err != nil {
			slog.ErrorContext(ctx, "Failed to add grouping policy", "userID", userID, "role", casbinRole, "error", err)
			return nil, errno.ErrAddRole.WithMessage(err.Error())
		}
	}

	return &v1.AssignRolesToUserResponse{}, nil
}

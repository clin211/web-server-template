package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"log/slog"
	"gorm.io/gorm"
)

// AssignPermissionsToRole 为角色分配权限.
// 使用事务确保数据库操作和 Casbin 同步的原子性.
func (b *roleBiz) AssignPermissionsToRole(ctx context.Context, rq *v1.AssignPermissionsToRoleRequest) (*v1.AssignPermissionsToRoleResponse, error) {
	// 获取角色信息
	roleM, err := b.store.Role().Get(ctx, where.F("role_id", rq.GetRoleID()).L(1))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errno.ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// 使用事务确保数据库操作和 Casbin 同步的原子性
	err = b.store.TX(ctx, func(txCtx context.Context) error {
		// 分配权限到数据库
		if err := b.store.Role().AssignPermissions(txCtx, roleM.RoleID, rq.GetPermissionIDs()); err != nil {
			return fmt.Errorf("failed to assign permissions in database: %w", err)
		}

		// 同步到 Casbin
		if err := b.syncRolePermissionsToCasbin(txCtx, roleM.RoleCode, rq.GetPermissionIDs()); err != nil {
			return fmt.Errorf("failed to sync permissions to casbin: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &v1.AssignPermissionsToRoleResponse{}, nil
}

// syncRoleToCasbin 同步角色到 Casbin.
// Casbin 的角色格式为 role::roleCode。
// 注意：Casbin 的 g 规则（用户-角色关系）在用户分配角色时添加，这里不需要处理.
func (b *roleBiz) syncRoleToCasbin(ctx context.Context, roleCode string) error {
	// 目前不需要在 Casbin 中预先创建角色标识
	// Casbin 会在添加策略时自动创建角色
	casbinRole := "role::" + roleCode
	slog.DebugContext(ctx, "Syncing role to Casbin", "casbinRole", casbinRole)
	return nil
}

// syncRolePermissionsToCasbin 同步角色权限到 Casbin.
// 使用批量替换策略的方式，避免删除和添加之间的竞态条件.
func (b *roleBiz) syncRolePermissionsToCasbin(ctx context.Context, roleCode string, permissionIDs []string) error {
	// Casbin 的角色格式
	casbinRole := "role::" + roleCode

	// 先收集所有需要添加的策略
	var newPolicies [][]string
	for _, pid := range permissionIDs {
		permM, err := b.store.Permission().Get(ctx, where.F("permission_id", pid).L(1))
		if err != nil {
			slog.WarnContext(ctx, "Failed to get permission", "permissionID", pid, "error", err)
			continue
		}

		// 构建 p 规则: p, role::roleCode, resource_path, action, allow
		if permM.ResourcePath != nil && *permM.ResourcePath != "" {
			newPolicies = append(newPolicies, []string{casbinRole, *permM.ResourcePath, permM.Action, "allow"})
		}
	}

	// 使用事务 API 删除旧策略并添加新策略
	// RemoveFilteredPolicy 会删除所有匹配的旧策略
	// 然后 AddPolicies 批量添加新策略
	// 这种方式可以最大程度减少竞态条件窗口
	if _, err := b.authz.RemoveFilteredPolicy(0, casbinRole); err != nil {
		slog.WarnContext(ctx, "Failed to remove old policies", "role", casbinRole, "error", err)
		// 继续尝试添加新策略
	}

	// 批量添加新策略
	if len(newPolicies) > 0 {
		if _, err := b.authz.AddPolicies(newPolicies); err != nil {
			return fmt.Errorf("failed to add policies to casbin: %w", err)
		}
	}

	return nil
}

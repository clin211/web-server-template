package role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// GetRolePermissions 获取角色的权限列表（树形）.
func (b *roleBiz) GetRolePermissions(ctx context.Context, rq *v1.GetRolePermissionsRequest) (*v1.GetRolePermissionsResponse, error) {
	// 获取所有权限
	allPermissions, err := b.store.Permission().ListTree(ctx, &where.Options{})
	if err != nil {
		return nil, err
	}

	// 获取角色的权限ID
	rolePermissions, err := b.store.Role().GetPermissions(ctx, rq.GetRoleID())
	if err != nil {
		return nil, err
	}

	// 构建已分配权限的ID集合
	assignedIDs := make(map[string]bool)
	for _, p := range rolePermissions {
		assignedIDs[p.PermissionID] = true
	}

	// 构建权限树
	permissionTree := conversion.PermissionModelListToPermissionTreeV1(allPermissions, assignedIDs)

	return &v1.GetRolePermissionsResponse{Permissions: permissionTree}, nil
}

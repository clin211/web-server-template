package menu_role

import (
	"context"
	"fmt"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// RemoveMenuRole 移除菜单允许的角色.
func (b *menuRoleBiz) RemoveMenuRole(ctx context.Context, rq *v1.RemoveMenuRoleRequest) (*v1.RemoveMenuRoleResponse, error) {
	menuID := rq.GetMenuID()
	roleID := rq.GetRoleId()

	if err := b.validateMenuExists(ctx, menuID); err != nil {
		return nil, fmt.Errorf("remove menu role: %w", err)
	}

	if err := b.validateRoleExists(ctx, roleID); err != nil {
		return nil, fmt.Errorf("remove menu role: %w", err)
	}

	if err := b.store.MenuRole().Delete(ctx, where.F("menu_id", menuID).F("role_id", roleID)); err != nil {
		return nil, fmt.Errorf("remove menu role: %w", err)
	}

	return &v1.RemoveMenuRoleResponse{}, nil
}
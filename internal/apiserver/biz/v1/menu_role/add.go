package menu_role

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// AddMenuRole 追加菜单允许的角色.
func (b *menuRoleBiz) AddMenuRole(ctx context.Context, rq *v1.AddMenuRoleRequest) (*v1.AddMenuRoleResponse, error) {
	menuID := rq.GetMenuID()
	roleID := rq.GetRoleId()

	if err := b.validateMenuExists(ctx, menuID); err != nil {
		return nil, fmt.Errorf("add menu role: %w", err)
	}

	if err := b.validateRoleExists(ctx, roleID); err != nil {
		return nil, fmt.Errorf("add menu role: %w", err)
	}

	menuRoles, err := b.store.MenuRole().ListByMenuID(ctx, menuID)
	if err != nil {
		return nil, fmt.Errorf("add menu role: %w", err)
	}

	for _, mr := range menuRoles {
		if mr.RoleID == roleID {
			return &v1.AddMenuRoleResponse{}, nil
		}
	}

	menuRole := &model.MenuRoleM{
		MenuID: menuID,
		RoleID: roleID,
	}
	if err := b.store.MenuRole().Create(ctx, menuRole); err != nil {
		return nil, fmt.Errorf("add menu role: %w", err)
	}

	return &v1.AddMenuRoleResponse{}, nil
}
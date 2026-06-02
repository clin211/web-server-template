package menu_role

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// SetMenuRoles 批量设置菜单允许的角色（覆盖模式）.
func (b *menuRoleBiz) SetMenuRoles(ctx context.Context, rq *v1.SetMenuRolesRequest) (*v1.SetMenuRolesResponse, error) {
	menuID := rq.GetMenuID()
	roleIDs := rq.GetRoleIds()

	if err := b.validateMenuExists(ctx, menuID); err != nil {
		return nil, fmt.Errorf("set menu roles: %w", err)
	}

	if err := b.store.MenuRole().SetMenuRoles(ctx, menuID, roleIDs); err != nil {
		return nil, fmt.Errorf("set menu roles: %w", err)
	}

	uniqueRoleIDs := conversion.UniqueStrings(roleIDs)

	return &v1.SetMenuRolesResponse{
		MenuId:  menuID,
		RoleIds: uniqueRoleIDs,
		Count:   int32(len(uniqueRoleIDs)),
	}, nil
}
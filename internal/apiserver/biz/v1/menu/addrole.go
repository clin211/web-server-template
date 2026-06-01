package menu

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// AddMenuRole 追加菜单允许的角色.
func (b *menuBiz) AddMenuRole(ctx context.Context, rq *v1.AddMenuRoleRequest) (*v1.AddMenuRoleResponse, error) {
	// 先验证菜单是否存在
	_, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, fmt.Errorf("get menu for add role: %w", err)
	}

	// 检查角色是否已存在
	menuRoles, err := b.store.MenuRole().ListByMenuID(ctx, rq.GetMenuID())
	if err != nil {
		return nil, fmt.Errorf("list menu roles: %w", err)
	}

	for _, mr := range menuRoles {
		if mr.RoleID == rq.GetRoleId() {
			// 角色已存在，无需重复添加
			return &v1.AddMenuRoleResponse{}, nil
		}
	}

	// 追加角色
	menuRole := &model.MenuRoleM{
		MenuID: rq.GetMenuID(),
		RoleID: rq.GetRoleId(),
	}
	if err := b.store.MenuRole().Create(ctx, menuRole); err != nil {
		return nil, fmt.Errorf("add menu role: %w", err)
	}

	return &v1.AddMenuRoleResponse{}, nil
}
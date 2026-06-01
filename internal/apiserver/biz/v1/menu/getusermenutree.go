package menu

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// GetUserMenuTree 获取用户可见的菜单树.
func (b *menuBiz) GetUserMenuTree(ctx context.Context, _ *v1.GetUserMenuTreeRequest) (*v1.GetUserMenuTreeResponse, error) {
	userID := contextx.UserID(ctx)

	// 获取用户可见的菜单及角色映射
	menus, rolesMap, err := b.store.Menu().GetUserMenusWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user menus for menu tree: %w", err)
	}

	// 使用 BuildMenuTreeWithRoles 构建树，避免 N+1 查询
	routes := conversion.BuildMenuTreeWithRoles(menus, rolesMap)

	return &v1.GetUserMenuTreeResponse{Menus: routes}, nil
}
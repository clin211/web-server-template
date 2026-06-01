package menu

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// GetUserRoutes 获取用户可访问的路由树.
// 包含用户有权限访问的所有菜单路由（常量路由 + 动态路由）。
func (b *menuBiz) GetUserRoutes(ctx context.Context, _ *v1.GetUserRoutesRequest) (*v1.GetUserRoutesResponse, error) {
	userID := contextx.UserID(ctx)

	// 获取用户可见的菜单及角色映射
	menus, rolesMap, err := b.store.Menu().GetUserMenusWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user menus with roles: %w", err)
	}

	// 使用预加载的角色映射构建路由树，避免 N+1 查询
	routes := conversion.BuildMenuTreeWithRoles(menus, rolesMap)

	return &v1.GetUserRoutesResponse{
		Routes: routes,
		Home:   "home",
	}, nil
}

// GetConstantRoutes 获取常量路由.
// 常量路由是不参与权限过滤的路由，对所有用户可见。
func (b *menuBiz) GetConstantRoutes(ctx context.Context, _ *v1.GetConstantRoutesRequest) (*v1.GetConstantRoutesResponse, error) {
	// 获取常量路由菜单及角色映射
	menus, rolesMap, err := b.store.Menu().GetConstantMenusWithRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("get constant menus with roles: %w", err)
	}

	// 使用预加载的角色映射构建路由树
	routes := conversion.BuildMenuTreeWithRoles(menus, rolesMap)

	return &v1.GetConstantRoutesResponse{Routes: routes}, nil
}
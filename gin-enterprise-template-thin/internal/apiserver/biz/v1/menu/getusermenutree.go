package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// GetUserMenuTree 获取用户可见的菜单树.
func (b *menuBiz) GetUserMenuTree(ctx context.Context, rq *v1.GetUserMenuTreeRequest) (*v1.GetUserMenuTreeResponse, error) {
	// 从上下文获取用户 ID
	userID := getUserIDFromContext(ctx)

	menus, err := b.store.Menu().GetUserMenus(ctx, userID)
	if err != nil {
		return nil, err
	}

	tree := conversion.MenuModelListToMenuTreeV1(menus)

	return &v1.GetUserMenuTreeResponse{Menus: tree}, nil
}

// getUserIDFromContext 从上下文获取用户 ID
// 这里应该从认证上下文中获取，暂时返回空字符串
func getUserIDFromContext(ctx context.Context) string {
	// TODO: 从 JWT 或其他认证方式获取用户 ID
	return ""
}

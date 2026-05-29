package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// GetUserMenuTree 获取用户可见的菜单树.
func (b *menuBiz) GetUserMenuTree(ctx context.Context, _ *v1.GetUserMenuTreeRequest) (*v1.GetUserMenuTreeResponse, error) {
	menus, err := b.store.Menu().GetUserMenus(ctx, contextx.UserID(ctx))
	if err != nil {
		return nil, err
	}

	tree := conversion.MenuModelListToMenuTreeV1(menus)

	return &v1.GetUserMenuTreeResponse{Menus: tree}, nil
}

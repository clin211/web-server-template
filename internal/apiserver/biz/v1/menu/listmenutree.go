package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// ListMenuTree 获取菜单树.
func (b *menuBiz) ListMenuTree(ctx context.Context, rq *v1.ListMenuTreeRequest) (*v1.ListMenuTreeResponse, error) {
	menus, err := b.store.Menu().ListTree(ctx, nil)
	if err != nil {
		return nil, err
	}

	tree := conversion.MenuModelListToMenuTreeV1(menus)

	return &v1.ListMenuTreeResponse{Menus: tree}, nil
}

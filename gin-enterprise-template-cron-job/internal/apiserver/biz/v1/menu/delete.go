package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Delete 删除菜单.
func (b *menuBiz) Delete(ctx context.Context, rq *v1.DeleteMenuRequest) (*v1.DeleteMenuResponse, error) {
	// 检查是否有子菜单
	children, err := b.store.Menu().GetChildren(ctx, rq.GetMenuID())
	if err == nil && len(children) > 0 {
		return nil, errno.ErrMenuHasChildren
	}

	if err := b.store.Menu().Delete(ctx, where.F("menu_id", rq.GetMenuID())); err != nil {
		return nil, err
	}

	return &v1.DeleteMenuResponse{}, nil
}

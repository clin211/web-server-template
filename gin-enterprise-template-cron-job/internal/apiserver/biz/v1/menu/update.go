package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/jinzhu/copier"
)

// Update 更新菜单.
func (b *menuBiz) Update(ctx context.Context, rq *v1.UpdateMenuRequest) (*v1.UpdateMenuResponse, error) {
	menuM, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, errno.ErrMenuNotFound
	}

	// 使用 copier 更新字段
	if err := copier.CopyWithOption(menuM, rq, copier.Option{IgnoreEmpty: true}); err != nil {
		return nil, err
	}

	if err := b.store.Menu().Update(ctx, menuM); err != nil {
		return nil, err
	}

	return &v1.UpdateMenuResponse{}, nil
}

package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/jinzhu/copier"
)

// Create 创建菜单.
func (b *menuBiz) Create(ctx context.Context, rq *v1.CreateMenuRequest) (*v1.CreateMenuResponse, error) {
	var menuM conversion.MenuModel
	if err := copier.Copy(&menuM, rq); err != nil {
		return nil, err
	}

	// 检查菜单编码是否已存在
	if existingMenu, err := b.store.Menu().GetByMenuCode(ctx, menuM.MenuCode); err == nil && existingMenu != nil {
		return nil, errno.ErrMenuAlreadyExists
	}

	if err := b.store.Menu().Create(ctx, &menuM); err != nil {
		return nil, err
	}

	return &v1.CreateMenuResponse{MenuID: menuM.MenuID}, nil
}

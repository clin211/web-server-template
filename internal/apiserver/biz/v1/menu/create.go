package menu

import (
	"context"
	"errors"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

// Create 创建菜单.
func (b *menuBiz) Create(ctx context.Context, rq *v1.CreateMenuRequest) (*v1.CreateMenuResponse, error) {
	var menuM conversion.MenuModel
	if err := copier.Copy(&menuM, rq); err != nil {
		return nil, fmt.Errorf("copy menu request: %w", err)
	}

	// 检查菜单编码是否已存在
	existingMenu, err := b.store.Menu().Get(ctx, where.NewWhere().F("menu_code", menuM.MenuCode).L(1))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check menu code: %w", err)
	}
	if existingMenu != nil {
		return nil, errno.ErrMenuAlreadyExists
	}

	if err := b.store.Menu().Create(ctx, &menuM); err != nil {
		return nil, fmt.Errorf("create menu: %w", err)
	}

	return &v1.CreateMenuResponse{MenuID: menuM.MenuID}, nil
}
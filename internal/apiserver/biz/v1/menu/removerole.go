package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// RemoveMenuRole 移除菜单允许的角色.
func (b *menuBiz) RemoveMenuRole(ctx context.Context, rq *v1.RemoveMenuRoleRequest) (*v1.RemoveMenuRoleResponse, error) {
	// 先验证菜单是否存在
	_, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, errno.ErrMenuNotFound
	}

	// 删除菜单角色关联
	if err := b.store.MenuRole().Delete(ctx, where.F("menu_id", rq.GetMenuID()).F("role_id", rq.GetRoleId())); err != nil {
		return nil, err
	}

	return &v1.RemoveMenuRoleResponse{}, nil
}
package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// SetMenuRoles 批量设置菜单允许的角色（覆盖模式）.
func (b *menuBiz) SetMenuRoles(ctx context.Context, rq *v1.SetMenuRolesRequest) (*v1.SetMenuRolesResponse, error) {
	// 先验证菜单是否存在
	_, err := b.store.Menu().Get(ctx, where.F("menu_id", rq.GetMenuID()).L(1))
	if err != nil {
		return nil, errno.ErrMenuNotFound
	}

	// 批量设置菜单角色（覆盖模式）
	if err := b.store.MenuRole().SetMenuRoles(ctx, rq.GetMenuID(), rq.GetRoleIds()); err != nil {
		return nil, err
	}

	// 去重后的角色ID列表
	uniqueRoleIDs := conversion.UniqueStrings(rq.GetRoleIds())

	return &v1.SetMenuRolesResponse{
		MenuId:  rq.GetMenuID(),
		RoleIds: uniqueRoleIDs,
		Count:   int32(len(uniqueRoleIDs)),
	}, nil
}
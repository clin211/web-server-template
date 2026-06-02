package menu_role

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// MenuRoleBiz 定义处理菜单角色关联请求所需的方法.
type MenuRoleBiz interface {
	// GetMenuRoles 获取菜单允许访问的角色列表
	GetMenuRoles(ctx context.Context, rq *v1.GetMenuRolesRequest) (*v1.GetMenuRolesResponse, error)
	// SetMenuRoles 批量设置菜单允许的角色（覆盖模式）
	SetMenuRoles(ctx context.Context, rq *v1.SetMenuRolesRequest) (*v1.SetMenuRolesResponse, error)
	// AddMenuRole 追加菜单允许的角色
	AddMenuRole(ctx context.Context, rq *v1.AddMenuRoleRequest) (*v1.AddMenuRoleResponse, error)
	// RemoveMenuRole 移除菜单允许的角色
	RemoveMenuRole(ctx context.Context, rq *v1.RemoveMenuRoleRequest) (*v1.RemoveMenuRoleResponse, error)
}

// menuRoleBiz 是 MenuRoleBiz 接口的实现.
type menuRoleBiz struct {
	store store.IStore
}

// 确保 menuRoleBiz 实现了 MenuRoleBiz 接口.
var _ MenuRoleBiz = (*menuRoleBiz)(nil)

// New 创建 menuRoleBiz 的实例.
func New(store store.IStore) *menuRoleBiz {
	return &menuRoleBiz{store: store}
}

// validateMenuExists 验证菜单是否存在.
func (b *menuRoleBiz) validateMenuExists(ctx context.Context, menuID string) error {
	if _, err := b.store.Menu().Get(ctx, where.F("menu_id", menuID).L(1)); err != nil {
		return fmt.Errorf("get menu: %w", err)
	}
	return nil
}

// validateRoleExists 验证角色是否存在.
func (b *menuRoleBiz) validateRoleExists(ctx context.Context, roleID string) error {
	if _, err := b.store.Role().Get(ctx, where.F("role_id", roleID).L(1)); err != nil {
		return fmt.Errorf("get role: %w", err)
	}
	return nil
}
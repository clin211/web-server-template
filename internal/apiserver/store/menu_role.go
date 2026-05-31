package store

import (
	"context"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// MenuRoleStore 定义了 menu_role 模块在 store 层所实现的方法.
type MenuRoleStore interface {
	// Create 创建菜单角色关联
	Create(ctx context.Context, obj *model.MenuRoleM) error
	// Delete 删除菜单角色关联
	Delete(ctx context.Context, opts *where.Options) error
	// ListByMenuID 根据菜单ID获取菜单角色关联列表
	ListByMenuID(ctx context.Context, menuID string) ([]*model.MenuRoleM, error)
	// ListByRoleID 根据角色ID获取菜单角色关联列表
	ListByRoleID(ctx context.Context, roleID string) ([]*model.MenuRoleM, error)

	MenuRoleExpansion
}

// MenuRoleExpansion 定义了菜单角色操作的附加方法.
type MenuRoleExpansion interface {
	// SetMenuRoles 为菜单设置角色（覆盖模式）
	SetMenuRoles(ctx context.Context, menuID string, roleIDs []string) error
	// GetMenuAllowedRoleCodes 获取菜单允许的角色代码列表
	GetMenuAllowedRoleCodes(ctx context.Context, menuID string) ([]string, error)
	// GetMenuRoles 获取菜单的所有角色信息（roleIds 和 roleCodes 一次查询）
	GetMenuRoles(ctx context.Context, menuID string) ([]string, []string, error)
	// BatchGetMenuAllowedRoleCodes 批量获取多个菜单允许的角色代码列表（解决 N+1 查询问题）
	BatchGetMenuAllowedRoleCodes(ctx context.Context, menuIDs []string) (map[string][]string, error)
}

// menuRoleStore 是 MenuRoleStore 接口的实现。
type menuRoleStore struct {
	*genericstore.Store[model.MenuRoleM]
	core *datastore
}

// 确保 menuRoleStore 实现了 MenuRoleStore 接口。
var _ MenuRoleStore = (*menuRoleStore)(nil)

// newMenuRoleStore 创建 menuRoleStore 的实例。
func newMenuRoleStore(store *datastore) *menuRoleStore {
	return &menuRoleStore{
		Store: genericstore.NewStore[model.MenuRoleM](store, storelogger.NewLogger()),
		core:  store,
	}
}

// SetMenuRoles 为菜单设置角色（覆盖模式）
func (s *menuRoleStore) SetMenuRoles(ctx context.Context, menuID string, roleIDs []string) error {
	// 使用事务确保删除和插入操作原子性
	return s.core.TX(ctx, func(ctx context.Context) error {
		// 先删除该菜单现有的角色关联
		if err := s.DeleteByMenuID(ctx, menuID); err != nil {
			return err
		}

		// 批量插入新角色
		if len(roleIDs) > 0 {
			menuRoles := make([]model.MenuRoleM, 0, len(roleIDs))
			for _, roleID := range roleIDs {
				menuRoles = append(menuRoles, model.MenuRoleM{
					MenuID: menuID,
					RoleID: roleID,
				})
			}
			if err := s.core.DB(ctx).Create(&menuRoles).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetMenuAllowedRoleCodes 获取菜单允许的角色代码列表
func (s *menuRoleStore) GetMenuAllowedRoleCodes(ctx context.Context, menuID string) ([]string, error) {
	var roleCodes []string

	// 通过 menu_role 关联表查询菜单允许的角色代码
	if err := s.core.DB(ctx).
		Table("role").
		Joins("INNER JOIN menu_role ON role.role_id = menu_role.role_id").
		Where("menu_role.menu_id = ?", menuID).
		Pluck("role.role_code", &roleCodes).Error; err != nil {
		return nil, err
	}

	return roleCodes, nil
}

// DeleteByMenuID 根据菜单ID删除菜单角色关联
func (s *menuRoleStore) DeleteByMenuID(ctx context.Context, menuID string) error {
	return s.core.DB(ctx).
		Where("menu_id = ?", menuID).
		Delete(&model.MenuRoleM{}).Error
}

// ListByMenuID 根据菜单ID获取菜单角色关联列表
func (s *menuRoleStore) ListByMenuID(ctx context.Context, menuID string) ([]*model.MenuRoleM, error) {
	var menuRoles []*model.MenuRoleM
	if err := s.core.DB(ctx).Where("menu_id = ?", menuID).Find(&menuRoles).Error; err != nil {
		return nil, err
	}
	return menuRoles, nil
}

// ListByRoleID 根据角色ID获取菜单角色关联列表
func (s *menuRoleStore) ListByRoleID(ctx context.Context, roleID string) ([]*model.MenuRoleM, error) {
	var menuRoles []*model.MenuRoleM
	if err := s.core.DB(ctx).Where("role_id = ?", roleID).Find(&menuRoles).Error; err != nil {
		return nil, err
	}
	return menuRoles, nil
}

// GetMenuRoles 获取菜单的所有角色信息（roleIds 和 roleCodes 一次查询）
func (s *menuRoleStore) GetMenuRoles(ctx context.Context, menuID string) ([]string, []string, error) {
	var results []struct {
		RoleID    string
		RoleCode  string
	}

	// 一次性查询角色ID和角色代码
	if err := s.core.DB(ctx).
		Table("menu_role").
		Joins("INNER JOIN role ON menu_role.role_id = role.role_id").
		Where("menu_role.menu_id = ?", menuID).
		Select("menu_role.role_id, role.role_code").
		Find(&results).Error; err != nil {
		return nil, nil, err
	}

	roleIDs := make([]string, 0, len(results))
	roleCodes := make([]string, 0, len(results))
	for _, r := range results {
		roleIDs = append(roleIDs, r.RoleID)
		roleCodes = append(roleCodes, r.RoleCode)
	}

	return roleIDs, roleCodes, nil
}

// BatchGetMenuAllowedRoleCodes 批量获取多个菜单允许的角色代码列表.
// 解决 N+1 查询问题，将多次查询合并为一次。
func (s *menuRoleStore) BatchGetMenuAllowedRoleCodes(ctx context.Context, menuIDs []string) (map[string][]string, error) {
	if len(menuIDs) == 0 {
		return make(map[string][]string), nil
	}

	var results []struct {
		MenuID   string
		RoleCode string
	}

	// 一次性查询所有菜单的角色代码
	if err := s.core.DB(ctx).
		Table("menu_role").
		Joins("INNER JOIN role ON menu_role.role_id = role.role_id").
		Where("menu_role.menu_id IN ?", menuIDs).
		Select("menu_role.menu_id, role.role_code").
		Find(&results).Error; err != nil {
		return nil, err
	}

	// 按菜单ID分组
	roleMap := make(map[string][]string, len(menuIDs))
	for _, r := range results {
		roleMap[r.MenuID] = append(roleMap[r.MenuID], r.RoleCode)
	}

	// 确保所有菜单ID都有对应的key（即使没有角色）
	for _, menuID := range menuIDs {
		if _, ok := roleMap[menuID]; !ok {
			roleMap[menuID] = []string{}
		}
	}

	return roleMap, nil
}
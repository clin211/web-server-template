package store

import (
	"context"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// MenuStore 定义了 menu 模块在 store 层所实现的方法.
type MenuStore interface {
	Create(ctx context.Context, obj *model.MenuM) error
	Update(ctx context.Context, obj *model.MenuM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.MenuM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.MenuM, error)

	MenuExpansion
}

// MenuExpansion 定义了菜单操作的附加方法.
type MenuExpansion interface {
	// GetByMenuCode 根据菜单编码获取菜单
	GetByMenuCode(ctx context.Context, menuCode string) (*model.MenuM, error)
	// ListTree 获取菜单树
	ListTree(ctx context.Context, opts *where.Options) ([]*model.MenuM, error)
	// GetChildren 获取子菜单列表
	GetChildren(ctx context.Context, parentID string) ([]*model.MenuM, error)
	// GetUserMenus 获取用户可见的菜单树
	GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error)
	// UpdateSortOrder 更新菜单排序
	UpdateSortOrder(ctx context.Context, menuID string, sortOrder int32) error
	// GetMenuAllowedRoles 获取菜单允许的角色代码列表
	GetMenuAllowedRoles(ctx context.Context, menuID string) ([]string, error)
}

// menuStore 是 MenuStore 接口的实现。
type menuStore struct {
	*genericstore.Store[model.MenuM]
	core *datastore
}

// 确保 menuStore 实现了 MenuStore 接口。
var _ MenuStore = (*menuStore)(nil)

// newMenuStore 创建 menuStore 的实例。
func newMenuStore(store *datastore) *menuStore {
	return &menuStore{
		Store: genericstore.NewStore[model.MenuM](store, storelogger.NewLogger()),
		core:  store,
	}
}

// GetByMenuCode 根据菜单编码获取菜单
func (s *menuStore) GetByMenuCode(ctx context.Context, menuCode string) (*model.MenuM, error) {
	var obj model.MenuM
	if err := s.core.DB(ctx, where.F("menu_code", menuCode).L(1)).First(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListTree 获取菜单树
func (s *menuStore) ListTree(ctx context.Context, opts *where.Options) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	// 按照父菜单和排序顺序获取所有菜单
	// 使用 NULLS FIRST 确保 NULL 值的 parent_id 排在最前面
	if err := s.core.DB(ctx, opts).
		Order("parent_id NULLS FIRST, sort_order ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// GetChildren 获取子菜单列表
func (s *menuStore) GetChildren(ctx context.Context, parentID string) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	query := s.core.DB(ctx)
	if parentID == "" {
		query = query.Where("parent_id IS NULL OR parent_id = ''")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	if err := query.Order("sort_order ASC").Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// GetUserMenus 获取用户可见的菜单树.
// 过滤逻辑：
// 1. 菜单状态必须启用且可见（status=0, visible=1）
// 2. 满足以下条件之一：
//    - 该菜单在 menu_role 表中无记录（对所有角色可见）
//    - 该菜单的 menu_role 记录中至少有一个角色是用户拥有的
//    - 该菜单为常量路由（constant=1）
func (s *menuStore) GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	// 获取用户拥有的角色ID列表
	var roleIDs []string
	if err := s.core.DB(ctx).
		Table("user_role").
		Select("role_id").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}

	// 如果用户没有任何角色，只返回常量路由
	if len(roleIDs) == 0 {
		if err := s.core.DB(ctx).
			Where("menu.status = ? AND menu.visible = ? AND menu.constant = ?", 0, 1, 1).
			Order("parent_id NULLS FIRST, sort_order ASC").
			Find(&menus).Error; err != nil {
			return nil, err
		}
		return menus, nil
	}

	// 查询满足以下条件的菜单：
	// 1. 状态启用且可见
	// 2. 软删除未删除
	// 3. 满足：
	//    - constant = 1（常量路由，对所有用户可见）
	//    - menu_role 表中无记录（对所有角色可见）
	//    - menu_role 表中有该用户的角色
	if err := s.core.DB(ctx).
		Where("menu.status = ? AND menu.visible = ?", 0, 1).
		Where("menu.deleted_at IS NULL").
		Where("(menu.constant = ? OR "+
			"menu_id NOT IN (SELECT menu_id FROM menu_role WHERE menu_id IS NOT NULL) OR "+
			"menu_id IN (SELECT menu_id FROM menu_role WHERE role_id IN ?))", 1, roleIDs).
		Order("parent_id NULLS FIRST, sort_order ASC").
		Find(&menus).Error; err != nil {
		return nil, err
	}

	return menus, nil
}

// UpdateSortOrder 更新菜单排序
func (s *menuStore) UpdateSortOrder(ctx context.Context, menuID string, sortOrder int32) error {
	return s.core.DB(ctx).Model(&model.MenuM{}).
		Where("menu_id = ?", menuID).
		Update("sort_order", sortOrder).Error
}

// GetMenuAllowedRoles 获取菜单允许的角色代码列表
func (s *menuStore) GetMenuAllowedRoles(ctx context.Context, menuID string) ([]string, error) {
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

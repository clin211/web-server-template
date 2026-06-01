package store

import (
	"context"
	"fmt"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"

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
	// ListTree 获取菜单树
	ListTree(ctx context.Context, opts *where.Options) ([]*model.MenuM, error)
	// GetChildren 获取子菜单列表
	GetChildren(ctx context.Context, parentID string) ([]*model.MenuM, error)
	// GetUserMenus 获取用户可见的菜单树
	GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error)
	// GetUserMenusWithRoles 获取用户可见的菜单树及角色映射
	// 返回 menus 列表和 menuID -> roles 映射
	GetUserMenusWithRoles(ctx context.Context, userID string) ([]*model.MenuM, map[string][]string, error)
	// GetConstantMenus 获取常量路由菜单
	GetConstantMenus(ctx context.Context) ([]*model.MenuM, error)
	// GetConstantMenusWithRoles 获取常量路由菜单及角色映射
	GetConstantMenusWithRoles(ctx context.Context) ([]*model.MenuM, map[string][]string, error)
	// UpdateSortOrder 更新单个菜单排序
	UpdateSortOrder(ctx context.Context, menuID string, sortOrder int32) error
	// BatchUpdateSortOrder 批量更新菜单排序
	BatchUpdateSortOrder(ctx context.Context, items map[string]int32) error
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

// ListTree 获取菜单树
func (s *menuStore) ListTree(ctx context.Context, opts *where.Options) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	var db *gorm.DB
	if opts != nil {
		db = s.core.DB(ctx, opts)
	} else {
		db = s.core.DB(ctx)
	}

	if err := db.Order("parent_id NULLS FIRST, sort_order ASC").Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("list menu tree: %w", err)
	}

	return menus, nil
}

// GetChildren 获取子菜单列表
func (s *menuStore) GetChildren(ctx context.Context, parentID string) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	opts := where.NewWhere().F("deleted_at", nil)
	if parentID == "" {
		opts.Q("parent_id IS NULL OR parent_id = ''")
	} else {
		opts.F("parent_id", parentID)
	}

	if err := s.core.DB(ctx, opts).Order("sort_order ASC").Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("get children of menu %s: %w", parentID, err)
	}

	return menus, nil
}

// GetUserMenus 获取用户可见的菜单树.
// 过滤逻辑：
// 1. 菜单状态必须启用且可见（status=0, visible=1）
// 2. 满足以下条件之一：
//    - 该菜单在 menu_role 表中无记录（对所有角色可见）
//    - 该菜单的 menu_role 记录中至少有一个角色是用户拥有的
// 3. 如果用户没有任何角色，返回空列表
func (s *menuStore) GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	// 获取用户拥有的角色ID列表
	var roleIDs []string
	if err := s.core.DB(ctx).
		Table("user_role").
		Select("role_id").
		Where("user_id = ?", userID).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, fmt.Errorf("get user roles for menu filter: %w", err)
	}

	// 如果用户没有任何角色，返回空列表
	if len(roleIDs) == 0 {
		return []*model.MenuM{}, nil
	}

	// 查询满足以下条件的菜单：
	// 1. 状态启用且可见
	// 2. 软删除未删除
	// 3. 满足：
	//    - menu_role 表中无记录（对所有角色可见）
	//    - menu_role 表中有该用户的角色
	if err := s.core.DB(ctx).
		Where("menu.status = ? AND menu.visible = ?", 0, 1).
		Where("menu.deleted_at IS NULL").
		Where("(menu_id NOT IN (SELECT menu_id FROM menu_role WHERE menu_id IS NOT NULL) OR "+
			"menu_id IN (SELECT menu_id FROM menu_role WHERE role_id IN ?))", roleIDs).
		Order("parent_id NULLS LAST, sort_order ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("get user menus: %w", err)
	}

	return menus, nil
}

// UpdateSortOrder 更新单个菜单排序
func (s *menuStore) UpdateSortOrder(ctx context.Context, menuID string, sortOrder int32) error {
	if err := s.core.DB(ctx).Model(&model.MenuM{}).
		Where("menu_id = ?", menuID).
		Update("sort_order", sortOrder).Error; err != nil {
		return fmt.Errorf("update sort order for menu %s: %w", menuID, err)
	}
	return nil
}

// BatchUpdateSortOrder 批量更新菜单排序，使用单条 SQL 语句.
func (s *menuStore) BatchUpdateSortOrder(ctx context.Context, items map[string]int32) error {
	if len(items) == 0 {
		return nil
	}

	// 构建 CASE WHEN 语句
	// UPDATE menu SET sort_order = CASE menu_id WHEN 'id1' THEN 1 WHEN 'id2' THEN 2 END WHERE menu_id IN ('id1', 'id2')
	var caseStmt string
	var args []any

	for menuID, sortOrder := range items {
		if caseStmt != "" {
			caseStmt += " "
		}
		caseStmt += "WHEN ? THEN ?"
		args = append(args, menuID, sortOrder)
	}

	menuIDs := make([]any, 0, len(items))
	for menuID := range items {
		menuIDs = append(menuIDs, menuID)
	}

	sql := fmt.Sprintf("UPDATE menu SET sort_order = CASE menu_id %s END WHERE menu_id IN ?", caseStmt)
	if err := s.core.DB(ctx).Exec(sql, args...).Error; err != nil {
		return fmt.Errorf("batch update sort order: %w", err)
	}

	return nil
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
		return nil, fmt.Errorf("get allowed roles for menu %s: %w", menuID, err)
	}

	return roleCodes, nil
}

// GetUserMenusWithRoles 获取用户可见的菜单树及角色映射.
// 返回 menus 列表和 menuID -> roles 映射，用于批量构建路由树.
func (s *menuStore) GetUserMenusWithRoles(ctx context.Context, userID string) ([]*model.MenuM, map[string][]string, error) {
	// 获取用户可见的菜单
	menus, err := s.GetUserMenus(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// 批量获取所有菜单的角色映射
	rolesMap, err := s.getMenuRolesMap(ctx, menus)
	if err != nil {
		return nil, nil, err
	}

	return menus, rolesMap, nil
}

// GetConstantMenus 获取常量路由菜单.
func (s *menuStore) GetConstantMenus(ctx context.Context) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	if err := s.core.DB(ctx).
		Where("constant = ? AND status = ? AND visible = ?", 1, 0, 1).
		Where("deleted_at IS NULL").
		Order("parent_id NULLS FIRST, sort_order ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("get constant menus: %w", err)
	}

	return menus, nil
}

// GetConstantMenusWithRoles 获取常量路由菜单及角色映射.
func (s *menuStore) GetConstantMenusWithRoles(ctx context.Context) ([]*model.MenuM, map[string][]string, error) {
	menus, err := s.GetConstantMenus(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 批量获取所有菜单的角色映射
	rolesMap, err := s.getMenuRolesMap(ctx, menus)
	if err != nil {
		return nil, nil, err
	}

	return menus, rolesMap, nil
}

// getMenuRolesMap 批量获取菜单的角色映射.
func (s *menuStore) getMenuRolesMap(ctx context.Context, menus []*model.MenuM) (map[string][]string, error) {
	if len(menus) == 0 {
		return make(map[string][]string), nil
	}

	// 收集所有菜单ID
	menuIDs := make([]string, len(menus))
	for i, menu := range menus {
		menuIDs[i] = menu.MenuID
	}

	// 批量查询菜单的角色映射
	type menuRole struct {
		MenuID   string
		RoleCode string
	}
	var results []menuRole
	if err := s.core.DB(ctx).
		Table("menu_role").
		Select("menu_role.menu_id, role.role_code").
		Joins("INNER JOIN role ON menu_role.role_id = role.role_id").
		Where("menu_role.menu_id IN ?", menuIDs).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("get menu roles map: %w", err)
	}

	// 构建映射
	rolesMap := make(map[string][]string)
	for _, mr := range results {
		rolesMap[mr.MenuID] = append(rolesMap[mr.MenuID], mr.RoleCode)
	}

	return rolesMap, nil
}
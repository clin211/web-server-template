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
	if err := s.core.DB(ctx, opts).
		Order("parent_id NULLS LAST, sort_order ASC").
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

// GetUserMenus 获取用户可见的菜单树
func (s *menuStore) GetUserMenus(ctx context.Context, userID string) ([]*model.MenuM, error) {
	var menus []*model.MenuM

	// 通过用户角色获取权限，再获取对应的菜单
	query := s.core.DB(ctx).
		Joins("INNER JOIN role_permission ON menu.permission_id = role_permission.permission_id").
		Joins("INNER JOIN user_role ON role_permission.role_id = user_role.role_id").
		Where("user_role.user_id = ?", userID).
		Where("menu.status = ? AND menu.visible = ?", 0, 1). // 0=启用, 1=可见
		Order("parent_id NULLS LAST, sort_order ASC")

	if err := query.Find(&menus).Error; err != nil {
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

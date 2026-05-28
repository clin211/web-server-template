package store

import (
	"context"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// PermissionStore 定义了 permission 模块在 store 层所实现的方法.
type PermissionStore interface {
	Create(ctx context.Context, obj *model.PermissionM) error
	Update(ctx context.Context, obj *model.PermissionM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.PermissionM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.PermissionM, error)

	PermissionExpansion
}

// PermissionExpansion 定义了权限操作的附加方法.
type PermissionExpansion interface {
	// GetByPermissionCode 根据权限编码获取权限
	GetByPermissionCode(ctx context.Context, permissionCode string) (*model.PermissionM, error)
	// ListTree 获取权限树
	ListTree(ctx context.Context, opts *where.Options) ([]*model.PermissionM, error)
	// GetChildren 获取子权限列表
	GetChildren(ctx context.Context, parentID string) ([]*model.PermissionM, error)
}

// permissionStore 是 PermissionStore 接口的实现。
type permissionStore struct {
	*genericstore.Store[model.PermissionM]
	core *datastore
}

// 确保 permissionStore 实现了 PermissionStore 接口。
var _ PermissionStore = (*permissionStore)(nil)

// newPermissionStore 创建 permissionStore 的实例。
func newPermissionStore(store *datastore) *permissionStore {
	return &permissionStore{
		Store: genericstore.NewStore[model.PermissionM](store, storelogger.NewLogger()),
		core:  store,
	}
}

// GetByPermissionCode 根据权限编码获取权限
func (s *permissionStore) GetByPermissionCode(ctx context.Context, permissionCode string) (*model.PermissionM, error) {
	var obj model.PermissionM
	if err := s.core.DB(ctx, where.F("permission_code", permissionCode).L(1)).First(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListTree 获取权限树
func (s *permissionStore) ListTree(ctx context.Context, opts *where.Options) ([]*model.PermissionM, error) {
	var permissions []*model.PermissionM

	// 首先获取所有权限
	if err := s.core.DB(ctx, opts).Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetChildren 获取子权限列表
func (s *permissionStore) GetChildren(ctx context.Context, parentID string) ([]*model.PermissionM, error) {
	var permissions []*model.PermissionM

	query := s.core.DB(ctx)
	if parentID == "" {
		query = query.Where("parent_id IS NULL OR parent_id = ''")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

package store

import (
	"context"
	"fmt"

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

// ListTree 获取权限树.
// 一次性获取所有权限数据，在应用层构建树形结构，避免 N+1 查询问题.
// opts 可指定 resource_type、status 等过滤条件.
func (s *permissionStore) ListTree(ctx context.Context, opts *where.Options) ([]*model.PermissionM, error) {
	var permissions []*model.PermissionM

	whereOpts := where.NewWhere().F("deleted_at", nil)
	if opts != nil {
		// 合并过滤条件
		for k, v := range opts.Filters {
			whereOpts.F(k, v)
		}
		for _, q := range opts.Queries {
			whereOpts.Q(q.Query, q.Args...)
		}
	}

	if err := s.core.DB(ctx, whereOpts).
		Order("parent_id NULLS FIRST, created_at ASC").
		Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("list permission tree: %w", err)
	}

	return permissions, nil
}

// GetChildren 获取子权限列表
func (s *permissionStore) GetChildren(ctx context.Context, parentID string) ([]*model.PermissionM, error) {
	var permissions []*model.PermissionM

	opts := where.NewWhere().F("deleted_at", nil)
	if parentID == "" {
		opts.Q("parent_id IS NULL OR parent_id = ''")
	} else {
		opts.F("parent_id", parentID)
	}

	if err := s.core.DB(ctx, opts).Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("get permission children: %w", err)
	}

	return permissions, nil
}
package store

import (
	"context"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"gorm.io/gorm"
)

// RoleStore 定义了 role 模块在 store 层所实现的方法.
type RoleStore interface {
	Create(ctx context.Context, obj *model.RoleM) error
	Update(ctx context.Context, obj *model.RoleM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.RoleM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.RoleM, error)

	RoleExpansion
}

// RoleExpansion 定义了角色操作的附加方法.
type RoleExpansion interface {
	// GetByRoleCode 根据角色编码获取角色
	GetByRoleCode(ctx context.Context, roleCode string) (*model.RoleM, error)
	// AssignPermissions 为角色分配权限
	AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	// GetPermissions 获取角色的权限列表
	GetPermissions(ctx context.Context, roleID string) ([]*model.PermissionM, error)
	// RemovePermissions 移除角色的所有权限
	RemovePermissions(ctx context.Context, roleID string) error
}

// roleStore 是 RoleStore 接口的实现。
type roleStore struct {
	*genericstore.Store[model.RoleM]
	core *datastore
}

// 确保 roleStore 实现了 RoleStore 接口。
var _ RoleStore = (*roleStore)(nil)

// newRoleStore 创建 roleStore 的实例。
func newRoleStore(store *datastore) *roleStore {
	return &roleStore{
		Store: genericstore.NewStore[model.RoleM](store, storelogger.NewLogger()),
		core:  store,
	}
}

// GetByRoleCode 根据角色编码获取角色
func (s *roleStore) GetByRoleCode(ctx context.Context, roleCode string) (*model.RoleM, error) {
	var obj model.RoleM
	if err := s.core.DB(ctx, where.F("role_code", roleCode).L(1)).First(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// AssignPermissions 为角色分配权限（覆盖模式）。
// 使用事务确保删除和插入操作的原子性，避免并发竞态条件。
func (s *roleStore) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	// 使用事务确保删除和插入操作的原子性
	return s.core.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除现有权限
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermissionM{}).Error; err != nil {
			return err
		}

		// 批量插入新权限
		if len(permissionIDs) > 0 {
			var rolePermissions []interface{}
			for _, permissionID := range permissionIDs {
				rolePermissions = append(rolePermissions, &model.RolePermissionM{
					RoleID:       roleID,
					PermissionID: permissionID,
				})
			}
			if err := tx.Create(rolePermissions).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetPermissions 获取角色的权限列表
func (s *roleStore) GetPermissions(ctx context.Context, roleID string) ([]*model.PermissionM, error) {
	var permissions []*model.PermissionM

	// 通过子查询获取角色的权限
	subQuery := s.core.DB(ctx).Table("role_permission").
		Select("permission_id").
		Where("role_id = ?", roleID)

	if err := s.core.DB(ctx).
		Where("permission_id IN (?)", subQuery).
		Find(&permissions).Error; err != nil {
		return nil, err
	}

	return permissions, nil
}

// RemovePermissions 移除角色的所有权限
func (s *roleStore) RemovePermissions(ctx context.Context, roleID string) error {
	return s.core.DB(ctx).Where("role_id = ?", roleID).Delete(&model.RolePermissionM{}).Error
}

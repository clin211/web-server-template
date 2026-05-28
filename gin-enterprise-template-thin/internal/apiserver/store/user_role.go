package store

import (
	"context"

	storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
	genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// UserRoleStore 定义了 user_role 模块在 store 层所实现的方法.
type UserRoleStore interface {
	// Create 创建用户角色关联
	Create(ctx context.Context, obj *model.UserRoleM) error
	// Delete 删除用户角色关联
	Delete(ctx context.Context, opts *where.Options) error
	// List 获取用户的角色列表
	List(ctx context.Context, opts *where.Options) ([]*model.UserRoleM, error)

	UserRoleExpansion
}

// UserRoleExpansion 定义了用户角色操作的附加方法.
type UserRoleExpansion interface {
	// AssignRoles 为用户分配角色（覆盖模式）
	AssignRoles(ctx context.Context, userID string, roleIDs []string) error
	// GetUserRoles 获取用户的角色列表（含角色详情）
	GetUserRoles(ctx context.Context, userID string) ([]*model.RoleM, error)
	// RemoveRole 从用户移除指定角色
	RemoveRole(ctx context.Context, userID, roleID string) error
	// RemoveAllRoles 移除用户的所有角色
	RemoveAllRoles(ctx context.Context, userID string) error
	// GetUserPermissions 获取用户的所有权限编码
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
}

// userRoleStore 是 UserRoleStore 接口的实现。
type userRoleStore struct {
	*genericstore.Store[model.UserRoleM]
	core *datastore
}

// 确保 userRoleStore 实现了 UserRoleStore 接口。
var _ UserRoleStore = (*userRoleStore)(nil)

// newUserRoleStore 创建 userRoleStore 的实例。
func newUserRoleStore(store *datastore) *userRoleStore {
	return &userRoleStore{
		Store: genericstore.NewStore[model.UserRoleM](store, storelogger.NewLogger()),
		core:  store,
	}
}

// AssignRoles 为用户分配角色（覆盖模式）
func (s *userRoleStore) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	// 先删除现有角色
	if err := s.RemoveAllRoles(ctx, userID); err != nil {
		return err
	}

	// 批量插入新角色
	if len(roleIDs) > 0 {
		var userRoles []interface{}
		for _, roleID := range roleIDs {
			userRoles = append(userRoles, &model.UserRoleM{
				UserID: userID,
				RoleID: roleID,
			})
		}
		if err := s.core.DB(ctx).Create(userRoles).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetUserRoles 获取用户的角色列表（含角色详情）
func (s *userRoleStore) GetUserRoles(ctx context.Context, userID string) ([]*model.RoleM, error) {
	var roles []*model.RoleM

	// 通过 user_role 关联表查询用户的角色
	if err := s.core.DB(ctx).
		Joins("INNER JOIN user_role ON role.role_id = user_role.role_id").
		Where("user_role.user_id = ?", userID).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// RemoveRole 从用户移除指定角色
func (s *userRoleStore) RemoveRole(ctx context.Context, userID, roleID string) error {
	return s.core.DB(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&model.UserRoleM{}).Error
}

// RemoveAllRoles 移除用户的所有角色
func (s *userRoleStore) RemoveAllRoles(ctx context.Context, userID string) error {
	return s.core.DB(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserRoleM{}).Error
}

// List 获取用户的角色关联列表
func (s *userRoleStore) List(ctx context.Context, opts *where.Options) ([]*model.UserRoleM, error) {
	var userRoles []*model.UserRoleM
	if err := s.core.DB(ctx, opts).Find(&userRoles).Error; err != nil {
		return nil, err
	}
	return userRoles, nil
}

// GetUserPermissions 获取用户的所有权限编码
func (s *userRoleStore) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	var permissionCodes []string

	// 通过用户角色 -> 角色权限 -> 权限的路径查询
	if err := s.core.DB(ctx).
		Table("permission").
		Joins("INNER JOIN role_permission ON permission.permission_id = role_permission.permission_id").
		Joins("INNER JOIN user_role ON role_permission.role_id = user_role.role_id").
		Where("user_role.user_id = ?", userID).
		Where("permission.status = ?", 0). // 0=启用
		Pluck("permission.permission_code", &permissionCodes).Error; err != nil {
		return nil, err
	}

	return permissionCodes, nil
}

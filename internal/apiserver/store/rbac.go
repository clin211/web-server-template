package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Role contains the public role identity and builtin protection flag.
type Role struct {
	RoleID  string `json:"role_id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

// RoleDetail includes the role's persisted permission bindings.
type RoleDetail struct {
	Role
	Permissions []string `json:"permissions"`
}

// UserRoles contains assigned roles and deduplicated effective permissions.
type UserRoles struct {
	UserID      string   `json:"user_id"`
	Roles       []Role   `json:"roles"`
	Permissions []string `json:"permissions"`
}

// Role loads an existing role with a stable, complete permission list.
func (s *AuthorizationStore) Role(ctx context.Context, id string) (RoleDetail, error) {
	result := RoleDetail{Permissions: []string{}}
	db := s.core.DB(ctx).WithContext(ctx)
	if err := db.Table("role").Select("role_id,name,builtin").Where("role_id=?", id).Take(&result.Role).Error; err != nil {
		return result, fmt.Errorf("load role: %w", err)
	}
	if err := db.Table("role_permission").Where("role_id=?", id).Order("permission_code").Pluck("permission_code", &result.Permissions).Error; err != nil {
		return result, fmt.Errorf("load role permissions: %w", err)
	}
	return result, nil
}

// UserRoles loads a user's assigned roles without requiring user management access.
func (s *AuthorizationStore) UserRoles(ctx context.Context, id string) (UserRoles, error) {
	result := UserRoles{UserID: id, Roles: []Role{}, Permissions: []string{}}
	db := s.core.DB(ctx).WithContext(ctx)
	var count int64
	if err := db.Table("user").Where("user_id=?", id).Count(&count).Error; err != nil {
		return result, fmt.Errorf("find user: %w", err)
	}
	if count == 0 {
		return result, gorm.ErrRecordNotFound
	}
	if err := db.Table("role r").Select("r.role_id,r.name,r.builtin").Joins("JOIN user_role ur ON ur.role_id=r.role_id").Where("ur.user_id=?", id).Order("r.role_id").Find(&result.Roles).Error; err != nil {
		return result, fmt.Errorf("load user roles: %w", err)
	}
	permissions, err := s.Permissions(ctx, id)
	if err != nil {
		return result, err
	}
	result.Permissions = permissions
	return result, nil
}

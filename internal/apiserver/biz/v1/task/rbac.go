package task

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
)

// Role returns the current permissions of a role to RBAC managers.
func (s *Service) Role(ctx context.Context, id string) (store.RoleDetail, error) {
	if err := s.Require(ctx, "rbac.manage", ""); err != nil {
		return store.RoleDetail{}, err
	}
	if id == "" || len(id) > 64 {
		return store.RoleDetail{}, errno.ErrInvalidArgument
	}
	role, err := s.auth.Role(ctx, id)
	return role, errno.FromStore(err)
}

// UserRoles returns assigned roles and effective grants to RBAC managers.
func (s *Service) UserRoles(ctx context.Context, id string) (store.UserRoles, error) {
	if err := s.Require(ctx, "rbac.manage", ""); err != nil {
		return store.UserRoles{}, err
	}
	if id == "" || len(id) > 32 {
		return store.UserRoles{}, errno.ErrInvalidArgument
	}
	roles, err := s.auth.UserRoles(ctx, id)
	return roles, errno.FromStore(err)
}

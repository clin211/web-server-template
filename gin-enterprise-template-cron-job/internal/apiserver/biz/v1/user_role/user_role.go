package user_role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authz"
)

// UserRoleBiz 定义处理用户角色请求所需的方法.
type UserRoleBiz interface {
	// AssignRolesToUser 为用户分配角色
	AssignRolesToUser(ctx context.Context, rq *v1.AssignRolesToUserRequest) (*v1.AssignRolesToUserResponse, error)
	// GetUserRoles 获取用户的角色和权限
	GetUserRoles(ctx context.Context, rq *v1.GetUserRolesRequest) (*v1.GetUserRolesResponse, error)
	// RemoveRoleFromUser 从用户移除角色
	RemoveRoleFromUser(ctx context.Context, rq *v1.RemoveRoleFromUserRequest) (*v1.RemoveRoleFromUserResponse, error)
}

// userRoleBiz 是 UserRoleBiz 接口的实现.
type userRoleBiz struct {
	store store.IStore
	authz *authz.Authz
}

// 确保 userRoleBiz 实现了 UserRoleBiz 接口.
var _ UserRoleBiz = (*userRoleBiz)(nil)

func New(store store.IStore, authz *authz.Authz) *userRoleBiz {
	return &userRoleBiz{store: store, authz: authz}
}

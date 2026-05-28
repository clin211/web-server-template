package role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authz"
)

// RoleBiz 定义处理角色请求所需的方法.
type RoleBiz interface {
	Create(ctx context.Context, rq *v1.CreateRoleRequest) (*v1.CreateRoleResponse, error)
	Update(ctx context.Context, rq *v1.UpdateRoleRequest) (*v1.UpdateRoleResponse, error)
	Delete(ctx context.Context, rq *v1.DeleteRoleRequest) (*v1.DeleteRoleResponse, error)
	Get(ctx context.Context, rq *v1.GetRoleRequest) (*v1.GetRoleResponse, error)
	List(ctx context.Context, rq *v1.ListRoleRequest) (*v1.ListRoleResponse, error)

	RoleExpansion
}

// RoleExpansion 定义角色操作的扩展方法.
type RoleExpansion interface {
	// AssignPermissionsToRole 为角色分配权限
	AssignPermissionsToRole(ctx context.Context, rq *v1.AssignPermissionsToRoleRequest) (*v1.AssignPermissionsToRoleResponse, error)
	// GetRolePermissions 获取角色的权限列表
	GetRolePermissions(ctx context.Context, rq *v1.GetRolePermissionsRequest) (*v1.GetRolePermissionsResponse, error)
}

// roleBiz 是 RoleBiz 接口的实现.
type roleBiz struct {
	store store.IStore
	authz *authz.Authz
}

// 确保 roleBiz 实现了 RoleBiz 接口.
var _ RoleBiz = (*roleBiz)(nil)

func New(store store.IStore, authz *authz.Authz) *roleBiz {
	return &roleBiz{store: store, authz: authz}
}

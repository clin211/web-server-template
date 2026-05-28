package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// PermissionBiz 定义处理权限请求所需的方法.
type PermissionBiz interface {
	Create(ctx context.Context, rq *v1.CreatePermissionRequest) (*v1.CreatePermissionResponse, error)
	Update(ctx context.Context, rq *v1.UpdatePermissionRequest) (*v1.UpdatePermissionResponse, error)
	Delete(ctx context.Context, rq *v1.DeletePermissionRequest) (*v1.DeletePermissionResponse, error)
	Get(ctx context.Context, rq *v1.GetPermissionRequest) (*v1.GetPermissionResponse, error)
	List(ctx context.Context, rq *v1.ListPermissionRequest) (*v1.ListPermissionResponse, error)

	PermissionExpansion
}

// PermissionExpansion 定义权限操作的扩展方法.
type PermissionExpansion interface {
	// ListPermissionTree 获取权限树
	ListPermissionTree(ctx context.Context, rq *v1.ListPermissionTreeRequest) (*v1.ListPermissionTreeResponse, error)
}

// permissionBiz 是 PermissionBiz 接口的实现.
type permissionBiz struct {
	store store.IStore
}

// 确保 permissionBiz 实现了 PermissionBiz 接口.
var _ PermissionBiz = (*permissionBiz)(nil)

func New(store store.IStore) *permissionBiz {
	return &permissionBiz{store: store}
}

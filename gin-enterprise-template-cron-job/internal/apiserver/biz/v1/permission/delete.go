package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// Delete 删除权限.
func (b *permissionBiz) Delete(ctx context.Context, rq *v1.DeletePermissionRequest) (*v1.DeletePermissionResponse, error) {
	// 检查是否有子权限
	children, err := b.store.Permission().GetChildren(ctx, rq.GetPermissionID())
	if err == nil && len(children) > 0 {
		return nil, errno.ErrPermissionHasChildren
	}

	if err := b.store.Permission().Delete(ctx, where.F("permission_id", rq.GetPermissionID())); err != nil {
		return nil, err
	}

	return &v1.DeletePermissionResponse{}, nil
}

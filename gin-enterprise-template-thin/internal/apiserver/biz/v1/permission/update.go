package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/jinzhu/copier"
)

// Update 更新权限.
func (b *permissionBiz) Update(ctx context.Context, rq *v1.UpdatePermissionRequest) (*v1.UpdatePermissionResponse, error) {
	permM, err := b.store.Permission().Get(ctx, where.F("permission_id", rq.GetPermissionID()).L(1))
	if err != nil {
		return nil, errno.ErrPermissionNotFound
	}

	// 使用 copier 更新字段
	if err := copier.CopyWithOption(permM, rq, copier.Option{IgnoreEmpty: true}); err != nil {
		return nil, err
	}

	if err := b.store.Permission().Update(ctx, permM); err != nil {
		return nil, err
	}

	return &v1.UpdatePermissionResponse{}, nil
}

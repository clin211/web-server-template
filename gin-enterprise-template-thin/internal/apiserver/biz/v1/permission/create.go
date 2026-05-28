package permission

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/jinzhu/copier"
)

// Create 创建权限.
func (b *permissionBiz) Create(ctx context.Context, rq *v1.CreatePermissionRequest) (*v1.CreatePermissionResponse, error) {
	var permM conversion.PermissionModel
	if err := copier.Copy(&permM, rq); err != nil {
		return nil, err
	}

	// 检查权限编码是否已存在
	if existingPerm, err := b.store.Permission().GetByPermissionCode(ctx, permM.PermissionCode); err == nil && existingPerm != nil {
		return nil, errno.ErrPermissionAlreadyExists
	}

	if err := b.store.Permission().Create(ctx, &permM); err != nil {
		return nil, err
	}

	return &v1.CreatePermissionResponse{PermissionID: permM.PermissionID}, nil
}

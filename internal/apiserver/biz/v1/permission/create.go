package permission

import (
	"context"
	"errors"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

// Create 创建权限.
func (b *permissionBiz) Create(ctx context.Context, rq *v1.CreatePermissionRequest) (*v1.CreatePermissionResponse, error) {
	var permM conversion.PermissionModel
	if err := copier.Copy(&permM, rq); err != nil {
		return nil, fmt.Errorf("copy permission request: %w", err)
	}

	// 检查权限编码是否已存在
	existingPerm, err := b.store.Permission().Get(ctx, where.NewWhere().F("permission_code", permM.PermissionCode).L(1))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check permission code: %w", err)
	}
	if existingPerm != nil {
		return nil, errno.ErrPermissionAlreadyExists
	}

	if err := b.store.Permission().Create(ctx, &permM); err != nil {
		return nil, fmt.Errorf("create permission: %w", err)
	}

	return &v1.CreatePermissionResponse{PermissionID: permM.PermissionID}, nil
}
package role

import (
	"context"
	"errors"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

// Update 更新角色.
func (b *roleBiz) Update(ctx context.Context, rq *v1.UpdateRoleRequest) (*v1.UpdateRoleResponse, error) {
	roleM, err := b.store.Role().Get(ctx, where.F("role_id", rq.GetRoleID()).L(1))
	if err != nil {
		// 区分"角色不存在"和"数据库错误"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errno.ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// 使用 copier 更新字段
	if err := copier.CopyWithOption(roleM, rq, copier.Option{IgnoreEmpty: true}); err != nil {
		return nil, fmt.Errorf("failed to copy update fields: %w", err)
	}

	if err := b.store.Role().Update(ctx, roleM); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return &v1.UpdateRoleResponse{}, nil
}

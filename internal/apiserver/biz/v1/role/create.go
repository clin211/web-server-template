package role

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

// Create 创建角色.
// 使用事务确保数据库操作和 Casbin 同步的原子性。
func (b *roleBiz) Create(ctx context.Context, rq *v1.CreateRoleRequest) (*v1.CreateRoleResponse, error) {
	var roleM conversion.RoleModel
	if err := copier.Copy(&roleM, rq); err != nil {
		return nil, fmt.Errorf("copy role request: %w", err)
	}

	// 检查角色编码是否已存在
	existingRole, err := b.store.Role().Get(ctx, where.NewWhere().F("role_code", roleM.RoleCode).L(1))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check role code: %w", err)
	}
	if existingRole != nil {
		return nil, errno.ErrRoleAlreadyExists
	}

	// 使用事务确保数据库操作和 Casbin 同步的原子性
	var createResp *v1.CreateRoleResponse
	err = b.store.TX(ctx, func(txCtx context.Context) error {
		// 创建角色到数据库
		if err := b.store.Role().Create(txCtx, &roleM); err != nil {
			return fmt.Errorf("create role: %w", err)
		}

		// 同步到 Casbin
		if err := b.syncRoleToCasbin(txCtx, roleM.RoleCode); err != nil {
			return fmt.Errorf("sync role to casbin: %w", err)
		}

		createResp = &v1.CreateRoleResponse{RoleID: roleM.RoleID}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createResp, nil
}
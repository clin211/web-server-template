package role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"log/slog"
)

// Delete 删除角色.
func (b *roleBiz) Delete(ctx context.Context, rq *v1.DeleteRoleRequest) (*v1.DeleteRoleResponse, error) {
	roleID := rq.GetRoleID()

	// 获取角色信息
	roleM, err := b.store.Role().Get(ctx, where.F("role_id", roleID).L(1))
	if err != nil {
		return nil, errno.ErrRoleNotFound
	}

	if err := b.store.Role().Delete(ctx, where.F("role_id", roleID)); err != nil {
		return nil, err
	}

	// 从 Casbin 中删除该角色的策略
	casbinRole := "role::" + roleM.RoleCode
	if _, err := b.authz.RemoveFilteredGroupingPolicy(0, casbinRole); err != nil {
		slog.WarnContext(ctx, "Failed to remove grouping policy", "error", err)
	}

	return &v1.DeleteRoleResponse{}, nil
}

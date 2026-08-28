package user

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Logout 撤销当前用户已签发的令牌。
// token_version 与审计记录在同一事务中写入，认证中间件会拒绝旧版本令牌。
func (b *userBiz) Logout(ctx context.Context, _ *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	userID := contextx.UserID(ctx)
	if userID == "" {
		return nil, errno.ErrUnauthenticated
	}
	if err := b.store.TX(ctx, func(txCtx context.Context) error {
		if err := b.store.User().Update(txCtx, userID, map[string]any{
			"token_version": gorm.Expr("token_version + 1"),
		}); err != nil {
			return fmt.Errorf("increment token version: %w", err)
		}
		if err := b.store.AuditLog().Create(txCtx, &model.AuditLogM{
			UserID:    userID,
			Action:    "auth.logout",
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("create logout audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("logout transaction: %w", err)
	}
	return &v1.LogoutResponse{}, nil
}

package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/pkg/authn"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// ChangePassword 实现 UserBiz 接口中的 ChangePassword 方法.
func (b *userBiz) ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error) {
	if contextx.UserID(ctx) == "" {
		return nil, errno.ErrUnauthenticated
	}
	if rq.GetUserID() != contextx.UserID(ctx) {
		return nil, errno.ErrPermissionDenied
	}
	userM, err := b.store.User().Get(ctx, where.F("user_id", contextx.UserID(ctx)))
	if err != nil {
		// 认证后用户被删除等场景：按资源不存在处理，而非 500 系统错误。
		return nil, errno.FromStore(err)
	}

	if err := authn.Compare(userM.Password, rq.GetOldPassword()); err != nil {
		slog.ErrorContext(ctx, "Failed to compare password", "error", err)
		return nil, errno.ErrPasswordInvalid
	}

	encryptedPassword, err := authn.Encrypt(rq.GetNewPassword())
	if err != nil {
		slog.ErrorContext(ctx, "Failed to encrypt password", "error", err)
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}
	if err := b.store.TX(ctx, func(txCtx context.Context) error {
		if err := b.store.User().Update(txCtx, userM.UserID, map[string]any{
			"password":      encryptedPassword,
			"token_version": gorm.Expr("token_version + 1"),
		}); err != nil {
			return fmt.Errorf("update user password: %w", err)
		}
		if err := b.store.AuditLog().Create(txCtx, &model.AuditLogM{UserID: contextx.UserID(ctx), Action: "user.change_password", Resource: &userM.UserID, CreatedAt: time.Now().UTC()}); err != nil {
			return fmt.Errorf("create password change audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("change password transaction: %w", err)
	}
	// token_version 与密码在同一事务内递增；认证中间件会拒绝此前签发的令牌。

	return &v1.ChangePasswordResponse{}, nil
}

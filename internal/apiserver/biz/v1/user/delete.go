package user

import (
	"context"
	"fmt"
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Delete 实现 UserBiz 接口中的 Delete 方法.
func (b *userBiz) Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.F("user_id", rq.GetUserID()))
	if err != nil {
		// 资源不存在应表现为业务错误（404 语义），而非 500 系统错误。
		return nil, errno.FromStore(err)
	}
	if err := b.store.TX(ctx, func(txCtx context.Context) error {
		if err := b.store.User().Delete(txCtx, where.F("user_id", userM.UserID)); err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		if err := b.store.AuditLog().Create(txCtx, &model.AuditLogM{UserID: contextx.UserID(ctx), Action: "user.delete", Resource: &userM.UserID, CreatedAt: time.Now().UTC()}); err != nil {
			return fmt.Errorf("create user delete audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("delete user transaction: %w", err)
	}

	return &v1.DeleteUserResponse{}, nil
}

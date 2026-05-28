package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// ChangePassword 实现 UserBiz 接口中的 ChangePassword 方法.
func (b *userBiz) ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error) {
	userM, err := b.store.User().Get(ctx, where.T(ctx))
	if err != nil {
		return nil, err
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
	userM.Password = encryptedPassword
	if err := b.store.User().Update(ctx, userM); err != nil {
		return nil, err
	}

	return &v1.ChangePasswordResponse{}, nil
}

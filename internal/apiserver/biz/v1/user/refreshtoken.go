package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/token"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// RefreshToken 用于刷新用户的身份验证令牌.
// 当用户的令牌即将过期时，可以调用此方法生成新的访问令牌和刷新令牌.
// 返回与 Login 相同的 LoginResponse 格式.
func (b *userBiz) RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.LoginResponse, error) {
	accessToken, refreshToken, accessExpireAt, _, err := token.Sign(contextx.UserID(ctx))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to sign token", "error", err)
		return nil, errno.ErrSignToken
	}

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     accessExpireAt.Format(time.RFC3339),
	}, nil
}

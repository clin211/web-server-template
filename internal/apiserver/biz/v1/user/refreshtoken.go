package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// RefreshToken 用于刷新用户的身份验证令牌.
// 当用户的令牌即将过期时，可以调用此方法生成新的访问令牌和刷新令牌.
// 返回与 Login 相同的 LoginResponse 格式.
func (b *userBiz) RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.LoginResponse, error) {
	// rq 不提供身份来源：RefreshAuthnMiddleware 已验证 refresh token，并把可信 user_id
	// 写入 context；Biz 不能从未经认证的请求体或路径参数签发新 token。
	accessToken, refreshToken, accessExpireAt, _, err := b.tokenManager.RotateRefreshToken(
		ctx,
		contextx.AccessToken(ctx),
		contextx.UserID(ctx),
		contextx.TokenVersion(ctx),
	)
	if err != nil {
		slog.WarnContext(ctx, "Failed to rotate refresh token", "error", err)
		return nil, errno.ErrTokenInvalid
	}

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     accessExpireAt.Format(time.RFC3339),
	}, nil
}

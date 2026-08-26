package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// dummyPasswordHash 只用于让不存在用户和错误密码都执行一次 bcrypt 比较，降低
// 基于响应耗时枚举用户名的可行性；它不对应任何可登录账号。
const dummyPasswordHash = "$2a$10$IhKAl47y0wMFlBgYRHVy2uRLBxcuPgOged2Qrk4EXFkzEFztOVLZ2"

// Login 实现 UserBiz 接口中的 Login 方法.
func (b *userBiz) Login(ctx context.Context, rq *v1.LoginRequest) (*v1.LoginResponse, error) {
	// 获取登录用户的所有信息
	whr := where.F("username", rq.GetUsername())
	userM, err := b.store.User().Get(ctx, whr)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Compare 的 mismatch 是预期结果；调用它只是为了靠近真实用户失败路径的成本。
			if compareErr := authn.Compare(dummyPasswordHash, rq.GetPassword()); compareErr == nil {
				slog.WarnContext(ctx, "Dummy login password unexpectedly matched")
			}
			b.recordLogin(ctx, rq.GetUsername(), false, "invalid_credentials")
			return nil, errno.ErrUnauthenticated
		}
		slog.ErrorContext(ctx, "Failed to load login user", "error", err)
		b.recordLogin(ctx, rq.GetUsername(), false, "authentication_unavailable")
		return nil, fmt.Errorf("load login user: %w", err)
	}

	// 对比传入的明文密码和数据库中已加密过的密码是否匹配
	if err := authn.Compare(userM.Password, rq.GetPassword()); err != nil {
		slog.WarnContext(ctx, "Login rejected due to invalid credentials")
		b.recordLogin(ctx, rq.GetUsername(), false, "invalid_credentials")
		return nil, errno.ErrUnauthenticated
	}
	if userM.Status != 0 {
		b.recordLogin(ctx, rq.GetUsername(), false, "user_disabled")
		return nil, errno.ErrUnauthenticated
	}

	// 如果匹配成功，说明登录成功，签发 access token 和 refresh token 并返回
	accessToken, refreshToken, accessExpireAt, _, err := b.tokenManager.SignWithVersion(userM.UserID, userM.TokenVersion)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to sign token", "error", err)
		b.recordLogin(ctx, rq.GetUsername(), false, "token_sign_failed")
		return nil, errno.ErrSignToken
	}

	// last_login_at 是 best-effort 运营元数据；认证和 token 已成功后，不应因该次辅助
	// 写入失败而让客户端误以为登录失败并重复提交凭据。
	if err := b.store.User().UpdateLastLoginAt(ctx, userM.UserID, time.Now()); err != nil {
		slog.WarnContext(ctx, "Failed to update last login time", "userID", userM.UserID, "error", err)
	}
	b.recordLogin(ctx, rq.GetUsername(), true, "")

	return &v1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     accessExpireAt.Format(time.RFC3339),
	}, nil
}

func (b *userBiz) recordLogin(ctx context.Context, username string, success bool, reason string) {
	username = truncateLoginField(username, 255)
	address := truncateLoginField(contextx.ClientAddress(ctx), 64)
	userAgent := truncateLoginField(contextx.UserAgent(ctx), 2000)
	record := &model.UserLoginLogM{
		Username:  optionalLoginField(username),
		IPAddress: optionalLoginField(address),
		UserAgent: optionalLoginField(userAgent),
		Status:    success,
		CreatedAt: time.Now().UTC(),
	}
	if reason != "" {
		record.ErrorMessage = &reason
	}
	if err := b.store.UserLoginLog().Create(ctx, record); err != nil {
		slog.WarnContext(ctx, "Failed to persist login security audit", "error", err)
	}
}

func optionalLoginField(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func truncateLoginField(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
}

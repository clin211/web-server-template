package gin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/clin211/gin-enterprise-template/pkg/token"
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
)

// UserRetriever 是用于根据用户名获取用户的接口。
type UserRetriever interface {
	// GetUser 根据用户 ID 获取用户信息
	GetUser(ctx context.Context, userID string) (*model.UserM, error)
}

// AuthnMiddleware 是一个认证中间件，用于从 gin.Context 中提取 token 并验证 token 是否合法。
// 只接受 Access Token（token_type="access"）。
func AuthnMiddleware(retriever UserRetriever) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析 JWT Token
		userID, err := token.ParseRequest(c)
		if err != nil {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid.WithMessage(err.Error()))
			c.Abort()
			return
		}

		slog.Info("Token parsing successful", "userID", userID)

		user, err := retriever.GetUser(c, userID)
		if err != nil {
			core.WriteResponse(c, nil, errno.ErrUnauthenticated.WithMessage(err.Error()))
			c.Abort()
			return
		}

		ctx := contextx.WithUserID(c.Request.Context(), user.UserID)
		ctx = contextx.WithUsername(ctx, user.Username)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RefreshAuthnMiddleware 是一个专门用于刷新令牌的认证中间件。
// 只接受 Refresh Token（token_type="refresh"）。
func RefreshAuthnMiddleware(retriever UserRetriever) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization header 获取 token
		header := c.Request.Header.Get("Authorization")
		if header == "" {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid.WithMessage("authorization header is empty"))
			c.Abort()
			return
		}

		// 解析 Bearer token
		var tokenString string
		n, err := fmt.Sscanf(header, "Bearer %s", &tokenString)
		if err != nil || n != 1 {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid.WithMessage("invalid authorization header format"))
			c.Abort()
			return
		}

		// 验证是 refresh token 并解析 userID
		userID, err := token.ParseRefreshToken(tokenString)
		if err != nil {
			slog.ErrorContext(c, "Failed to parse refresh token", "error", err)
			core.WriteResponse(c, nil, errno.ErrTokenInvalid.WithMessage(err.Error()))
			c.Abort()
			return
		}

		slog.Info("Refresh token parsing successful", "userID", userID)

		user, err := retriever.GetUser(c, userID)
		if err != nil {
			core.WriteResponse(c, nil, errno.ErrUnauthenticated.WithMessage(err.Error()))
			c.Abort()
			return
		}

		ctx := contextx.WithUserID(c.Request.Context(), user.UserID)
		ctx = contextx.WithUsername(ctx, user.Username)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

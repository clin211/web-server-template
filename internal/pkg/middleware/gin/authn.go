package gin

import (
	"context"
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
func AuthnMiddleware(retriever UserRetriever, tokenManager *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokenManager == nil {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}
		// 解析 JWT Token
		userID, tokenVersion, err := tokenManager.ParseRequestWithVersion(c)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "Failed to parse access token", "error", err)
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}

		if !loadAuthenticatedUser(c, retriever, userID, tokenVersion) {
			return
		}
		c.Next()
	}
}

// RefreshAuthnMiddleware 是一个专门用于刷新令牌的认证中间件。
// 只接受 Refresh Token（token_type="refresh"）。
func RefreshAuthnMiddleware(retriever UserRetriever, tokenManager *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokenManager == nil {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}
		// 从 Authorization header 获取 token
		header := c.Request.Header.Get("Authorization")
		if header == "" {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}

		// 解析 Bearer token
		tokenString, err := token.ParseAuthorizationHeader(header)
		if err != nil {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}

		// 验证是 refresh token 并解析 userID
		userID, tokenVersion, err := tokenManager.ParseRefreshTokenWithVersion(tokenString)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "Failed to parse refresh token", "error", err)
			core.WriteResponse(c, nil, errno.ErrTokenInvalid)
			c.Abort()
			return
		}

		if !loadAuthenticatedUser(c, retriever, userID, tokenVersion) {
			return
		}
		ctx := contextx.WithAccessToken(c.Request.Context(), tokenString)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func loadAuthenticatedUser(c *gin.Context, retriever UserRetriever, userID string, tokenVersion int64) bool {
	user, err := retriever.GetUser(c.Request.Context(), userID)
	if err != nil || user == nil || user.UserID != userID {
		slog.WarnContext(c.Request.Context(), "Failed to retrieve authenticated user", "userID", userID, "error", err)
		core.WriteResponse(c, nil, errno.ErrUnauthenticated)
		c.Abort()
		return false
	}
	if user.Status != 0 {
		slog.WarnContext(c.Request.Context(), "Disabled user attempted authentication", "userID", userID)
		core.WriteResponse(c, nil, errno.ErrUserDisabled)
		c.Abort()
		return false
	}
	if user.TokenVersion != tokenVersion {
		slog.WarnContext(c.Request.Context(), "Stale token version rejected", "userID", userID)
		core.WriteResponse(c, nil, errno.ErrUnauthenticated)
		c.Abort()
		return false
	}

	ctx := contextx.WithUserID(c.Request.Context(), user.UserID)
	ctx = contextx.WithUsername(ctx, user.Username)
	ctx = contextx.WithTokenVersion(ctx, tokenVersion)
	c.Request = c.Request.WithContext(ctx)
	return true
}

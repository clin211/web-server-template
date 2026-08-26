package gin

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"time"

	ginframework "github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/clin211/gin-enterprise-template/pkg/core"
)

// LoginRateLimiter 定义跨实例登录尝试计数能力。
type LoginRateLimiter interface {
	AllowLogin(ctx context.Context, identity string, maximum int64, window time.Duration) (bool, error)
}

// LoginRateLimitMiddleware 按直连客户端地址限制登录尝试。代理部署必须在受信任
// 网关上执行等价限流；本中间件刻意不信任可伪造的 X-Forwarded-For。
func LoginRateLimitMiddleware(limiter LoginRateLimiter, maximum int64, window time.Duration) ginframework.HandlerFunc {
	return func(c *ginframework.Context) {
		if limiter == nil {
			core.WriteResponse(c, nil, errno.ErrServiceUnavailable)
			c.Abort()
			return
		}
		identity := c.Request.RemoteAddr
		if host, _, err := net.SplitHostPort(identity); err == nil {
			identity = host
		}
		allowed, err := limiter.AllowLogin(c.Request.Context(), identity, maximum, window)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "Login rate limiter unavailable", "error", err)
			core.WriteResponse(c, nil, errno.ErrServiceUnavailable)
			c.Abort()
			return
		}
		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(max(int64(window.Seconds()), 1), 10))
			core.WriteResponse(c, nil, errno.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}

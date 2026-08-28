package gin

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
)

// validRequestID 约束客户端回传的 X-Request-ID：仅接受受限长度与字符集的值
// （兼容带/不带连字符的 UUID）。客户端可控的任意内容若不加校验，会原样流入
// 日志、链路追踪关联与响应头，形成日志注入面，因此不合法的值一律重新生成。
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// newRequestID 生成时间有序的 UUID v7（RFC 9562）：日志按 ID 排序近似请求
// 到达顺序，未来若落库建索引也不会像随机 v4 那样造成 B-tree 写热点。
// NewV7 在 crypto/rand 故障时返回错误，此时退回 v4，避免在中间件里 panic。
func newRequestID() string {
	if newID, err := uuid.NewV7(); err == nil {
		return newID.String()
	}
	return uuid.NewString()
}

// RequestIDMiddleware 是一个 Gin 中间件，用于在每个 HTTP 请求的上下文和
// 响应中注入 `x-request-id` 键值对。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取 `x-request-id`；缺失或格式非法时生成新的 UUID
		requestID := c.Request.Header.Get(known.XRequestID)

		if !validRequestID.MatchString(requestID) {
			requestID = newRequestID()
		}

		// 将 RequestID 保存到 context.Context 中，以便后续程序使用
		ctx := contextx.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// 将 RequestID 保存到 HTTP 返回头中，Header 的键为 `x-request-id`
		c.Writer.Header().Set(known.XRequestID, requestID)

		// 继续处理请求
		c.Next()
	}
}

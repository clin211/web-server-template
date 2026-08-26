package gin

import (
	"net/http"

	ginframework "github.com/gin-gonic/gin"
)

// CORS 创建仅允许指定来源的跨域请求中间件。
func CORS(allowedOrigins []string) ginframework.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *ginframework.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		c.Writer.Header().Add("Vary", "Origin")
		if _, ok := allowed[origin]; !ok {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

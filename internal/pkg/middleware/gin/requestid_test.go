package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
)

// TestRequestIDMiddleware 表格驱动验证客户端回传 X-Request-ID 的接受规则：
// 仅受限长度与字符集的值被采纳，其余一律重新生成，防止日志注入。
func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		inbound  string
		wantKept bool // true 表示期望原样采纳客户端值
	}{
		{name: "标准 UUID 被采纳", inbound: "550e8400-e29b-41d4-a716-446655440000", wantKept: true},
		{name: "无连字符 UUID 被采纳", inbound: "550e8400e29b41d4a716446655440000", wantKept: true},
		{name: "合法短 ID 被采纳", inbound: "req-1234", wantKept: true},
		{name: "空值重新生成", inbound: "", wantKept: false},
		{name: "过短值重新生成", inbound: "abc", wantKept: false},
		{name: "超长值重新生成", inbound: "0123456789012345678901234567890123456789012345678901234567890123456789", wantKept: false},
		{name: "含空白字符重新生成", inbound: "550e8400 e29b", wantKept: false},
		{name: "含换行注入重新生成", inbound: "550e8400\ne29b", wantKept: false},
		{name: "含特殊符号重新生成", inbound: "req; DROP TABLE logs", wantKept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.inbound != "" {
				c.Request.Header.Set("X-Request-ID", tt.inbound)
			}

			RequestIDMiddleware()(c)

			got := w.Header().Get("X-Request-ID")
			if tt.wantKept {
				assert.Equal(t, tt.inbound, got)
			} else {
				assert.NotEqual(t, tt.inbound, got)
				assert.NotEmpty(t, got)
				// 重新生成的值必须是合法 UUID，且为时间有序的 v7
				parsed, err := uuid.Parse(got)
				assert.NoError(t, err)
				assert.Equal(t, uuid.Version(7), parsed.Version())
			}
			// 请求 context 中保存的值必须与响应头一致
			assert.Equal(t, got, contextx.RequestID(c.Request.Context()))
		})
	}
}

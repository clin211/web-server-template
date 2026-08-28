package handler

import (
	"log/slog"
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/clin211/gin-enterprise-template/pkg/version"
	"github.com/gin-gonic/gin"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// Healthz 服务健康检查.
func (h *Handler) Healthz(c *gin.Context) {
	slog.InfoContext(c.Request.Context(), "Healthz handler is called", "method", "Healthz", "status", "healthy")
	// 通过 version 包获取版本信息；必须传指针：值类型无法通过 proto.Message
	// 断言，会退化到 encoding/json 并丢失 status/message 等零值字段。
	core.WriteResponse(c, &v1.HealthzResponse{
		Status:    v1.ServiceStatus_Healthy,
		Version:   version.Get().Text(),
		Timestamp: time.Now().Format(time.DateTime),
	}, nil)
}

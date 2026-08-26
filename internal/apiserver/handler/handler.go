package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/biz"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/validation"
)

// Handler 实现 HTTP 服务。
type Handler struct {
	biz biz.IBiz
	val *validation.Validator
	mws []gin.HandlerFunc
}

// NewHandler 创建 Handler 的新实例。
func NewHandler(biz biz.IBiz, val *validation.Validator, mws ...gin.HandlerFunc) *Handler {
	return &Handler{biz: biz, val: val, mws: mws}
}

// InstallAll 显式安装当前服务支持的资源路由。
func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
	installUserRoutes(v1, h)
}

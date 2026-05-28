package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/biz"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/validation"
)

// Handler 实现 gRPC 服务。
type Handler struct {
	biz biz.IBiz
	val *validation.Validator
	mws []gin.HandlerFunc
}

type Registrar func(v1 *gin.RouterGroup, h *Handler)

var registrars []Registrar

// NewHandler 创建 Handler 的新实例。
func NewHandler(biz biz.IBiz, val *validation.Validator, mws ...gin.HandlerFunc) *Handler {
	return &Handler{biz: biz, val: val, mws: mws}
}

func Register(r Registrar) {
	registrars = append(registrars, r)
}

func (h *Handler) InstallAll(v1 *gin.RouterGroup) {
	for _, r := range registrars {
		r(v1, h)
	}
}

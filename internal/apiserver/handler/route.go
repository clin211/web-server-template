package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/pkg/core"
)

func init() {
	Register(func(v1 *gin.RouterGroup, h *Handler) {
		// 路由相关路由
		rg := v1.Group("/route")
		rg.Use(h.mws...)
		rg.GET("", h.GetUserRoutes)           // 获取用户可访问的路由
		rg.GET("/constant", h.GetConstantRoutes) // 获取常量路由
	})
}

// GetUserRoutes 获取用户可访问的路由树.
// 包含用户有权限访问的所有菜单路由（常量路由 + 动态路由）。
// 常量路由（constant=1）不参与权限过滤，对所有用户可见。
func (h *Handler) GetUserRoutes(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.MenuV1().GetUserRoutes, h.val.ValidateGetUserRoutesRequest)
}

// GetConstantRoutes 获取常量路由.
// 常量路由是不参与权限过滤的路由，对所有用户可见。
// 通常包括：root、login、not-found、403、404、500 等。
func (h *Handler) GetConstantRoutes(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.MenuV1().GetConstantRoutes, h.val.ValidateGetConstantRoutesRequest)
}

package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/pkg/core"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func init() {
	Register(func(rg *gin.RouterGroup, handler *Handler) {
		// 路由相关路由
		rg2 := rg.Group("/route")
		rg2.Use(handler.mws...)
		rg2.GET("/constant", handler.GetConstantRoutes) // 获取常量路由
		rg2.GET("", handler.GetUserRoutes)            // 获取用户可访问的路由
	})
}

// GetConstantRoutes 获取常量路由.
// 常量路由是不参与权限过滤的路由，对所有用户可见。
func (h *Handler) GetConstantRoutes(c *gin.Context) {
	resp, err := h.biz.MenuV1().GetConstantRoutes(c.Request.Context(), &v1.GetConstantRoutesRequest{})
	if err != nil {
		core.WriteResponse(c, nil, err)
		return
	}
	core.WriteResponse(c, resp, nil)
}

// GetUserRoutes 获取用户可访问的路由树.
// 包含用户有权限访问的所有菜单路由（常量路由 + 动态路由）。
func (h *Handler) GetUserRoutes(c *gin.Context) {
	resp, err := h.biz.MenuV1().GetUserRoutes(c.Request.Context(), &v1.GetUserRoutesRequest{})
	if err != nil {
		core.WriteResponse(c, nil, err)
		return
	}
	core.WriteResponse(c, resp, nil)
}

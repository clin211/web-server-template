package handler

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		// 权限相关路由
		rg := v1.Group("/permissions")
		rg.Use(handler.mws...)
		rg.POST("", handler.CreatePermission)           // 创建权限
		rg.PUT(":permissionID", handler.UpdatePermission) // 更新权限
		rg.DELETE(":permissionID", handler.DeletePermission) // 删除权限
		rg.GET(":permissionID", handler.GetPermission)   // 查询权限详情
		rg.GET("", handler.ListPermission)              // 查询权限列表
		rg.GET("/tree", handler.ListPermissionTree)     // 获取权限树
	})
}

// CreatePermission 创建新权限.
func (h *Handler) CreatePermission(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.PermissionV1().Create, h.val.ValidateCreatePermissionRequest)
}

// UpdatePermission 更新权限信息.
func (h *Handler) UpdatePermission(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.PermissionV1().Update, h.val.ValidateUpdatePermissionRequest)
}

// DeletePermission 删除权限.
func (h *Handler) DeletePermission(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.PermissionV1().Delete, h.val.ValidateDeletePermissionRequest)
}

// GetPermission 获取权限信息.
func (h *Handler) GetPermission(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.PermissionV1().Get, h.val.ValidateGetPermissionRequest)
}

// ListPermission 列出权限信息.
func (h *Handler) ListPermission(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.PermissionV1().List, h.val.ValidateListPermissionRequest)
}

// ListPermissionTree 获取权限树.
func (h *Handler) ListPermissionTree(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.PermissionV1().ListPermissionTree, h.val.ValidateListPermissionTreeRequest)
}

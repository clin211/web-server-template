package handler

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		// 角色相关路由
		rg := v1.Group("/roles")
		rg.Use(handler.mws...)
		rg.POST("", handler.CreateRole)                                  // 创建角色
		rg.PUT(":roleID", handler.UpdateRole)                            // 更新角色
		rg.DELETE(":roleID", handler.DeleteRole)                         // 删除角色
		rg.GET(":roleID", handler.GetRole)                               // 查询角色详情
		rg.GET("", handler.ListRole)                                     // 查询角色列表
		rg.POST(":roleID/permissions", handler.AssignPermissionsToRole)  // 为角色分配权限
		rg.GET(":roleID/permissions", handler.GetRolePermissions)        // 获取角色的权限列表
	})
}

// CreateRole 创建新角色.
func (h *Handler) CreateRole(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.RoleV1().Create, h.val.ValidateCreateRoleRequest)
}

// UpdateRole 更新角色信息.
func (h *Handler) UpdateRole(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.RoleV1().Update, h.val.ValidateUpdateRoleRequest)
}

// DeleteRole 删除角色.
func (h *Handler) DeleteRole(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.RoleV1().Delete, h.val.ValidateDeleteRoleRequest)
}

// GetRole 获取角色信息.
func (h *Handler) GetRole(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.RoleV1().Get, h.val.ValidateGetRoleRequest)
}

// ListRole 列出角色信息.
func (h *Handler) ListRole(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.RoleV1().List, h.val.ValidateListRoleRequest)
}

// AssignPermissionsToRole 为角色分配权限.
func (h *Handler) AssignPermissionsToRole(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.RoleV1().AssignPermissionsToRole, h.val.ValidateAssignPermissionsToRoleRequest)
}

// GetRolePermissions 获取角色的权限列表（树形）.
func (h *Handler) GetRolePermissions(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.RoleV1().GetRolePermissions, h.val.ValidateGetRolePermissionsRequest)
}

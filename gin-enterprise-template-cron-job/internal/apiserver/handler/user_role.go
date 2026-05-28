package handler

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		// 用户角色相关路由（注册在 /users 路由组下）
		// 注意：这些路由会与 user.go 中的路由合并
	})
}

// AssignRolesToUser 为用户分配角色（覆盖模式）.
func (h *Handler) AssignRolesToUser(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserRoleV1().AssignRolesToUser, h.val.ValidateAssignRolesToUserRequest)
}

// GetUserRoles 获取用户的角色和权限.
func (h *Handler) GetUserRoles(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.UserRoleV1().GetUserRoles, h.val.ValidateGetUserRolesRequest)
}

// RemoveRoleFromUser 从用户移除角色.
func (h *Handler) RemoveRoleFromUser(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.UserRoleV1().RemoveRoleFromUser, h.val.ValidateRemoveRoleFromUserRequest)
}

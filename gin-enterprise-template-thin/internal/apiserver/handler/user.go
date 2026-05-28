package handler

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		// 用户相关路由
		rg := v1.Group("/users")
		rg.POST("", handler.CreateUser) // 创建用户。这里要注意：创建用户是不用进行认证和授权的
		rg.Use(handler.mws...)
		rg.PUT(":userID/change-password", handler.ChangePassword) // 修改用户密码
		rg.PUT(":userID", handler.UpdateUser)                     // 更新用户信息
		rg.DELETE(":userID", handler.DeleteUser)                  // 删除用户
		rg.GET(":userID", handler.GetUser)                        // 查询用户详情
		rg.GET("", handler.ListUser)                              // 查询用户列表
		rg.GET("/menu-tree", handler.GetUserMenuTree)             // 获取用户可见的菜单树

		// 用户角色相关路由
		rg.POST(":userID/roles", handler.AssignRolesToUser)       // 为用户分配角色
		rg.GET(":userID/roles", handler.GetUserRoles)             // 获取用户的角色和权限
		rg.DELETE(":userID/roles/:roleID", handler.RemoveRoleFromUser) // 从用户移除角色
	})
}

// Login 用户登录并返回 JWT Token.
func (h *Handler) Login(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserV1().Login, h.val.ValidateLoginRequest)
}

// RefreshToken 刷新 JWT Token.
// refresh token 从 Authorization header 获取，request body 可以为空.
func (h *Handler) RefreshToken(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.UserV1().RefreshToken)
}

// ChangePassword 修改用户密码.
func (h *Handler) ChangePassword(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserV1().ChangePassword, h.val.ValidateChangePasswordRequest)
}

// CreateUser 创建新用户.
func (h *Handler) CreateUser(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserV1().Create, h.val.ValidateCreateUserRequest)
}

// UpdateUser 更新用户信息.
func (h *Handler) UpdateUser(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserV1().Update, h.val.ValidateUpdateUserRequest)
}

// DeleteUser 删除用户.
func (h *Handler) DeleteUser(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.UserV1().Delete, h.val.ValidateDeleteUserRequest)
}

// GetUser 获取用户信息.
func (h *Handler) GetUser(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.UserV1().Get, h.val.ValidateGetUserRequest)
}

// ListUser 列出用户信息.
func (h *Handler) ListUser(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.UserV1().List, h.val.ValidateListUserRequest)
}

// GetUserMenuTree 获取用户可见的菜单树.
func (h *Handler) GetUserMenuTree(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.MenuV1().GetUserMenuTree, h.val.ValidateGetUserMenuTreeRequest)
}

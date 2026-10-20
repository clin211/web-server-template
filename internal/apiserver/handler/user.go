package handler

import (
	"net"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func installUserRoutes(v1 *gin.RouterGroup, handler *Handler) {
	// 用户管理按数据库中的实时授权校验；个人资料允许本人访问。
	rg := v1.Group("/users", handler.mws...)
	rg.POST("", handler.CreateUser)
	rg.PUT(":userID/change-password", handler.ChangePassword) // 修改用户密码
	rg.PUT(":userID", handler.UpdateUser)                     // 更新用户信息
	rg.DELETE(":userID", handler.DeleteUser)                  // 删除用户
	rg.GET(":userID", handler.GetUser)                        // 查询用户详情
	rg.GET("", handler.ListUser)                              // 查询用户列表
}

// Login 用户登录并返回 JWT Token.
func (h *Handler) Login(c *gin.Context) {
	address := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(address); err == nil {
		address = host
	}
	ctx := contextx.WithClientAddress(c.Request.Context(), address)
	ctx = contextx.WithUserAgent(ctx, c.Request.UserAgent())
	c.Request = c.Request.WithContext(ctx)
	core.HandleJSONRequest(c, h.biz.UserV1().Login, h.val.ValidateLoginRequest)
}

// RefreshToken 刷新 JWT Token.
// refresh token 从 Authorization header 获取，request body 可以为空.
func (h *Handler) RefreshToken(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.UserV1().RefreshToken)
}

// Info 获取当前用户信息。
func (h *Handler) Info(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.UserV1().Info)
}

// Logout 撤销当前用户的所有已签发令牌。
func (h *Handler) Logout(c *gin.Context) {
	core.HandleNoBodyRequest(c, h.biz.UserV1().Logout)
}

// ChangePassword 修改用户密码.
func (h *Handler) ChangePassword(c *gin.Context) {
	core.HandleUriJSONRequest(c, h.biz.UserV1().ChangePassword, h.val.ValidateChangePasswordRequest)
}

// CreateUser 创建新用户.
func (h *Handler) CreateUser(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.UserV1().Create, h.val.ValidateCreateUserRequest)
}

// UpdateUser 更新用户信息.
func (h *Handler) UpdateUser(c *gin.Context) {
	core.HandleUriJSONRequest(c, h.biz.UserV1().Update, h.val.ValidateUpdateUserRequest)
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

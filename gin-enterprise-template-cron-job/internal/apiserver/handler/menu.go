package handler

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/gin-gonic/gin"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		// 菜单相关路由
		rg := v1.Group("/menus")
		rg.Use(handler.mws...)
		rg.POST("", handler.CreateMenu)               // 创建菜单
		rg.PUT(":menuID", handler.UpdateMenu)         // 更新菜单
		rg.DELETE(":menuID", handler.DeleteMenu)      // 删除菜单
		rg.GET(":menuID", handler.GetMenu)            // 查询菜单详情
		rg.GET("", handler.ListMenu)                  // 查询菜单列表
		rg.GET("/tree", handler.ListMenuTree)         // 获取菜单树
	})
}

// CreateMenu 创建新菜单.
func (h *Handler) CreateMenu(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.MenuV1().Create, h.val.ValidateCreateMenuRequest)
}

// UpdateMenu 更新菜单信息.
func (h *Handler) UpdateMenu(c *gin.Context) {
	core.HandleJSONRequest(c, h.biz.MenuV1().Update, h.val.ValidateUpdateMenuRequest)
}

// DeleteMenu 删除菜单.
func (h *Handler) DeleteMenu(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.MenuV1().Delete, h.val.ValidateDeleteMenuRequest)
}

// GetMenu 获取菜单信息.
func (h *Handler) GetMenu(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.MenuV1().Get, h.val.ValidateGetMenuRequest)
}

// ListMenu 列出菜单信息.
func (h *Handler) ListMenu(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.MenuV1().List, h.val.ValidateListMenuRequest)
}

// ListMenuTree 获取菜单树.
func (h *Handler) ListMenuTree(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.MenuV1().ListMenuTree, h.val.ValidateListMenuTreeRequest)
}

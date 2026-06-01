package conversion

import (
	"fmt"
	"sort"

	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/clin211/gin-enterprise-template/pkg/ptr"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// MenuModel 菜单模型别名
type MenuModel = model.MenuM

// MenuModelToMenuV1 将模型层的 MenuM 转换为 Protobuf 层的 Menu.
func MenuModelToMenuV1(menuModel *model.MenuM) *v1.Menu {
	if menuModel == nil {
		return &v1.Menu{}
	}

	return &v1.Menu{
		MenuID:       menuModel.MenuID,
		ParentID:     derefString(menuModel.ParentID),
		MenuName:     menuModel.MenuName,
		MenuCode:     menuModel.MenuCode,
		MenuType:     menuModel.MenuType,
		I18NKey:     menuModel.I18nKey,
		Icon:         menuModel.Icon,
		LocalIcon:    menuModel.LocalIcon,
		IconFontSize: ptr.Map(menuModel.IconFontSize, func(v int) int32 { return int32(v) }),
		Path:         menuModel.Path,
		Component:    menuModel.Component,
		PermissionID: menuModel.PermissionID,
		SortOrder:    menuModel.SortOrder,
		Visible:      int32(menuModel.Visible),
		Status:       int32(menuModel.Status),
		Constant:     int32(menuModel.Constant),
		ActiveMenu:   menuModel.ActiveMenu,
		HideInMenu:   int32(menuModel.HideInMenu),
		KeepAlive:    int32(menuModel.KeepAlive),
		Href:         menuModel.Href,
		CreatedAt:    menuModel.CreatedAt.Unix(),
		UpdatedAt:    menuModel.UpdatedAt.Unix(),
	}
}

// MenuV1ToMenuModel 将 Protobuf 层的 Menu 转换为模型层的 MenuM.
func MenuV1ToMenuModel(protoMenu *v1.Menu) (*model.MenuM, error) {
	var menuModel model.MenuM
	if err := core.CopyWithConverters(&menuModel, protoMenu); err != nil {
		return nil, fmt.Errorf("convert proto menu to model: %w", err)
	}
	return &menuModel, nil
}

// MenuModelListToMenuV1List 将菜单模型列表转换为 Protobuf 列表.
func MenuModelListToMenuV1List(menus []*model.MenuM) []*v1.Menu {
	result := make([]*v1.Menu, len(menus))
	for i, menu := range menus {
		result[i] = MenuModelToMenuV1(menu)
	}
	return result
}

// menuTreeNodeFields 将菜单模型转换为 MenuTreeNode 的字段映射.
func menuTreeNodeFields(menu *model.MenuM) v1.MenuTreeNode {
	// 空字符串的 parentID 替换为 "0"，确保 protojson 序列化时字段被包含
	parentID := "0"
	if menu.ParentID != nil && *menu.ParentID != "" {
		parentID = *menu.ParentID
	}
	return v1.MenuTreeNode{
		MenuID:       menu.MenuID,
		ParentID:     parentID,
		MenuName:     menu.MenuName,
		MenuCode:     menu.MenuCode,
		MenuType:     menu.MenuType,
		I18NKey:      menu.I18nKey,
		Icon:         menu.Icon,
		LocalIcon:    menu.LocalIcon,
		IconFontSize: ptr.Map(menu.IconFontSize, func(v int) int32 { return int32(v) }),
		Path:         menu.Path,
		Component:    menu.Component,
		PermissionID: menu.PermissionID,
		SortOrder:    menu.SortOrder,
		Visible:      int32(menu.Visible),
		Status:       int32(menu.Status),
		Constant:     int32(menu.Constant),
		ActiveMenu:   menu.ActiveMenu,
		HideInMenu:   int32(menu.HideInMenu),
		KeepAlive:    int32(menu.KeepAlive),
		Href:         menu.Href,
		CreatedAt:    menu.CreatedAt.Unix(),
		UpdatedAt:    menu.UpdatedAt.Unix(),
	}
}

// MenuModelToMenuTreeNodeV1 将模型层的 MenuM 转换为 Protobuf 层的 MenuTreeNode.
func MenuModelToMenuTreeNodeV1(menuModel *model.MenuM) *v1.MenuTreeNode {
	if menuModel == nil {
		return &v1.MenuTreeNode{}
	}
	node := menuTreeNodeFields(menuModel)
	return &node
}

// MenuModelListToMenuTreeV1 将菜单模型列表转换为菜单树.
func MenuModelListToMenuTreeV1(menus []*model.MenuM) []*v1.MenuTreeNode {
	if len(menus) == 0 {
		return nil
	}

	treeMap := make(map[string]*v1.MenuTreeNode, len(menus))
	roots := make([]*v1.MenuTreeNode, 0, len(menus))

	// 创建所有节点
	for _, menu := range menus {
		node := menuTreeNodeFields(menu)
		treeMap[menu.MenuID] = &node
	}

	// 构建树关系
	for _, menu := range menus {
		node := treeMap[menu.MenuID]
		if menu.ParentID == nil || *menu.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := treeMap[*menu.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}

// MenuToRouteProto 将菜单模型转换为路由 proto.
// 该函数用于 GetUserRoutes 接口，将菜单转换为前端期望的 MenuRoute 格式.
func MenuToRouteProto(menu *model.MenuM, roles []string) *v1.MenuRoute {
	if menu == nil {
		return &v1.MenuRoute{}
	}

	return &v1.MenuRoute{
		Id:       menu.MenuID,
		Name:     menu.MenuCode,
		Path:     menu.Path,
		Component: menu.Component,
		Meta: &v1.MenuRouteMeta{
			Title:        &menu.MenuName,
			I18NKey:       menu.I18nKey,
			Icon:          menu.Icon,
			LocalIcon:     menu.LocalIcon,
			IconFontSize:  ptr.Map(menu.IconFontSize, func(v int) int32 { return int32(v) }),
			Order:         &menu.SortOrder,
			ActiveMenu:    menu.ActiveMenu,
			HideInMenu:    menu.HideInMenu == 1,
			KeepAlive:     menu.KeepAlive == 1,
			Constant:      menu.Constant == 1,
			Href:          menu.Href,
			Roles:         roles,
		},
	}
}

// BuildMenuTree 将扁平菜单列表转换为嵌套路由树结构.
// menus: 扁平菜单列表
// getAllowedRoles: 获取菜单允许角色的函数
// 注意: 循环内调用 getAllowedRoles 会导致 N+1 查询问题，建议使用 BuildMenuTreeWithRoles
func BuildMenuTree(menus []*model.MenuM, getAllowedRoles func(menuID string) ([]string, error)) ([]*v1.MenuRoute, error) {
	childrenMap := make(map[string][]*model.MenuM)

	// 按父ID分组
	for _, menu := range menus {
		parentID := ""
		if menu.ParentID != nil {
			parentID = *menu.ParentID
		}
		childrenMap[parentID] = append(childrenMap[parentID], menu)
	}

	// 递归构建树
	var buildRoutes func(parentID string) ([]*v1.MenuRoute, error)
	buildRoutes = func(parentID string) ([]*v1.MenuRoute, error) {
		children := childrenMap[parentID]
		routes := make([]*v1.MenuRoute, 0, len(children))
		for _, menu := range children {
			// 获取菜单允许的角色
			roles, err := getAllowedRoles(menu.MenuID)
			if err != nil {
				return nil, err
			}
			route := MenuToRouteProto(menu, roles)
			// 递归构建子路由
			childRoutes, err := buildRoutes(menu.MenuID)
			if err != nil {
				return nil, err
			}
			route.Children = childRoutes
			routes = append(routes, route)
		}
		return routes, nil
	}

	return buildRoutes("")
}

// BuildMenuTreeWithRoles 将扁平菜单列表转换为嵌套路由树结构（优化版）.
// 使用迭代方式构建，避免深层递归导致的栈溢出风险.
// menus: 扁平菜单列表
// rolesMap: 预加载的菜单角色映射，避免 N+1 查询
func BuildMenuTreeWithRoles(menus []*model.MenuM, rolesMap map[string][]string) []*v1.MenuRoute {
	if len(menus) == 0 {
		return []*v1.MenuRoute{}
	}

	childrenMap := make(map[string][]*model.MenuM)
	nodes := make(map[string]*v1.MenuRoute, len(menus))
	var roots []*v1.MenuRoute

	// 第一遍：建立节点映射和子节点分组
	for _, menu := range menus {
		parentID := ""
		if menu.ParentID != nil {
			parentID = *menu.ParentID
		}
		childrenMap[parentID] = append(childrenMap[parentID], menu)

		// 从预加载的映射中获取角色
		roles := rolesMap[menu.MenuID]
		nodes[menu.MenuID] = MenuToRouteProto(menu, roles)
	}

	// 第二遍：建立父子关系（从根向叶子遍历）
	for _, node := range nodes {
		menu := findMenuByMenuID(menus, node.Id)
		if menu == nil {
			continue
		}
		if menu.ParentID == nil || *menu.ParentID == "" {
			// 根节点
			roots = append(roots, node)
		} else if parent, ok := nodes[*menu.ParentID]; ok {
			// 添加到父节点
			parent.Children = append(parent.Children, node)
		} else {
			// 孤儿节点（父节点不存在），作为根节点处理
			roots = append(roots, node)
		}
	}

	// 对根节点按 sortOrder 排序
	sortMenuRoutes(roots)
	return roots
}

// findMenuByMenuID 根据菜单ID查找菜单
func findMenuByMenuID(menus []*model.MenuM, menuID string) *model.MenuM {
	for _, menu := range menus {
		if menu.MenuID == menuID {
			return menu
		}
	}
	return nil
}

// sortMenuRoutes 对菜单路由列表按 sortOrder 排序
func sortMenuRoutes(routes []*v1.MenuRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].Meta.GetOrder() < routes[j].Meta.GetOrder()
	})
	for _, route := range routes {
		if len(route.Children) > 0 {
			sortMenuRoutes(route.Children)
		}
	}
}

// UniqueStrings 去重字符串切片.
func UniqueStrings(strs []string) []string {
	if len(strs) == 0 {
		return strs
	}
	seen := make(map[string]struct{}, len(strs))
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}
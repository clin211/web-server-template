package conversion

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"

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
		ParentID:     stringPtrValue(menuModel.ParentID),
		MenuName:     menuModel.MenuName,
		MenuCode:     menuModel.MenuCode,
		MenuType:     menuModel.MenuType,
		Icon:         stringPtrValue(menuModel.Icon),
		Path:         stringPtrValue(menuModel.Path),
		Component:    stringPtrValue(menuModel.Component),
		PermissionID: stringPtrValue(menuModel.PermissionID),
		SortOrder:    menuModel.SortOrder,
		Visible:      int32(menuModel.Visible),
		Status:       int32(menuModel.Status),
		CreatedAt:    menuModel.CreatedAt.Unix(),
		UpdatedAt:    menuModel.UpdatedAt.Unix(),
	}
}

// MenuV1ToMenuModel 将 Protobuf 层的 Menu 转换为模型层的 MenuM.
func MenuV1ToMenuModel(protoMenu *v1.Menu) *model.MenuM {
	var menuModel model.MenuM
	_ = core.CopyWithConverters(&menuModel, protoMenu)
	return &menuModel
}

// MenuModelListToMenuV1List 将菜单模型列表转换为 Protobuf 列表.
func MenuModelListToMenuV1List(menus []*model.MenuM) []*v1.Menu {
	result := make([]*v1.Menu, len(menus))
	for i, menu := range menus {
		result[i] = MenuModelToMenuV1(menu)
	}
	return result
}

// MenuModelToMenuTreeNodeV1 将模型层的 MenuM 转换为 Protobuf 层的 MenuTreeNode.
func MenuModelToMenuTreeNodeV1(menuModel *model.MenuM) *v1.MenuTreeNode {
	return &v1.MenuTreeNode{Menu: MenuModelToMenuV1(menuModel)}
}

// MenuModelListToMenuTreeV1 将菜单模型列表转换为菜单树.
func MenuModelListToMenuTreeV1(menus []*model.MenuM) []*v1.MenuTreeNode {
	treeMap := make(map[string]*v1.MenuTreeNode, len(menus))
	roots := make([]*v1.MenuTreeNode, 0, len(menus))

	for _, menu := range menus {
		node := &v1.MenuTreeNode{Menu: MenuModelToMenuV1(menu)}
		treeMap[menu.MenuID] = node
	}

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

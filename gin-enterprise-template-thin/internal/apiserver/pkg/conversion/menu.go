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
	var protoMenu v1.Menu
	_ = core.CopyWithConverters(&protoMenu, menuModel)
	return &protoMenu
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
	for i, m := range menus {
		result[i] = MenuModelToMenuV1(m)
	}
	return result
}

// MenuModelToMenuTreeNodeV1 将模型层的 MenuM 转换为 Protobuf 层的 MenuTreeNode.
func MenuModelToMenuTreeNodeV1(menuModel *model.MenuM) *v1.MenuTreeNode {
	return &v1.MenuTreeNode{
		Menu: MenuModelToMenuV1(menuModel),
	}
}

// MenuModelListToMenuTreeV1 将菜单模型列表转换为菜单树.
func MenuModelListToMenuTreeV1(menus []*model.MenuM) []*v1.MenuTreeNode {
	treeMap := make(map[string]*v1.MenuTreeNode)
	var roots []*v1.MenuTreeNode

	// 第一遍：创建所有节点
	for _, m := range menus {
		node := &v1.MenuTreeNode{
			Menu: MenuModelToMenuV1(m),
		}
		treeMap[m.MenuID] = node
	}

	// 第二遍：构建树形结构
	for _, m := range menus {
		node := treeMap[m.MenuID]
		if m.ParentID == nil || *m.ParentID == "" {
			roots = append(roots, node)
		} else {
			if parent, ok := treeMap[*m.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return roots
}

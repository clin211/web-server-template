package conversion

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// PermissionModel 权限模型别名
type PermissionModel = model.PermissionM

// PermissionModelToPermissionV1 将模型层的 PermissionM 转换为 Protobuf 层的 Permission.
func PermissionModelToPermissionV1(permissionModel *model.PermissionM) *v1.Permission {
	var protoPermission v1.Permission
	_ = core.CopyWithConverters(&protoPermission, permissionModel)
	return &protoPermission
}

// PermissionV1ToPermissionModel 将 Protobuf 层的 Permission 转换为模型层的 PermissionM.
func PermissionV1ToPermissionModel(protoPermission *v1.Permission) *model.PermissionM {
	var permissionModel model.PermissionM
	_ = core.CopyWithConverters(&permissionModel, protoPermission)
	return &permissionModel
}

// PermissionModelListToPermissionV1List 将权限模型列表转换为 Protobuf 列表.
func PermissionModelListToPermissionV1List(permissions []*model.PermissionM) []*v1.Permission {
	result := make([]*v1.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = PermissionModelToPermissionV1(p)
	}
	return result
}

// PermissionModelToPermissionTreeV1 将模型层的 PermissionM 转换为 Protobuf 层的 PermissionTree.
func PermissionModelToPermissionTreeV1(permissionModel *model.PermissionM) *v1.PermissionTree {
	return &v1.PermissionTree{
		PermissionID:   permissionModel.PermissionID,
		PermissionName: permissionModel.PermissionName,
		PermissionCode: permissionModel.PermissionCode,
		ResourceType:   permissionModel.ResourceType,
		ResourcePath:   safeString(permissionModel.ResourcePath),
		Action:         permissionModel.Action,
	}
}

// safeString 安全地获取字符串值，处理 nil 指针.
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PermissionModelListToPermissionTreeV1 将权限模型列表转换为权限树.
func PermissionModelListToPermissionTreeV1(permissions []*model.PermissionM, assignedIDs map[string]bool) []*v1.PermissionTree {
	treeMap := make(map[string]*v1.PermissionTree)
	var roots []*v1.PermissionTree

	// 第一遍：创建所有节点
	for _, p := range permissions {
		node := &v1.PermissionTree{
			PermissionID:   p.PermissionID,
			PermissionName: p.PermissionName,
			PermissionCode: p.PermissionCode,
			ResourceType:   p.ResourceType,
			ResourcePath:   safeString(p.ResourcePath),
			Action:         p.Action,
			Assigned:       assignedIDs[p.PermissionID],
		}
		treeMap[p.PermissionID] = node
	}

	// 第二遍：构建树形结构
	for _, p := range permissions {
		node := treeMap[p.PermissionID]
		if p.ParentID == nil || *p.ParentID == "" {
			roots = append(roots, node)
		} else {
			if parent, ok := treeMap[*p.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return roots
}

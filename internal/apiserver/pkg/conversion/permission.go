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
	if permissionModel == nil {
		return &v1.Permission{}
	}

	return &v1.Permission{
		PermissionID:   permissionModel.PermissionID,
		PermissionName: permissionModel.PermissionName,
		PermissionCode: permissionModel.PermissionCode,
		ResourceType:   permissionModel.ResourceType,
		ResourcePath:   stringPtrValue(permissionModel.ResourcePath),
		Action:         permissionModel.Action,
		Description:    stringPtrValue(permissionModel.Description),
		ParentID:       stringPtrValue(permissionModel.ParentID),
		Path:           stringPtrValue(permissionModel.Path),
		Status:         int32(permissionModel.Status),
		CreatedAt:      permissionModel.CreatedAt.Unix(),
		UpdatedAt:      permissionModel.UpdatedAt.Unix(),
	}
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
	for i, permission := range permissions {
		result[i] = PermissionModelToPermissionV1(permission)
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
		ResourcePath:   stringPtrValue(permissionModel.ResourcePath),
		Action:         permissionModel.Action,
	}
}

// PermissionModelListToPermissionTreeV1 将权限模型列表转换为权限树.
func PermissionModelListToPermissionTreeV1(permissions []*model.PermissionM, assignedIDs map[string]bool) []*v1.PermissionTree {
	treeMap := make(map[string]*v1.PermissionTree, len(permissions))
	roots := make([]*v1.PermissionTree, 0, len(permissions))

	for _, permission := range permissions {
		node := &v1.PermissionTree{
			PermissionID:   permission.PermissionID,
			PermissionName: permission.PermissionName,
			PermissionCode: permission.PermissionCode,
			ResourceType:   permission.ResourceType,
			ResourcePath:   stringPtrValue(permission.ResourcePath),
			Action:         permission.Action,
			Assigned:       assignedIDs[permission.PermissionID],
		}
		treeMap[permission.PermissionID] = node
	}

	for _, permission := range permissions {
		node := treeMap[permission.PermissionID]
		if permission.ParentID == nil || *permission.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := treeMap[*permission.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	return roots
}

// PermissionTreeNodeFields 将权限模型转换为 PermissionTreeNode 的字段映射.
func PermissionTreeNodeFields(perm *model.PermissionM) v1.PermissionTreeNode {
	return v1.PermissionTreeNode{
		Permission: PermissionModelToPermissionV1(perm),
	}
}

// PermissionModelToPermissionTreeNodeV1 将模型层的 PermissionM 转换为 Protobuf 层的 PermissionTreeNode.
func PermissionModelToPermissionTreeNodeV1(permModel *model.PermissionM) *v1.PermissionTreeNode {
	if permModel == nil {
		return &v1.PermissionTreeNode{}
	}
	node := PermissionTreeNodeFields(permModel)
	return &node
}

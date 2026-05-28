package conversion

import (
	"github.com/clin211/gin-enterprise-template/pkg/core"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// RoleModel 角色模型别名
type RoleModel = model.RoleM

// RoleModelToRoleV1 将模型层的 RoleM 转换为 Protobuf 层的 Role.
func RoleModelToRoleV1(roleModel *model.RoleM) *v1.Role {
	if roleModel == nil {
		return &v1.Role{}
	}

	return &v1.Role{
		RoleID:      roleModel.RoleID,
		RoleName:    roleModel.RoleName,
		RoleCode:    roleModel.RoleCode,
		Description: stringPtrValue(roleModel.Description),
		Status:      int32(roleModel.Status),
		SortOrder:   roleModel.SortOrder,
		CreatedAt:   roleModel.CreatedAt.Unix(),
		UpdatedAt:   roleModel.UpdatedAt.Unix(),
	}
}

// RoleV1ToRoleModel 将 Protobuf 层的 Role 转换为模型层的 RoleM.
func RoleV1ToRoleModel(protoRole *v1.Role) *model.RoleM {
	var roleModel model.RoleM
	_ = core.CopyWithConverters(&roleModel, protoRole)
	return &roleModel
}

// RoleModelListToRoleV1List 将角色模型列表转换为 Protobuf 列表.
func RoleModelListToRoleV1List(roles []*model.RoleM) []*v1.Role {
	result := make([]*v1.Role, len(roles))
	for i, role := range roles {
		result[i] = RoleModelToRoleV1(role)
	}
	return result
}

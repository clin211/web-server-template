package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateRoleRules() genericvalidation.Rules {
	return genericvalidation.Rules{
		"RoleID": func(value any) error {
			str, ok := value.(string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("roleID must be a string")
			}
			if str == "" {
				return errno.ErrInvalidArgument.WithMessage("roleID cannot be empty")
			}
			return nil
		},
		"RoleName": func(value any) error {
			name, ok := value.(string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("roleName must be a string")
			}
			if len(name) == 0 || len(name) > 50 {
				return errno.ErrInvalidArgument.WithMessage("roleName must be between 1 and 50 characters")
			}
			return nil
		},
		"RoleCode": func(value any) error {
			code, ok := value.(string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("roleCode must be a string")
			}
			if len(code) == 0 || len(code) > 50 {
				return errno.ErrInvalidArgument.WithMessage("roleCode must be between 1 and 50 characters")
			}
			if !isValidUsername(code) {
				return errno.ErrInvalidArgument.WithMessage("roleCode can only contain letters, numbers and underscores")
			}
			return nil
		},
		"Description": func(value any) error {
			if value == nil {
				return nil
			}
			desc, ok := value.(string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("description must be a string")
			}
			if len(desc) > 200 {
				return errno.ErrInvalidArgument.WithMessage("description must be less than 200 characters")
			}
			return nil
		},
		"Status": func(value any) error {
			status, ok := value.(int32)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("status must be an int32")
			}
			if status != 0 && status != 1 {
				return errno.ErrInvalidArgument.WithMessage("status must be 0 (enabled) or 1 (disabled)")
			}
			return nil
		},
		"SortOrder": func(value any) error {
			return nil // 排序可以是任意整数
		},
		"Mode": func(value any) error {
			if value == nil {
				return nil
			}
			mode, ok := value.(string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("mode must be a string")
			}
			if mode != "override" && mode != "append" {
				return errno.ErrInvalidArgument.WithMessage("mode must be 'override' or 'append'")
			}
			return nil
		},
		"PermissionIDs": func(value any) error {
			return nil // 权限 ID 列表，无需额外验证
		},
		"PageSize": func(value any) error {
			size, ok := value.(int64)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("pageSize must be an int64")
			}
			if size <= 0 || size > 100 {
				return errno.ErrInvalidArgument.WithMessage("pageSize must be between 1 and 100")
			}
			return nil
		},
	}
}

// ValidateCreateRoleRequest 校验创建角色请求.
func (v *Validator) ValidateCreateRoleRequest(ctx context.Context, rq *v1.CreateRoleRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateRoleRules())
}

// ValidateUpdateRoleRequest 校验更新角色请求.
func (v *Validator) ValidateUpdateRoleRequest(ctx context.Context, rq *v1.UpdateRoleRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateRoleRules(), "RoleID")
}

// ValidateDeleteRoleRequest 校验删除角色请求.
func (v *Validator) ValidateDeleteRoleRequest(ctx context.Context, rq *v1.DeleteRoleRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateRoleRules())
}

// ValidateGetRoleRequest 校验获取角色请求.
func (v *Validator) ValidateGetRoleRequest(ctx context.Context, rq *v1.GetRoleRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateRoleRules())
}

// ValidateListRoleRequest 校验角色列表请求.
func (v *Validator) ValidateListRoleRequest(ctx context.Context, rq *v1.ListRoleRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateRoleRules(), "PageToken", "PageSize", "Status", "Keyword")
}

// ValidateAssignPermissionsToRoleRequest 校验分配权限请求.
func (v *Validator) ValidateAssignPermissionsToRoleRequest(ctx context.Context, rq *v1.AssignPermissionsToRoleRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateRoleRules(), "RoleID", "PermissionIDs", "Mode")
}

// ValidateGetRolePermissionsRequest 校验获取角色权限请求.
func (v *Validator) ValidateGetRolePermissionsRequest(ctx context.Context, rq *v1.GetRolePermissionsRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateRoleRules())
}

package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateUserRoleRules() genericvalidation.Rules {
	return genericvalidation.Rules{
		"UserID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("userID cannot be empty")
			}
			return nil
		},
		"RoleID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("roleID cannot be empty")
			}
			return nil
		},
		"RoleIDs": func(value any) error {
			// 检查角色 ID 列表是否为空
			roleIDs, ok := value.([]string)
			if !ok {
				return errno.ErrInvalidArgument.WithMessage("roleIDs must be a string array")
			}
			if len(roleIDs) == 0 {
				return errno.ErrInvalidArgument.WithMessage("roleIDs cannot be empty")
			}
			return nil
		},
	}
}

// ValidateAssignRolesToUserRequest 校验分配角色请求.
func (v *Validator) ValidateAssignRolesToUserRequest(ctx context.Context, rq *v1.AssignRolesToUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRoleRules())
}

// ValidateGetUserRolesRequest 校验获取用户角色请求.
func (v *Validator) ValidateGetUserRolesRequest(ctx context.Context, rq *v1.GetUserRolesRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRoleRules())
}

// ValidateRemoveRoleFromUserRequest 校验移除角色请求.
func (v *Validator) ValidateRemoveRoleFromUserRequest(ctx context.Context, rq *v1.RemoveRoleFromUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRoleRules())
}

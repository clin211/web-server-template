package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidatePermissionRules() genericvalidation.Rules {
	return genericvalidation.Rules{
		"PermissionID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("permissionID cannot be empty")
			}
			return nil
		},
		"PermissionName": func(value any) error {
			name := value.(string)
			if len(name) == 0 || len(name) > 100 {
				return errno.ErrInvalidArgument.WithMessage("permissionName must be between 1 and 100 characters")
			}
			return nil
		},
		"PermissionCode": func(value any) error {
			code := value.(string)
			if len(code) == 0 || len(code) > 100 {
				return errno.ErrInvalidArgument.WithMessage("permissionCode must be between 1 and 100 characters")
			}
			// 权限编码格式：module:action，如 user:list
			if !isValidUsername(code) {
				return errno.ErrInvalidArgument.WithMessage("permissionCode format is invalid")
			}
			return nil
		},
		"ResourceType": func(value any) error {
			rtype := value.(string)
			if rtype != "menu" && rtype != "button" {
				return errno.ErrInvalidArgument.WithMessage("resourceType must be 'menu' or 'button'")
			}
			return nil
		},
		"ResourcePath": func(value any) error {
			if value == nil {
				return nil
			}
			path := value.(string)
			if len(path) > 200 {
				return errno.ErrInvalidArgument.WithMessage("resourcePath must be less than 200 characters")
			}
			return nil
		},
		"Action": func(value any) error {
			action := value.(string)
			if len(action) == 0 || len(action) > 20 {
				return errno.ErrInvalidArgument.WithMessage("action must be between 1 and 20 characters")
			}
			return nil
		},
		"Description": func(value any) error {
			if value == nil {
				return nil
			}
			desc := value.(string)
			if len(desc) > 200 {
				return errno.ErrInvalidArgument.WithMessage("description must be less than 200 characters")
			}
			return nil
		},
		"ParentID": func(value any) error {
			if value == nil {
				return nil
			}
			if value.(string) == "" {
				return nil // 空字符串表示根节点
			}
			return nil
		},
		"Status": func(value any) error {
			status := value.(int32)
			if status != 0 && status != 1 {
				return errno.ErrInvalidArgument.WithMessage("status must be 0 (enabled) or 1 (disabled)")
			}
			return nil
		},
		"Level": func(value any) error {
			if value == nil {
				return nil
			}
			level := value.(int32)
			if level < 0 {
				return errno.ErrInvalidArgument.WithMessage("level must be >= 0")
			}
			return nil
		},
	}
}

// ValidateCreatePermissionRequest 校验创建权限请求.
func (v *Validator) ValidateCreatePermissionRequest(ctx context.Context, rq *v1.CreatePermissionRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidatePermissionRules())
}

// ValidateUpdatePermissionRequest 校验更新权限请求.
func (v *Validator) ValidateUpdatePermissionRequest(ctx context.Context, rq *v1.UpdatePermissionRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidatePermissionRules(), "PermissionID")
}

// ValidateDeletePermissionRequest 校验删除权限请求.
func (v *Validator) ValidateDeletePermissionRequest(ctx context.Context, rq *v1.DeletePermissionRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidatePermissionRules())
}

// ValidateGetPermissionRequest 校验获取权限请求.
func (v *Validator) ValidateGetPermissionRequest(ctx context.Context, rq *v1.GetPermissionRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidatePermissionRules())
}

// ValidateListPermissionRequest 校验权限列表请求.
func (v *Validator) ValidateListPermissionRequest(ctx context.Context, rq *v1.ListPermissionRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidatePermissionRules(), "PageToken", "PageSize", "ResourceType", "Status", "ParentID")
}

// ValidateListPermissionTreeRequest 校验权限树请求.
func (v *Validator) ValidateListPermissionTreeRequest(ctx context.Context, rq *v1.ListPermissionTreeRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidatePermissionRules(), "Level", "ResourceType", "Status")
}

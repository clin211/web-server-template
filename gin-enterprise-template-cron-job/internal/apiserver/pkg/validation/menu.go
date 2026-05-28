package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateMenuRules() genericvalidation.Rules {
	return genericvalidation.Rules{
		"MenuID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("menuID cannot be empty")
			}
			return nil
		},
		"MenuName": func(value any) error {
			name := value.(string)
			if len(name) == 0 || len(name) > 50 {
				return errno.ErrInvalidArgument.WithMessage("menuName must be between 1 and 50 characters")
			}
			return nil
		},
		"MenuCode": func(value any) error {
			code := value.(string)
			if len(code) == 0 || len(code) > 50 {
				return errno.ErrInvalidArgument.WithMessage("menuCode must be between 1 and 50 characters")
			}
			if !isValidUsername(code) {
				return errno.ErrInvalidArgument.WithMessage("menuCode can only contain letters, numbers and underscores")
			}
			return nil
		},
		"MenuType": func(value any) error {
			mtype := value.(string)
			if mtype != "menu" && mtype != "page" {
				return errno.ErrInvalidArgument.WithMessage("menuType must be 'menu' or 'page'")
			}
			return nil
		},
		"Icon": func(value any) error {
			if value == nil {
				return nil
			}
			icon := value.(string)
			if len(icon) > 50 {
				return errno.ErrInvalidArgument.WithMessage("icon must be less than 50 characters")
			}
			return nil
		},
		"Path": func(value any) error {
			if value == nil {
				return nil
			}
			path := value.(string)
			if len(path) > 200 {
				return errno.ErrInvalidArgument.WithMessage("path must be less than 200 characters")
			}
			return nil
		},
		"Component": func(value any) error {
			if value == nil {
				return nil
			}
			component := value.(string)
			if len(component) > 200 {
				return errno.ErrInvalidArgument.WithMessage("component must be less than 200 characters")
			}
			return nil
		},
		"PermissionID": func(value any) error {
			if value == nil {
				return nil
			}
			if value.(string) == "" {
				return nil // 空字符串表示无权限关联
			}
			return nil
		},
		"SortOrder": func(value any) error {
			return nil // 排序可以是任意整数
		},
		"Visible": func(value any) error {
			visible := value.(int32)
			if visible != 0 && visible != 1 {
				return errno.ErrInvalidArgument.WithMessage("visible must be 0 (hidden) or 1 (visible)")
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
	}
}

// ValidateCreateMenuRequest 校验创建菜单请求.
func (v *Validator) ValidateCreateMenuRequest(ctx context.Context, rq *v1.CreateMenuRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateMenuRules())
}

// ValidateUpdateMenuRequest 校验更新菜单请求.
func (v *Validator) ValidateUpdateMenuRequest(ctx context.Context, rq *v1.UpdateMenuRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "MenuID")
}

// ValidateDeleteMenuRequest 校验删除菜单请求.
func (v *Validator) ValidateDeleteMenuRequest(ctx context.Context, rq *v1.DeleteMenuRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateMenuRules())
}

// ValidateGetMenuRequest 校验获取菜单请求.
func (v *Validator) ValidateGetMenuRequest(ctx context.Context, rq *v1.GetMenuRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateMenuRules())
}

// ValidateListMenuRequest 校验菜单列表请求.
func (v *Validator) ValidateListMenuRequest(ctx context.Context, rq *v1.ListMenuRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "PageToken", "PageSize", "Status", "MenuType", "ParentID")
}

// ValidateListMenuTreeRequest 校验菜单树请求.
func (v *Validator) ValidateListMenuTreeRequest(ctx context.Context, rq *v1.ListMenuTreeRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "Status")
}

// ValidateGetUserMenuTreeRequest 校验获取用户菜单树请求.
func (v *Validator) ValidateGetUserMenuTreeRequest(ctx context.Context, rq *v1.GetUserMenuTreeRequest) error {
	return nil // 该请求从 JWT 获取用户 ID，无需额外验证
}

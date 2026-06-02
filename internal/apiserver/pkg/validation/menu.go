package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// safeStringValue 安全地获取字符串值，如果类型不匹配则返回错误
func safeStringValue(value any) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", errno.ErrInvalidArgument.WithMessage("expected string type")
	}
	return s, nil
}

// safeInt32Value 安全地获取 int32 值，如果类型不匹配则返回错误
func safeInt32Value(value any) (int32, error) {
	i, ok := value.(int32)
	if !ok {
		return 0, errno.ErrInvalidArgument.WithMessage("expected int32 type")
	}
	return i, nil
}

func (v *Validator) ValidateMenuRules() genericvalidation.Rules {
	return genericvalidation.Rules{
		"MenuID": func(value any) error {
			s, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if s == "" {
				return errno.ErrInvalidArgument.WithMessage("menuID cannot be empty")
			}
			return nil
		},
		"MenuName": func(value any) error {
			name, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(name) == 0 || len(name) > 50 {
				return errno.ErrInvalidArgument.WithMessage("menuName must be between 1 and 50 characters")
			}
			return nil
		},
		"MenuCode": func(value any) error {
			code, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if !isValidMenuCode(code) {
				return errno.ErrInvalidArgument.WithMessage("menuCode must be 1-50 characters, containing only letters, numbers, underscores and hyphens")
			}
			return nil
		},
		"MenuType": func(value any) error {
			mtype, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if mtype != "menu" && mtype != "page" {
				return errno.ErrInvalidArgument.WithMessage("menuType must be 'menu' or 'page'")
			}
			return nil
		},
		"I18nKey": func(value any) error {
			if value == nil {
				return nil
			}
			key, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(key) > 100 {
				return errno.ErrInvalidArgument.WithMessage("i18nKey must be less than 100 characters")
			}
			return nil
		},
		"Icon": func(value any) error {
			if value == nil {
				return nil
			}
			icon, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(icon) > 50 {
				return errno.ErrInvalidArgument.WithMessage("icon must be less than 50 characters")
			}
			return nil
		},
		"LocalIcon": func(value any) error {
			if value == nil {
				return nil
			}
			localIcon, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(localIcon) > 50 {
				return errno.ErrInvalidArgument.WithMessage("localIcon must be less than 50 characters")
			}
			return nil
		},
		"IconFontSize": func(value any) error {
			if value == nil {
				return nil
			}
			size, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if size < 0 || size > 100 {
				return errno.ErrInvalidArgument.WithMessage("iconFontSize must be between 0 and 100")
			}
			return nil
		},
		"Path": func(value any) error {
			if value == nil {
				return nil
			}
			path, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(path) > 200 {
				return errno.ErrInvalidArgument.WithMessage("path must be less than 200 characters")
			}
			return nil
		},
		"Component": func(value any) error {
			if value == nil {
				return nil
			}
			component, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(component) > 200 {
				return errno.ErrInvalidArgument.WithMessage("component must be less than 200 characters")
			}
			return nil
		},
		"PermissionID": func(value any) error {
			if value == nil {
				return nil
			}
			permissionID, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if permissionID == "" {
				return nil // 空字符串表示无权限关联
			}
			return nil
		},
		"SortOrder": func(value any) error {
			return nil // 排序可以是任意整数
		},
		"Visible": func(value any) error {
			visible, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if visible != 0 && visible != 1 {
				return errno.ErrInvalidArgument.WithMessage("visible must be 0 (hidden) or 1 (visible)")
			}
			return nil
		},
		"Status": func(value any) error {
			status, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if status != 0 && status != 1 {
				return errno.ErrInvalidArgument.WithMessage("status must be 0 (enabled) or 1 (disabled)")
			}
			return nil
		},
		"Constant": func(value any) error {
			constant, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if constant != 0 && constant != 1 {
				return errno.ErrInvalidArgument.WithMessage("constant must be 0 (no) or 1 (yes)")
			}
			return nil
		},
		"ActiveMenu": func(value any) error {
			if value == nil {
				return nil
			}
			activeMenu, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(activeMenu) > 100 {
				return errno.ErrInvalidArgument.WithMessage("activeMenu must be less than 100 characters")
			}
			return nil
		},
		"HideInMenu": func(value any) error {
			hideInMenu, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if hideInMenu != 0 && hideInMenu != 1 {
				return errno.ErrInvalidArgument.WithMessage("hideInMenu must be 0 (no) or 1 (yes)")
			}
			return nil
		},
		"KeepAlive": func(value any) error {
			keepAlive, err := safeInt32Value(value)
			if err != nil {
				return err
			}
			if keepAlive != 0 && keepAlive != 1 {
				return errno.ErrInvalidArgument.WithMessage("keepAlive must be 0 (no) or 1 (yes)")
			}
			return nil
		},
		"Href": func(value any) error {
			if value == nil {
				return nil
			}
			href, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(href) > 500 {
				return errno.ErrInvalidArgument.WithMessage("href must be less than 500 characters")
			}
			return nil
		},
		"ParentID": func(value any) error {
			if value == nil {
				return nil
			}
			parentID, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if len(parentID) > 50 {
				return errno.ErrInvalidArgument.WithMessage("parentID must be less than 50 characters")
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

// ValidateGetMenuRolesRequest 校验获取菜单角色请求.
func (v *Validator) ValidateGetMenuRolesRequest(ctx context.Context, rq *v1.GetMenuRolesRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "MenuID")
}

// ValidateSetMenuRolesRequest 校验设置菜单角色请求.
func (v *Validator) ValidateSetMenuRolesRequest(ctx context.Context, rq *v1.SetMenuRolesRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "MenuID")
}

// ValidateAddMenuRoleRequest 校验添加菜单角色请求.
func (v *Validator) ValidateAddMenuRoleRequest(ctx context.Context, rq *v1.AddMenuRoleRequest) error {
	return genericvalidation.ValidateAllFields(rq, genericvalidation.Rules{
		"MenuID": v.ValidateMenuRules()["MenuID"],
		"RoleId": func(value any) error {
			id, err := safeStringValue(value)
			if err != nil {
				return err
			}
			if id == "" {
				return errno.ErrInvalidArgument.WithMessage("roleId cannot be empty")
			}
			return nil
		},
	})
}

// ValidateRemoveMenuRoleRequest 校验移除菜单角色请求.
func (v *Validator) ValidateRemoveMenuRoleRequest(ctx context.Context, rq *v1.RemoveMenuRoleRequest) error {
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateMenuRules(), "MenuID")
}

// ValidateSortMenuRequest 校验批量更新菜单排序请求.
func (v *Validator) ValidateSortMenuRequest(ctx context.Context, rq *v1.SortMenuRequest) error {
	if len(rq.Items) == 0 {
		return errno.ErrInvalidArgument.WithMessage("sort items cannot be empty")
	}
	return nil
}

// ValidateGetUserRoutesRequest 校验获取用户路由请求.
func (v *Validator) ValidateGetUserRoutesRequest(ctx context.Context, rq *v1.GetUserRoutesRequest) error {
	return nil // 该请求从 JWT 获取用户 ID，无需额外验证
}

// ValidateGetConstantRoutesRequest 校验获取常量路由请求.
func (v *Validator) ValidateGetConstantRoutesRequest(ctx context.Context, rq *v1.GetConstantRoutesRequest) error {
	return nil // 该请求无需参数，无需额外验证
}
package validation

import (
	"context"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateUserRules() genericvalidation.Rules {
	// 通用的密码校验函数
	validatePassword := func() genericvalidation.ValidatorFunc {
		return func(value any) error {
			return isValidPassword(value.(string))
		}
	}

	// 定义各字段的校验逻辑，通过一个 map 实现模块化和简化
	return genericvalidation.Rules{
		"Password":    validatePassword(),
		"OldPassword": validatePassword(),
		"NewPassword": validatePassword(),
		"UserID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("userID cannot be empty")
			}
			return nil
		},
		"Username": func(value any) error {
			if !isValidUsername(value.(string)) {
				return errno.ErrUsernameInvalid
			}
			return nil
		},
		"Nickname": func(value any) error {
			if len(value.(string)) >= 30 {
				return errno.ErrInvalidArgument.WithMessage("nickname must be less than 30 characters")
			}
			return nil
		},
		"Email": func(value any) error {
			return isValidEmail(value.(string))
		},
		"Phone": func(value any) error {
			return isValidPhone(value.(string))
		},
		"Limit": func(value any) error {
			if value.(int64) <= 0 {
				return errno.ErrInvalidArgument.WithMessage("limit must be greater than 0")
			}
			return nil
		},
		"Offset": func(value any) error {
			return nil
		},
	}
}

// ValidateLogin 校验修改密码请求.
func (v *Validator) ValidateLoginRequest(ctx context.Context, rq *v1.LoginRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateChangePasswordRequest 校验 ChangePasswordRequest 结构体的有效性.
func (v *Validator) ValidateChangePasswordRequest(ctx context.Context, rq *v1.ChangePasswordRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return validateCurrentUser(ctx, rq.GetUserID())
}

// ValidateCreateUserRequest 校验 CreateUserRequest 结构体的有效性.
func (v *Validator) ValidateCreateUserRequest(ctx context.Context, rq *v1.CreateUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateUpdateUserRequest 校验更新用户请求.
func (v *Validator) ValidateUpdateUserRequest(ctx context.Context, rq *v1.UpdateUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return validateCurrentUser(ctx, rq.GetUserID())
}

// ValidateDeleteUserRequest 校验 DeleteUserRequest 结构体的有效性.
func (v *Validator) ValidateDeleteUserRequest(ctx context.Context, rq *v1.DeleteUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return validateCurrentUser(ctx, rq.GetUserID())
}

// ValidateGetUserRequest 校验 GetUserRequest 结构体的有效性.
func (v *Validator) ValidateGetUserRequest(ctx context.Context, rq *v1.GetUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return validateCurrentUser(ctx, rq.GetUserID())
}

// ValidateListUserRequest 校验 ListUserRequest 结构体的有效性.
func (v *Validator) ValidateListUserRequest(ctx context.Context, rq *v1.ListUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}

	// 列表接口会返回所有用户。当前模型没有可验证的管理员/角色声明，因此默认拒绝，
	// 避免把“已登录”误当成“有用户管理权限”。引入 RBAC 后应在这里替换为显式授权。
	return errno.ErrPermissionDenied
}

func validateCurrentUser(ctx context.Context, targetUserID string) error {
	currentUserID := contextx.UserID(ctx)
	if currentUserID == "" {
		return errno.ErrUnauthenticated
	}
	if targetUserID != currentUserID {
		return errno.ErrPermissionDenied
	}
	return nil
}

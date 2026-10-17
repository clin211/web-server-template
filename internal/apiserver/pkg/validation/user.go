package validation

import (
	"context"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"

	genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateUserRules() genericvalidation.Rules {
	password := stringRule(isValidPassword)
	return genericvalidation.Rules{
		"Password": password, "OldPassword": password, "NewPassword": password,
		"UserID": stringRule(func(value string) error {
			if value == "" {
				return errno.ErrInvalidArgument.WithMessage("userID cannot be empty")
			}
			return nil
		}),
		"Username": stringRule(func(value string) error {
			if !isValidUsername(value) {
				return errno.ErrUsernameInvalid
			}
			return nil
		}),
		"Nickname": stringRule(func(value string) error {
			if len(value) >= 30 {
				return errno.ErrInvalidArgument.WithMessage("nickname must be less than 30 characters")
			}
			return nil
		}),
		"Email": stringRule(isValidEmail),
		"Phone": stringRule(isValidPhone),
		"Limit": func(value any) error {
			limit, ok := value.(int64)
			if !ok || limit <= 0 {
				return errno.ErrInvalidArgument.WithMessage("limit must be greater than 0")
			}
			return nil
		},
		"Offset": func(any) error { return nil },
	}
}

func stringRule(validate func(string) error) genericvalidation.ValidatorFunc {
	return func(value any) error {
		text, ok := value.(string)
		if !ok {
			return errno.ErrInvalidArgument.WithMessage("field must be a string")
		}
		return validate(text)
	}
}

// ValidateLoginRequest 校验登录请求.
func (v *Validator) ValidateLoginRequest(ctx context.Context, rq *v1.LoginRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateChangePasswordRequest 校验 ChangePasswordRequest 结构体的有效性.
func (v *Validator) ValidateChangePasswordRequest(ctx context.Context, rq *v1.ChangePasswordRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return v.validateUserAccess(ctx, rq.GetUserID())
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
	return v.validateUserAccess(ctx, rq.GetUserID())
}

// ValidateDeleteUserRequest 校验 DeleteUserRequest 结构体的有效性.
func (v *Validator) ValidateDeleteUserRequest(ctx context.Context, rq *v1.DeleteUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return v.validateUserAccess(ctx, rq.GetUserID())
}

// ValidateGetUserRequest 校验 GetUserRequest 结构体的有效性.
func (v *Validator) ValidateGetUserRequest(ctx context.Context, rq *v1.GetUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}
	return v.validateUserAccess(ctx, rq.GetUserID())
}

// ValidateListUserRequest 校验 ListUserRequest 结构体的有效性.
func (v *Validator) ValidateListUserRequest(ctx context.Context, rq *v1.ListUserRequest) error {
	if err := genericvalidation.ValidateAllFields(rq, v.ValidateUserRules()); err != nil {
		return err
	}

	// 用户列表需要实时校验 user.manage，避免已签发 JWT 缓存过期的管理员授权。
	if v.store != nil {
		allowed, err := store.NewAuthorizationStore(v.store).Allowed(ctx, contextx.UserID(ctx), "user.manage")
		if err != nil {
			return errno.ErrServiceUnavailable
		}
		if allowed {
			return nil
		}
	}
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

func (v *Validator) validateUserAccess(ctx context.Context, id string) error {
	err := validateCurrentUser(ctx, id)
	if err == nil || v.store == nil || contextx.UserID(ctx) == "" {
		return err
	}
	allowed, e := store.NewAuthorizationStore(v.store).Allowed(ctx, contextx.UserID(ctx), "user.manage")
	if e != nil {
		return errno.ErrServiceUnavailable
	}
	if allowed {
		return nil
	}
	return err
}

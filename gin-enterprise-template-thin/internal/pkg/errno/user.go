package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

var (
	// 用户相关错误
	ErrUsernameInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidUsername,
		"User.UsernameInvalid",
		"Invalid username: Username must consist of letters, digits, and underscores only, and its length must be between 3 and 20 characters.",
	)

	ErrPasswordInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidPassword,
		"User.PasswordInvalid",
		"Password is incorrect.",
	)

	ErrUserAlreadyExists = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.AlreadyExists",
		"User already exists.",
	)

	ErrUserNotFound = errorsx.NewBizError(
		errorsx.CodeUserNotFound,
		"User.NotFound",
		"User not found.",
	)

	// 新增更多用户相关错误
	ErrUserDisabled = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Disabled",
		"用户账户已被禁用。",
	)

	ErrUserLocked = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Locked",
		"用户账户因多次登录失败而被锁定。",
	)

	ErrUserPasswordExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.PasswordExpired",
		"用户密码已过期，请重置您的密码。",
	)

	ErrUserInsufficientBalance = errorsx.NewBizError(
		errorsx.CodeUserInsufficientBalance,
		"User.InsufficientBalance",
		"用户余额不足以执行此操作。",
	)

	ErrUserEmailAlreadyVerified = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.EmailAlreadyVerified",
		"用户邮箱已经验证过。",
	)

	ErrUserEmailVerificationExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.EmailVerificationExpired",
		"邮箱验证令牌已过期。",
	)
)

package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

var (
	// 用户相关错误
	ErrUsernameInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidUsername,
		"User.InvalidUsername",
		"用户名无效：用户名只能包含字母、数字和下划线，长度必须在 3 到 20 个字符之间。",
	)

	ErrPasswordInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidPassword,
		"User.InvalidPassword",
		"密码强度不符合太低，需要包含大小写字母、数字。",
	)

	ErrUserAlreadyExists = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.AlreadyExists",
		"用户已存在。",
	)

	ErrUserNotFound = errorsx.NewBizError(
		errorsx.CodeUserNotFound,
		"User.NotFound",
		"用户未找到。",
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

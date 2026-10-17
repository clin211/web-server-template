package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
)

var (
	// ErrUsernameInvalid 表示用户名无效。
	ErrUsernameInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidUsername,
		"User.InvalidUsername",
		i18n.DefaultMessage("User.InvalidUsername"),
	)

	ErrPasswordInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidPassword,
		"User.InvalidPassword",
		i18n.DefaultMessage("User.InvalidPassword"),
	)

	ErrUserAlreadyExists = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.AlreadyExists",
		i18n.DefaultMessage("User.AlreadyExists"),
	)

	ErrUserNotFound = errorsx.NewBizError(
		errorsx.CodeUserNotFound,
		"User.NotFound",
		i18n.DefaultMessage("User.NotFound"),
	)

	// ErrUserDisabled 表示账号已被禁用。
	ErrUserDisabled = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Disabled",
		i18n.DefaultMessage("User.Disabled"),
	)

	ErrUserLocked = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Locked",
		i18n.DefaultMessage("User.Locked"),
	)

	ErrUserPasswordExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.PasswordExpired",
		i18n.DefaultMessage("User.PasswordExpired"),
	)

	ErrUserEmailAlreadyVerified = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.EmailAlreadyVerified",
		i18n.DefaultMessage("User.EmailAlreadyVerified"),
	)

	ErrUserEmailVerificationExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.EmailVerificationExpired",
		i18n.DefaultMessage("User.EmailVerificationExpired"),
	)
)

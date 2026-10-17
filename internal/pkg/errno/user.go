package errno

import (
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

var (
	// ErrUsernameInvalid 表示用户名无效。
	ErrUsernameInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidUsername,
		"User.InvalidUsername",
		localization.DefaultMessage("User.InvalidUsername"),
	)

	ErrPasswordInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidPassword,
		"User.InvalidPassword",
		localization.DefaultMessage("User.InvalidPassword"),
	)

	ErrUserAlreadyExists = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.AlreadyExists",
		localization.DefaultMessage("User.AlreadyExists"),
	)

	ErrUserNotFound = errorsx.NewBizError(
		errorsx.CodeUserNotFound,
		"User.NotFound",
		localization.DefaultMessage("User.NotFound"),
	)

	// ErrUserDisabled 表示账号已被禁用。
	ErrUserDisabled = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Disabled",
		localization.DefaultMessage("User.Disabled"),
	)

	ErrUserLocked = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Locked",
		localization.DefaultMessage("User.Locked"),
	)

	ErrUserPasswordExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.PasswordExpired",
		localization.DefaultMessage("User.PasswordExpired"),
	)

	ErrUserEmailAlreadyVerified = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.EmailAlreadyVerified",
		localization.DefaultMessage("User.EmailAlreadyVerified"),
	)

	ErrUserEmailVerificationExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.EmailVerificationExpired",
		localization.DefaultMessage("User.EmailVerificationExpired"),
	)
)

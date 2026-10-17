package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
)

// 通用预定义错误的别名。唯一词表在 pkg/errorsx/code.go；本包仅为业务层
// 提供贴近分层习惯的引用入口，不再单独定义同码不同文案的重复哨兵。
var (
	ErrInternal         = errorsx.ErrInternal
	ErrNotFound         = errorsx.ErrNotFound
	ErrBind             = errorsx.ErrBind
	ErrInvalidArgument  = errorsx.ErrInvalidArgument
	ErrUnauthenticated  = errorsx.ErrUnauthenticated
	ErrPermissionDenied = errorsx.ErrPermissionDenied
	ErrOperationFailed  = errorsx.ErrOperationFailed

	// ErrSignToken 表示令牌签名失败。
	ErrSignToken    = errorsx.NewBizError(errorsx.CodeAuthSignToken, "Auth.SignFailed", i18n.DefaultMessage("Auth.SignFailed"))
	ErrTokenInvalid = errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", i18n.DefaultMessage("Auth.TokenInvalid"))
	ErrTokenExpired = errorsx.NewBizError(errorsx.CodeAuthTokenExpired, "Auth.TokenExpired", i18n.DefaultMessage("Auth.TokenExpired"))

	// ErrPageNotFound 表示页面不存在。
	ErrPageNotFound       = errorsx.NewBizError(errorsx.CodeUserNotFound, "Common.PageNotFound", i18n.DefaultMessage("Common.PageNotFound"))
	ErrServiceUnavailable = errorsx.NewBizError(errorsx.CodeServiceUnavailable, "Service.Unavailable", i18n.DefaultMessage("Service.Unavailable"))
	ErrTooManyRequests    = errorsx.NewBizError(errorsx.CodeTooManyRequests, "Service.TooManyRequests", i18n.DefaultMessage("Service.TooManyRequests"))

	// ErrAddRole 表示角色分配失败。
	ErrAddRole    = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleAddFailed", i18n.DefaultMessage("Auth.RoleAddFailed"))
	ErrRemoveRole = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleRemoveFailed", i18n.DefaultMessage("Auth.RoleRemoveFailed"))
)

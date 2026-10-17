package errno

import (
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
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
	ErrSignToken    = errorsx.NewBizError(errorsx.CodeAuthSignToken, "Auth.SignFailed", localization.DefaultMessage("Auth.SignFailed"))
	ErrTokenInvalid = errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", localization.DefaultMessage("Auth.TokenInvalid"))
	ErrTokenExpired = errorsx.NewBizError(errorsx.CodeAuthTokenExpired, "Auth.TokenExpired", localization.DefaultMessage("Auth.TokenExpired"))

	// ErrPageNotFound 表示页面不存在。
	ErrPageNotFound       = errorsx.NewBizError(errorsx.CodeUserNotFound, "Common.PageNotFound", localization.DefaultMessage("Common.PageNotFound"))
	ErrServiceUnavailable = errorsx.NewBizError(errorsx.CodeServiceUnavailable, "Service.Unavailable", localization.DefaultMessage("Service.Unavailable"))
	ErrTooManyRequests    = errorsx.NewBizError(errorsx.CodeTooManyRequests, "Service.TooManyRequests", localization.DefaultMessage("Service.TooManyRequests"))

	// ErrAddRole 表示角色分配失败。
	ErrAddRole    = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleAddFailed", localization.DefaultMessage("Auth.RoleAddFailed"))
	ErrRemoveRole = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleRemoveFailed", localization.DefaultMessage("Auth.RoleRemoveFailed"))
)

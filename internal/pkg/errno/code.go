package errno

import (
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

	// Token 相关错误
	ErrSignToken    = errorsx.NewBizError(errorsx.CodeAuthSignToken, "Auth.SignFailed", "签名 JSON Web 令牌时发生错误。")
	ErrTokenInvalid = errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效。")
	ErrTokenExpired = errorsx.NewBizError(errorsx.CodeAuthTokenExpired, "Auth.TokenExpired", "令牌已过期。")

	// 通用业务错误
	ErrPageNotFound       = errorsx.NewBizError(errorsx.CodeUserNotFound, "Common.PageNotFound", "页面未找到。")
	ErrServiceUnavailable = errorsx.NewBizError(errorsx.CodeServiceUnavailable, "Service.Unavailable", "服务暂时不可用。")
	ErrTooManyRequests    = errorsx.NewBizError(errorsx.CodeTooManyRequests, "Service.TooManyRequests", "请求过多，请稍后再试。")

	// 角色管理错误
	ErrAddRole    = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleAddFailed", "添加角色时发生错误。")
	ErrRemoveRole = errorsx.NewBizError(errorsx.CodeInternalServer, "Auth.RoleRemoveFailed", "移除角色时发生错误。")
)

package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

// 重新导出错误码和预定义错误，方便业务层使用
var (
	// 预定义错误
	OK = errorsx.OK

	ErrInternal         = errorsx.NewBizError(errorsx.CodeInternalServer, "Common.InternalError", "内部服务器错误。")
	ErrNotFound         = errorsx.NewBizError(errorsx.CodeUserNotFound, "Common.NotFound", "资源未找到。")
	ErrBind             = errorsx.NewBizError(errorsx.CodeUserInvalidCredentials, "Common.BindError", "请求体绑定失败。")
	ErrInvalidArgument  = errorsx.NewBizError(errorsx.CodeUserInvalidCredentials, "Common.InvalidArgument", "参数校验失败。")
	ErrUnauthenticated  = errorsx.NewBizError(errorsx.CodeAuthUnauthenticated, "Auth.Unauthenticated", "未认证。")
	ErrPermissionDenied = errorsx.NewBizError(errorsx.CodeUserPermissionDenied, "Auth.PermissionDenied", "权限不足。")
	ErrOperationFailed  = errorsx.NewBizError(errorsx.CodeInternalServer, "Common.OperationFailed", "操作失败，请稍后重试。")

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

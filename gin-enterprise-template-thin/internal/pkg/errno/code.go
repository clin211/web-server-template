package errno

import (
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

// 重新导出错误码和预定义错误，方便业务层使用
var (
	// 错误码
	CodeOK                      = errorsx.CodeOK
	CodeUserNotFound            = errorsx.CodeUserNotFound
	CodeUserAlreadyExists       = errorsx.CodeUserAlreadyExists
	CodeUserInvalidCredentials  = errorsx.CodeUserInvalidCredentials
	CodeUserInsufficientBalance = errorsx.CodeUserInsufficientBalance
	CodeUserInvalidUsername     = errorsx.CodeUserInvalidUsername
	CodeUserInvalidPassword     = errorsx.CodeUserInvalidPassword
	CodeUserPermissionDenied    = errorsx.CodeUserPermissionDenied

	CodePostNotFound         = errorsx.CodePostNotFound
	CodePostAlreadyPublished = errorsx.CodePostAlreadyPublished
	CodePostPermissionDenied = errorsx.CodePostPermissionDenied

	CodeCommentNotFound         = errorsx.CodeCommentNotFound
	CodeCommentPermissionDenied = errorsx.CodeCommentPermissionDenied

	CodeAuthUnauthenticated = errorsx.CodeAuthUnauthenticated
	CodeAuthTokenInvalid    = errorsx.CodeAuthTokenInvalid
	CodeAuthTokenExpired    = errorsx.CodeAuthTokenExpired
	CodeAuthSignToken       = errorsx.CodeAuthSignToken

	CodeDatabaseConnectFailed = errorsx.CodeDatabaseConnectFailed
	CodeDatabaseReadFailed    = errorsx.CodeDatabaseReadFailed
	CodeDatabaseWriteFailed   = errorsx.CodeDatabaseWriteFailed

	CodeCacheConnectFailed = errorsx.CodeCacheConnectFailed
	CodeCacheReadFailed    = errorsx.CodeCacheReadFailed
	CodeCacheWriteFailed   = errorsx.CodeCacheWriteFailed

	CodeInternalServer     = errorsx.CodeInternalServer
	CodeServiceUnavailable = errorsx.CodeServiceUnavailable
	CodeRequestTimeout     = errorsx.CodeRequestTimeout
	CodeTooManyRequests    = errorsx.CodeTooManyRequests

	// 预定义错误
	OK = errorsx.OK

	ErrInternal         = errorsx.ErrInternal
	ErrNotFound         = errorsx.ErrNotFound
	ErrBind             = errorsx.ErrBind
	ErrInvalidArgument  = errorsx.ErrInvalidArgument
	ErrUnauthenticated  = errorsx.ErrUnauthenticated
	ErrPermissionDenied = errorsx.ErrPermissionDenied
	ErrOperationFailed  = errorsx.ErrOperationFailed

	// 数据库错误
	ErrDBRead    = errorsx.NewBizError(errorsx.CodeDatabaseReadFailed, "Database.ReadFailed", "数据库读取失败。")
	ErrDBWrite   = errorsx.NewBizError(errorsx.CodeDatabaseWriteFailed, "Database.WriteFailed", "数据库写入失败。")
	ErrDBConnect = errorsx.NewBizError(errorsx.CodeDatabaseConnectFailed, "Database.ConnectFailed", "数据库连接失败。")

	// 缓存错误
	ErrCacheRead    = errorsx.NewBizError(errorsx.CodeCacheReadFailed, "Cache.ReadFailed", "缓存读取失败。")
	ErrCacheWrite   = errorsx.NewBizError(errorsx.CodeCacheWriteFailed, "Cache.WriteFailed", "缓存写入失败。")
	ErrCacheConnect = errorsx.NewBizError(errorsx.CodeCacheConnectFailed, "Cache.ConnectFailed", "缓存连接失败。")

	// Token 相关错误
	ErrSignToken    = errorsx.NewBizError(errorsx.CodeAuthSignToken, "Auth.SignToken", "签名 JSON Web 令牌时发生错误。")
	ErrTokenInvalid = errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效。")
	ErrTokenExpired = errorsx.NewBizError(errorsx.CodeAuthTokenExpired, "Auth.TokenExpired", "令牌已过期。")

	// 通用业务错误
	ErrPageNotFound       = errorsx.NewBizError(errorsx.CodeUserNotFound, "NotFound.PageNotFound", "页面未找到。")
	ErrServiceUnavailable = errorsx.NewBizError(errorsx.CodeServiceUnavailable, "Service.Unavailable", "服务暂时不可用。")
	ErrTooManyRequests    = errorsx.NewBizError(errorsx.CodeTooManyRequests, "Service.TooManyRequests", "请求过多，请稍后再试。")

	// 角色管理错误
	ErrAddRole    = errorsx.NewBizError(errorsx.CodeInternalServer, "Role.AddFailed", "添加角色时发生错误。")
	ErrRemoveRole = errorsx.NewBizError(errorsx.CodeInternalServer, "Role.RemoveFailed", "移除角色时发生错误。")
)

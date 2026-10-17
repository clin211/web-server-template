package errorsx

import "github.com/clin211/gin-enterprise-template/pkg/i18n"

// 预定义的标准错误（唯一词表）。
// internal/pkg/errno 中的同名变量是以下哨兵的别名；新增预定义错误时只在本包
// 定义，业务层继续从 errno 引用，避免两套 reason/文案并存导致漂移。
var (
	// ErrInternal 表示所有未知的服务器端错误
	ErrInternal = NewBizError(CodeInternalServer, "Common.InternalError", i18n.DefaultMessage("Common.InternalError"))

	// ErrNotFound 表示资源未找到
	ErrNotFound = NewBizError(CodeUserNotFound, "Common.NotFound", i18n.DefaultMessage("Common.NotFound"))

	// ErrBind 表示请求体绑定错误
	ErrBind = NewBizError(CodeUserInvalidCredentials, "Common.BindError", i18n.DefaultMessage("Common.BindError"))

	// ErrInvalidArgument 表示参数验证失败
	ErrInvalidArgument = NewBizError(CodeUserInvalidCredentials, "Common.InvalidArgument", i18n.DefaultMessage("Common.InvalidArgument"))

	// ErrUnauthenticated 表示认证失败
	ErrUnauthenticated = NewBizError(CodeAuthUnauthenticated, "Auth.Unauthenticated", i18n.DefaultMessage("Auth.Unauthenticated"))

	// ErrPermissionDenied 表示请求没有权限
	ErrPermissionDenied = NewBizError(CodeUserPermissionDenied, "Auth.PermissionDenied", i18n.DefaultMessage("Auth.PermissionDenied"))

	// ErrOperationFailed 表示操作失败
	ErrOperationFailed = NewBizError(CodeInternalServer, "Common.OperationFailed", i18n.DefaultMessage("Common.OperationFailed"))
)

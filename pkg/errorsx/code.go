package errorsx

// 预定义的标准错误（共享哨兵）。
// internal/pkg/errno 中的同名变量是以下哨兵的别名；新增预定义错误时只在本包
// 定义，业务层继续从 errno 引用，避免两套 reason/文案并存导致漂移。
var (
	// ErrInternal 表示所有未知的服务器端错误
	ErrInternal = NewBizError(CodeInternalServer, "Common.InternalError", defaultMessage("Common.InternalError"))

	// ErrNotFound 表示资源未找到
	ErrNotFound = NewBizError(CodeUserNotFound, "Common.NotFound", defaultMessage("Common.NotFound"))

	// ErrBind 表示请求体绑定错误
	ErrBind = NewBizError(CodeUserInvalidCredentials, "Common.BindError", defaultMessage("Common.BindError"))

	// ErrInvalidArgument 表示参数验证失败
	ErrInvalidArgument = NewBizError(CodeUserInvalidCredentials, "Common.InvalidArgument", defaultMessage("Common.InvalidArgument"))

	// ErrUnauthenticated 表示认证失败
	ErrUnauthenticated = NewBizError(CodeAuthUnauthenticated, "Auth.Unauthenticated", defaultMessage("Auth.Unauthenticated"))

	// ErrPermissionDenied 表示请求没有权限
	ErrPermissionDenied = NewBizError(CodeUserPermissionDenied, "Auth.PermissionDenied", defaultMessage("Auth.PermissionDenied"))

	// ErrOperationFailed 表示操作失败
	ErrOperationFailed = NewBizError(CodeInternalServer, "Common.OperationFailed", defaultMessage("Common.OperationFailed"))
)

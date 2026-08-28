package errorsx

// 预定义的标准错误（唯一词表）。
// internal/pkg/errno 中的同名变量是以下哨兵的别名；新增预定义错误时只在本包
// 定义，业务层继续从 errno 引用，避免两套 reason/文案并存导致漂移。
var (
	// ErrInternal 表示所有未知的服务器端错误
	ErrInternal = NewBizError(CodeInternalServer, "Common.InternalError", "内部服务器错误。")

	// ErrNotFound 表示资源未找到
	ErrNotFound = NewBizError(CodeUserNotFound, "Common.NotFound", "资源未找到。")

	// ErrBind 表示请求体绑定错误
	ErrBind = NewBizError(CodeUserInvalidCredentials, "Common.BindError", "请求体绑定失败。")

	// ErrInvalidArgument 表示参数验证失败
	ErrInvalidArgument = NewBizError(CodeUserInvalidCredentials, "Common.InvalidArgument", "参数校验失败。")

	// ErrUnauthenticated 表示认证失败
	ErrUnauthenticated = NewBizError(CodeAuthUnauthenticated, "Auth.Unauthenticated", "未认证。")

	// ErrPermissionDenied 表示请求没有权限
	ErrPermissionDenied = NewBizError(CodeUserPermissionDenied, "Auth.PermissionDenied", "权限不足。")

	// ErrOperationFailed 表示操作失败
	ErrOperationFailed = NewBizError(CodeInternalServer, "Common.OperationFailed", "操作失败，请稍后重试。")
)

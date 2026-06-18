package errorsx

// 预定义的标准错误。
var (
	// OK 代表请求成功
	OK = NewBizError(CodeOK, "OK", "成功")

	// ErrInternal 表示所有未知的服务器端错误
	ErrInternal = NewBizError(CodeInternalServer, "InternalError", "内部服务器错误")

	// ErrNotFound 表示资源未找到
	ErrNotFound = NewBizError(CodeUserNotFound, "NotFound", "资源未找到")

	// ErrBind 表示请求体绑定错误
	ErrBind = NewBizError(CodeUserInvalidCredentials, "BindError", "请求体绑定失败")

	// ErrInvalidArgument 表示参数验证失败
	ErrInvalidArgument = NewBizError(CodeUserInvalidCredentials, "InvalidArgument", "参数校验失败")

	// ErrUnauthenticated 表示认证失败
	ErrUnauthenticated = NewBizError(CodeAuthUnauthenticated, "Auth.Unauthenticated", "未认证")

	// ErrPermissionDenied 表示请求没有权限
	ErrPermissionDenied = NewBizError(CodeUserPermissionDenied, "Auth.PermissionDenied", "权限不足")

	// ErrOperationFailed 表示操作失败
	ErrOperationFailed = NewBizError(CodeInternalServer, "OperationFailed", "操作失败，请稍后重试")
)

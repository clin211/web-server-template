package known

// 定义 HTTP/gRPC 头。
// gRPC 使用 HTTP/2 作为其底层传输协议，而 HTTP/2 规范
// 要求头键必须为小写。因此，在 gRPC 中，所有头键都
// 被强制转换为小写以符合 HTTP/2 的要求。
// 在 HTTP/1.x 中，许多实现保留了用户设置的大小写格式，
// 但某些 HTTP 框架或工具库（例如某些 Web 服务器或代理）
// 可能会自动将头转换为小写以简化处理逻辑。
// 为了兼容性，这里将所有头统一设置为小写。
// 此外，以 "x-" 为前缀的头键表示它们是自定义头。
const (
	// XRequestID 定义表示请求 ID 的 context 键。
	XRequestID = "x-request-id"

	// XUserID 定义表示请求用户 ID 的 context 键。
	// UserID 在用户的整个生命周期中是唯一的。
	XUserID = "x-user-id"

	// XUsername 定义表示请求用户名的 context 键。
	XUsername = "x-username"
)

// 定义其他常量。
const (
	// AdminUsername 表示管理员用户的用户名。
	AdminUsername = "root"

	// MaxErrGroupConcurrency 定义 errgroup 的最大并发任务数。
	// 它用于限制在 errgroup 中同时执行的 Goroutine 数量，
	// 防止资源耗尽并增强程序稳定性。
	// 此值可以根据具体场景和需求进行调整。
	MaxErrGroupConcurrency = 1000
)

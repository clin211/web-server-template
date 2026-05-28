/*
Package contextx 为 context 提供扩展功能，允许在 context 中存储和提取用户相关信息，如用户 ID、用户名和访问令牌。

后缀 "x" 表示扩展或变体，使包名简洁且易于记忆。此包中的函数简化了在 context 中传递和管理用户信息的过程，适用于需要基于 context 的数据传输的场景。

典型用法：
在 HTTP 请求中间件或服务函数中，可以使用这些方法将用户信息存储在 context 中，确保在整个请求生命周期中安全共享，同时避免使用全局变量和参数传递。

示例：

	// 创建一个新的 context
	ctx := context.Background()

	// 在 context 中存储用户 ID 和用户名
	ctx = contextx.WithUserID(ctx, "user-xxxx")
	ctx = contextx.WithUsername(ctx, "sampleUser")

	// 从 context 中提取用户信息
	userID := contextx.UserID(ctx)
	username := contextx.Username(ctx)
*/
package contextx // import "github.com/clin211/gin-enterprise-template/internal/pkg/contextx"

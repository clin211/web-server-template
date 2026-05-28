package empty

import "context"

// emptyLogger 是一个实现 Logger 接口的空操作日志记录器。
// 它不执行任何日志记录操作。
type emptyLogger struct{}

// NewLogger 创建并返回一个新的 emptyLogger 实例。
func NewLogger() *emptyLogger {
	return &emptyLogger{} // 返回一个新的 emptyLogger 实例
}

// Error 是一个满足 Logger 接口的空操作方法。
// 它不记录任何错误消息或上下文。
func (l *emptyLogger) Error(ctx context.Context, err error, msg string, kvs ...any) {
	// 不执行日志记录操作
}

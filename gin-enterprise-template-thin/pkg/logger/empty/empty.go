package empty

import "github.com/clin211/gin-enterprise-template/pkg/logger"

// emptyLogger 是 logger.Logger 接口的实现，不执行任何操作。
// 这在需要日志记录器但不希望输出日志的场景中很有用。
type emptyLogger struct{}

// 确保 emptyLogger 实现了 logger.Logger 接口。
var _ logger.Logger = (*emptyLogger)(nil)

// NewLogger 返回一个新的空日志记录器实例。
func NewLogger() *emptyLogger {
	return &emptyLogger{}
}

// Debug 以 Debug 级别记录日志消息。此实现不执行任何操作。
func (l *emptyLogger) Debug(msg string, keysAndValues ...any) {}

// Warn 以 Warn 级别记录日志消息。此实现不执行任何操作。
func (l *emptyLogger) Warn(msg string, keysAndValues ...any) {}

// Info 以 Info 级别记录日志消息。此实现不执行任何操作。
func (l *emptyLogger) Info(msg string, keysAndValues ...any) {}

// Error 以 Error 级别记录日志消息。此实现不执行任何操作。
func (l *emptyLogger) Error(msg string, keysAndValues ...any) {}

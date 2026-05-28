package watch

import (
	"log/slog"
)

// cronLogger 实现 cron.Logger 接口。
type cronLogger struct{}

// NewLogger 返回一个 cron 日志记录器。
func NewLogger() *cronLogger {
	return &cronLogger{}
}

// Debug 记录关于 cron 操作的常规消息。
func (l *cronLogger) Debug(msg string, kvs ...any) {
	slog.Debug(msg, kvs...)
}

// Info 记录关于 cron 操作的常规消息。
func (l *cronLogger) Info(msg string, kvs ...any) {
	slog.Debug(msg, kvs...)
}

// Error 记录错误条件。
func (l *cronLogger) Error(err error, msg string, kvs ...any) {
	// 将error作为第一个键值对参数添加
	args := make([]any, 0, len(kvs)+2)
	args = append(args, "error", err)
	args = append(args, kvs...)
	slog.Error(msg, args...)
}

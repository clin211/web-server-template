package store

import (
	"context"

	"k8s.io/klog/v2"
)

// Logger 是一个实现了 Logger 接口的日志记录器。
// 它使用 log 包来记录带有额外上下文的错误消息。
type Logger struct{}

// NewLogger 创建并返回一个新的 Logger 实例。
func NewLogger() *Logger {
	return &Logger{}
}

// Error 使用 log 包记录带有提供的上下文的错误消息。
func (l *Logger) Error(ctx context.Context, err error, msg string, kvs ...any) {
	klog.FromContext(ctx).Error(err, msg, kvs...)
}

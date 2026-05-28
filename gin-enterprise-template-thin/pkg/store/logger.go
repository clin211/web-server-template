package store

import (
	"context"
)

// Logger 定义了一个用于记录带有上下文信息的错误的接口。
type Logger interface {
	// Error 记录带有关联上下文的错误消息。
	Error(ctx context.Context, err error, message string, kvs ...any)
}

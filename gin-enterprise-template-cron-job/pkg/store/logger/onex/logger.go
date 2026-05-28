package onex

import (
	"github.com/clin211/gin-enterprise-template/pkg/log"
)

// onexLogger 是一个实现 Logger 接口的日志记录器。
// 它使用 log 包来记录带有额外上下文的错误消息。
type onexLogger struct{}

// NewLogger 创建并返回一个新的 onexLogger 实例。
func NewLogger() *onexLogger {
	return &onexLogger{}
}

// Error 使用 log 包记录带有提供上下文的错误消息。
func (l *onexLogger) Error(err error, msg string, kvs ...any) {
	log.Errorw(err, msg, kvs...)
}

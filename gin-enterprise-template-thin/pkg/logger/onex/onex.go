package onex

import (
	"github.com/clin211/gin-enterprise-template/pkg/log"
	"github.com/clin211/gin-enterprise-template/pkg/logger"
)

// onexLogger 提供 logger.Logger 接口的实现。
type onexLogger struct{}

// 确保 onexLogger 实现了 logger.Logger 接口。
var _ logger.Logger = (*onexLogger)(nil)

// NewLogger 创建一个新的 onexLogger 实例。
func NewLogger() *onexLogger {
	return &onexLogger{}
}

// Debug 记录带有附加键值对的调试消息。
func (l *onexLogger) Debug(msg string, kvs ...any) {
	log.Debugw(msg, kvs...)
}

// Warn 记录带有附加键值对的警告消息。
func (l *onexLogger) Warn(msg string, kvs ...any) {
	log.Warnw(msg, kvs...)
}

// Info 记录带有附加键值对的信息性消息。
func (l *onexLogger) Info(msg string, kvs ...any) {
	log.Infow(msg, kvs...)
}

// Error 记录带有附加键值对的错误消息。
func (l *onexLogger) Error(msg string, kvs ...any) {
	log.Errorw(nil, msg, kvs...)
}

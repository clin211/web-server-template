package logger

// Logger 定义了不同级别日志记录的方法。
type Logger interface {
	// Debug 以调试级别记录日志消息，可附带键值对。
	Debug(message string, keysAndValues ...any)

	// Warn 以警告级别记录日志消息，可附带键值对。
	Warn(message string, keysAndValues ...any)

	// Info 以信息级别记录日志消息，可附带键值对。
	Info(message string, keysAndValues ...any)

	// Error 以错误级别记录日志消息，可附带键值对。
	Error(message string, keysAndValues ...any)
}

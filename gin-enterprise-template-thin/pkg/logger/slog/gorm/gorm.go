package gormslog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// SlogLogger 使用 Go 的 log/slog 包实现 gorm.io/gorm/logger.Interface。
type SlogLogger struct {
	Logger                    *slog.Logger
	LogLevel                  logger.LogLevel // GORM 日志级别
	SlowThreshold             time.Duration   // 慢查询阈值
	IgnoreRecordNotFoundError bool            // 是否忽略 RecordNotFound 错误
}

// New 返回一个带有合理默认值的新 SlogLogger 实例。
func New(l *slog.Logger) *SlogLogger {
	return &SlogLogger{
		Logger:                    l,
		LogLevel:                  logger.Info,
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode 更改日志记录器的日志级别并返回新实例。
func (l *SlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 记录信息性消息。
func (l *SlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.Logger.Log(ctx, slog.LevelInfo, fmt.Sprintf(msg, data...))
	}
}

// Warn 记录警告消息。
func (l *SlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.Logger.Log(ctx, slog.LevelWarn, fmt.Sprintf(msg, data...))
	}
}

// Error 记录错误消息。
func (l *SlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.Logger.Log(ctx, slog.LevelError, fmt.Sprintf(msg, data...))
	}
}

// Trace 记录 SQL 语句、执行时长、影响行数和错误。
func (l *SlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []any{
		slog.String("sql", sql),
		slog.String("duration", elapsed.String()),
		slog.Int64("rows", rows),
	}

	switch {
	case err != nil && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		if l.LogLevel >= logger.Error {
			fields = append(fields, slog.String("error", err.Error()))
			l.Logger.Log(ctx, slog.LevelError, "SQL execution failed", fields...)
		}
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		fields = append(fields, slog.String("warning", "SLOW QUERY"))
		l.Logger.Log(ctx, slog.LevelWarn, "slow SQL query", fields...)
	case l.LogLevel == logger.Info:
		l.Logger.Log(ctx, slog.LevelInfo, "SQL", fields...)
	}
}

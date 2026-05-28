package gin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// 标准追踪头常量
const (
	// W3C 追踪上下文标准（最推荐）
	TraceParentHeaderKey = "traceparent"

	// 简单追踪 ID（最广泛使用）
	TraceIDHeaderKey = "X-Trace-Id"

	// 通用请求 ID（通用兼容）
	RequestIDHeaderKey = "X-Request-Id"

	// 用于额外上下文的追踪状态
	TraceStateHeaderKey = "tracestate"
)

// TraceInjectionMode 定义追踪信息的注入方式
type TraceInjectionMode int

const (
	// InjectW3CTraceContext 注入完整的 W3C 追踪上下文（推荐）
	InjectW3CTraceContext TraceInjectionMode = iota
	// InjectTraceIDOnly 仅注入追踪 ID
	InjectTraceIDOnly
	// InjectBoth 同时注入 W3C 格式和简单追踪 ID
	InjectBoth
	// InjectNone 禁用追踪注入
	InjectNone
)

// ObservabilityOptions 保存追踪注入的配置
type ObservabilityOptions struct {
	TraceInjectionMode TraceInjectionMode
	CustomTraceHeader  string   // 追踪 ID 的自定义头名称
	SkipPaths          []string // 跳过日志记录的路径（支持通配符）
}

// Option 是用于配置中间件的函数式选项
type Option func(*ObservabilityOptions)

// WithTraceInjection 配置追踪注入模式
func WithTraceInjection(mode TraceInjectionMode) Option {
	return func(o *ObservabilityOptions) {
		o.TraceInjectionMode = mode
	}
}

// WithCustomTraceHeader 设置追踪 ID 的自定义头名称
func WithCustomTraceHeader(headerName string) Option {
	return func(o *ObservabilityOptions) {
		o.CustomTraceHeader = headerName
	}
}

// WithSkipPaths 配置要跳过的路径（支持精确匹配和通配符）
func WithSkipPaths(paths ...string) Option {
	return func(o *ObservabilityOptions) {
		o.SkipPaths = append(o.SkipPaths, paths...)
	}
}

// WithSkipMetrics 是跳过常见指标端点的便捷函数
func WithSkipMetrics() Option {
	return func(o *ObservabilityOptions) {
		commonPaths := []string{
			"/health",
			"/healthz",
			"/health/*",
			"/ready",
			"/readiness",
			"/live",
			"/liveness",
			"/metrics",
			"/prometheus",
			"/status",
			"/ping",
			"/version",
			"/info",
			"/favicon.ico",
			"/robots.txt",
		}
		o.SkipPaths = append(o.SkipPaths, commonPaths...)
	}
}

// Observability 中间件，支持可配置的追踪注入
func Observability(opts ...Option) gin.HandlerFunc {
	// 默认配置
	config := &ObservabilityOptions{
		TraceInjectionMode: InjectTraceIDOnly,
		SkipPaths:          []string{"/metrics"}, // 默认跳过 /metrics
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// 检查此请求是否应该被跳过
		shouldSkip := shouldSkipPath(c.Request.URL.Path, c.Request.Method, config.SkipPaths)
		if shouldSkip {
			c.Next()
			return
		}

		// 尽早提取追踪信息
		span := trace.SpanFromContext(ctx)
		spanCtx := span.SpanContext()

		// 根据配置注入追踪头（除非跳过追踪）
		injectTraceHeaders(c, spanCtx, config)

		var requestBody string
		var responseBuffer bytes.Buffer

		// 仅在需要记录日志且启用调试时捕获请求体
		isDebugLevel := isDebugEnabled()

		if isDebugLevel && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if isDebugLevel {
			writer := &bodyCaptureWriter{ResponseWriter: c.Writer, body: &responseBuffer}
			c.Writer = writer
		}

		c.Next()

		duration := time.Since(start).Seconds()

		// 构建结构化日志
		httpData := map[string]any{
			"request": map[string]any{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			},
			"response": map[string]any{
				"status_code": c.Writer.Status(),
			},
		}

		if isDebugLevel {
			httpData["request"].(map[string]any)["body"] = map[string]any{
				"content": requestBody,
				"bytes":   len(requestBody),
			}

			httpData["response"].(map[string]any)["body"] = map[string]any{
				"content": responseBuffer.String(),
				"bytes":   responseBuffer.Len(),
			}
		}

		logLevel := slog.LevelInfo
		if isDebugLevel {
			logLevel = slog.LevelDebug
		}

		slog.Log(ctx, logLevel, "HTTP request completed",
			"duration_sec", duration,
			"source", map[string]any{"ip": c.ClientIP()},
			"http", httpData,
			"user", map[string]any{"agent": c.Request.UserAgent()},
			"trace", map[string]any{"id": spanCtx.TraceID().String()},
			"span", map[string]any{"id": spanCtx.SpanID().String()},
		)
	}
}

// shouldSkipPath 根据配置检查路径是否应该被跳过
func shouldSkipPath(path, method string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if matchPath(path, method, skipPath) {
			return true
		}
	}
	return false
}

// matchPath 将请求路径与跳过模式进行匹配
func matchPath(requestPath, method, pattern string) bool {
	// 处理特定方法的模式，如 "GET /metrics"
	if strings.Contains(pattern, " ") {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) == 2 {
			patternMethod := strings.ToUpper(strings.TrimSpace(parts[0]))
			patternPath := strings.TrimSpace(parts[1])

			if patternMethod != strings.ToUpper(method) {
				return false
			}
			return matchPathPattern(requestPath, patternPath)
		}
	}

	// 处理仅路径的模式
	return matchPathPattern(requestPath, pattern)
}

// matchPathPattern 将路径与模式进行匹配（支持通配符）
func matchPathPattern(path, pattern string) bool {
	// 精确匹配
	if path == pattern {
		return true
	}

	// 通配符支持
	if strings.Contains(pattern, "*") {
		return matchWildcard(path, pattern)
	}

	// 前缀匹配（如果模式以 / 结尾）
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}

	return false
}

// matchWildcard 执行简单的通配符匹配
func matchWildcard(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 简单的前缀/后缀通配符匹配
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(text, substr)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	return text == pattern
}

// injectTraceHeaders 根据配置注入追踪头
func injectTraceHeaders(c *gin.Context, spanCtx trace.SpanContext, config *ObservabilityOptions) {
	if !spanCtx.IsValid() {
		return
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	switch config.TraceInjectionMode {
	case InjectW3CTraceContext:
		// W3C 追踪上下文格式：version-trace_id-parent_id-trace_flags
		traceFlags := "01" // 已采样
		if !spanCtx.IsSampled() {
			traceFlags = "00" // 未采样
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

	case InjectTraceIDOnly:
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectBoth:
		// W3C 格式
		traceFlags := "01"
		if !spanCtx.IsSampled() {
			traceFlags = "00"
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

		// 简单追踪 ID
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectNone:
		// 不执行任何操作
	}
}

// 常见配置的便捷函数

// ObservabilityWithW3CTraceContext 创建带 W3C 追踪上下文的中间件
func ObservabilityWithW3CTraceContext() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

// ObservabilityWithTraceID 创建带简单追踪 ID 的中间件
func ObservabilityWithTraceID() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

// ObservabilityWithCustomHeader 创建带自定义头的中间件
func ObservabilityWithCustomHeader(headerName string) gin.HandlerFunc {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}

// ObservabilitySkipMetrics 创建跳过常见指标端点的中间件
func ObservabilitySkipMetrics() gin.HandlerFunc {
	return Observability(WithSkipMetrics())
}

// ObservabilityWithSkipPaths 创建带自定义跳过路径的中间件
func ObservabilityWithSkipPaths(paths ...string) gin.HandlerFunc {
	return Observability(WithSkipPaths(paths...))
}

// bodyCaptureWriter 捕获并复制写入的响应体
type bodyCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isDebugEnabled 检查是否为全局日志记录器启用了调试日志
func isDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}

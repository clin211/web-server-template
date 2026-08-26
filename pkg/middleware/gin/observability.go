package gin

import (
	"fmt"
	"log/slog"
	"net/http"
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
	SkipPaths          []string // 短路整个中间件的路径：同时跳过访问日志和响应追踪头注入
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

		// skip 会短路整个中间件，而不只是压低访问日志；被跳过的响应也不会注入追踪头。
		shouldSkip := shouldSkipPath(c.Request.URL.Path, c.Request.Method, config.SkipPaths)
		if shouldSkip {
			c.Next()
			return
		}

		// 尽早提取追踪信息
		span := trace.SpanFromContext(c.Request.Context())
		spanCtx := span.SpanContext()

		// 根据配置注入追踪头（除非跳过追踪）
		injectTraceHeaders(c, spanCtx, config)

		c.Next()

		// 优先记录 Gin 路由模板，避免用户 ID 等动态路径造成高基数或泄露敏感标识。
		// 只有未匹配路由没有模板，此时才回退原始 path，运维侧应避免把该字段用作指标标签。
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		statusCode := c.Writer.Status()
		// 请求体、响应体、query、完整 headers 和 cookie 有意不记录，防止密码和 token
		// 进入日志。ClientIP 依赖 Gin trusted proxies 配置，仅用于观测，不能作为鉴权或审计身份。
		slog.Log(c.Request.Context(), requestLogLevel(statusCode), "HTTP request completed",
			"event.name", "http.request.completed",
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
			"client.address", c.ClientIP(),
			"http.request.method", c.Request.Method,
			"http.route", route,
			"http.response.status_code", statusCode,
			"user_agent.original", c.Request.UserAgent(),
		)
	}
}

func requestLogLevel(statusCode int) slog.Level {
	switch {
	case statusCode >= http.StatusInternalServerError:
		return slog.LevelError
	case statusCode >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
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

// matchWildcard 执行简单的字符串通配符匹配。
// 这里不是 glob 或 path.Match：* 可以跨越 /，不支持转义，只处理前缀、后缀和包含关系。
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

// injectTraceHeaders 根据当前服务 span 注入响应追踪头，并沿用其采样标志。
// CustomTraceHeader 只影响 InjectTraceIDOnly 和 InjectBoth；W3C 与 None 模式会忽略它。
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

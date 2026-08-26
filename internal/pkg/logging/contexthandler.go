// Package logging 提供应用日志的上下文字段注入能力。
package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
)

// ContextHandler 将请求、用户和追踪标识添加到每条结构化日志中。
type ContextHandler struct {
	next slog.Handler
}

// NewContextHandler 使用给定的下游 Handler 创建 ContextHandler。
func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{next: next}
}

// Enabled 报告给定级别的日志是否启用。
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle 注入非空的关联字段并将日志交给下游 Handler。
func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	// 仅允许 request/user/trace/span ID 进入日志，不展开任意 context 值，避免凭据泄露
	// 和不可控的字段增长。
	if requestID := contextx.RequestID(ctx); requestID != "" {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	if userID := contextx.UserID(ctx); userID != "" {
		record.AddAttrs(slog.String("user_id", userID))
	}

	// 有效 OTel SpanContext 代表当前真实 span，必须优先于兼容字段中的 trace ID，
	// 防止外部传入或陈旧的 fallback 值覆盖当前链路；fallback 只服务无活动 span 的异步关联。
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	} else if traceID := contextx.TraceID(ctx); traceID != "" {
		record.AddAttrs(slog.String("trace_id", traceID))
	}

	return h.next.Handle(ctx, record)
}

// WithAttrs 返回带有附加属性的独立 ContextHandler。
// 保留包装层可确保派生 logger 仍在每条记录处理时读取当前 context，而不是提前固化关联字段。
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup 返回带有属性组的独立 ContextHandler。
// 保留包装层可确保分组后的 logger 不会丢失请求和追踪关联字段。
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{next: h.next.WithGroup(name)}
}

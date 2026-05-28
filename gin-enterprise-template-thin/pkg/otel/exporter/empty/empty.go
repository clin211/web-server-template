package tracing

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter 实现了 sdktrace.SpanExporter。
type Exporter struct{}

// 确保 Exporter 实现了 sdktrace.SpanExporter。
var _ sdktrace.SpanExporter = (*Exporter)(nil)

// ExportSpans 使用 slog 记录已完成的 span。
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

// Shutdown 如果需要则关闭 logger（此处为空操作）。
func (e *Exporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewEmptyExporter 创建并返回一个新的 Exporter 实例，该实例满足
// OpenTelemetry sdktrace.SpanExporter 接口，但不输出、存储
// 或转发任何 span 数据。
func NewEmptyExporter() *Exporter {
	return &Exporter{}
}

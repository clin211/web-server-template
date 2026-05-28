package tracing

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// FakeExporter 实现了 sdktrace.SpanExporter，但会丢弃或记录 span。
// 您可以传入自定义处理程序在测试中检查 span。
type FakeExporter struct {
	LogSpans bool                                                  // 如果为 true，则将 span 打印到 stdout
	Handle   func(ctx context.Context, span sdktrace.ReadOnlySpan) // 可选的 span 处理程序
}

// 确保 FakeExporter 实现了 sdktrace.SpanExporter。
var _ sdktrace.SpanExporter = (*FakeExporter)(nil)

// ExportSpans 在 span 完成时由 SDK 调用。
// 在这个伪造的导出器中，我们要么记录它们，要么丢弃它们。
func (f *FakeExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		if f.Handle != nil {
			f.Handle(ctx, span)
		}
		if f.LogSpans {
			fmt.Printf("[FAKE EXPORTER] span finished: name=%s traceID=%s spanID=%s status=%v duration=%v\n",
				span.Name(),
				span.SpanContext().TraceID().String(),
				span.SpanContext().SpanID().String(),
				span.Status(),
				time.Since(span.StartTime()),
			)
		}
	}
	return nil
}

// Shutdown 在提供者或进程退出时被调用。
func (f *FakeExporter) Shutdown(ctx context.Context) error {
	if f.LogSpans {
		log.Println("[FAKE EXPORTER] shutting down")
	}
	return nil
}

// InitFakeTracer 配置 OpenTelemetry 使用伪造的 tracer provider。
// 它创建有效的 TraceID/SpanID，保持传播功能正常，但不会将 span 导出到其他地方。
func InitFakeTracer(serviceName string, logSpans bool) {
	exporter := &FakeExporter{LogSpans: logSpans}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			"fake-resource",
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment", "development"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if logSpans {
		log.Printf("[FAKE EXPORTER] Initialized fake tracer for service=%s", serviceName)
	}
}

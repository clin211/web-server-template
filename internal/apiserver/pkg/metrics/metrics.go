package metrics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var errMetricsMeterRequired = errors.New("metrics meter is required")

// AsyncMetrics 记录 Outbox Relay、Consumer 和 DLQ 的低基数指标。
//
// event_type 只能来自已注册、版本化的有限事件集合，result 只能使用 Observer 契约约定的
// 固定枚举。event_id、topic、partition 和错误文本等高基数信息必须留在日志或 trace 中。
type AsyncMetrics struct {
	outboxPublishTotal      metric.Int64Counter
	outboxPublishDuration   metric.Float64Histogram
	consumerProcessTotal    metric.Int64Counter
	consumerProcessDuration metric.Float64Histogram
	consumerDLQTotal        metric.Int64Counter
	consumerDLQLastFailure  metric.Int64Gauge
}

// NewAsyncMetrics 创建异步链路指标，所有 instrument 创建错误都显式返回。
func NewAsyncMetrics(meter metric.Meter) (*AsyncMetrics, error) {
	if meter == nil {
		return nil, errMetricsMeterRequired
	}
	outboxTotal, err := meter.Int64Counter("outbox_publish_total", metric.WithDescription("Total Outbox publish attempts"))
	if err != nil {
		return nil, fmt.Errorf("create outbox publish counter: %w", err)
	}
	outboxDuration, err := meter.Float64Histogram("outbox_publish_duration_seconds", metric.WithUnit("s"), metric.WithDescription("Outbox publish duration"))
	if err != nil {
		return nil, fmt.Errorf("create outbox publish histogram: %w", err)
	}
	consumerTotal, err := meter.Int64Counter("consumer_process_total", metric.WithDescription("Total Kafka consumer processing attempts"))
	if err != nil {
		return nil, fmt.Errorf("create consumer process counter: %w", err)
	}
	consumerDuration, err := meter.Float64Histogram("consumer_process_duration_seconds", metric.WithUnit("s"), metric.WithDescription("Kafka consumer processing duration"))
	if err != nil {
		return nil, fmt.Errorf("create consumer process histogram: %w", err)
	}
	dlqTotal, err := meter.Int64Counter("consumer_dlq_publish_total", metric.WithDescription("Total persisted consumer DLQ records"))
	if err != nil {
		return nil, fmt.Errorf("create consumer DLQ counter: %w", err)
	}
	lastFailure, err := meter.Int64Gauge("consumer_dlq_last_failure_timestamp_seconds", metric.WithDescription("Unix timestamp of the latest consumer DLQ failure"))
	if err != nil {
		return nil, fmt.Errorf("create consumer DLQ gauge: %w", err)
	}
	return &AsyncMetrics{
		outboxPublishTotal: outboxTotal, outboxPublishDuration: outboxDuration,
		consumerProcessTotal: consumerTotal, consumerProcessDuration: consumerDuration,
		consumerDLQTotal: dlqTotal, consumerDLQLastFailure: lastFailure,
	}, nil
}

// RecordPublish 实现 outbox.Observer。
func (m *AsyncMetrics) RecordPublish(ctx context.Context, eventType, result string, duration time.Duration) {
	eventType = boundedEventType(eventType)
	attrs := metric.WithAttributes(attribute.String("event_type", eventType), attribute.String("result", result))
	m.outboxPublishTotal.Add(ctx, 1, attrs)
	// 成功率由带 result 的 counter 计算；duration 有意只按 event_type 聚合，以控制时序数量。
	m.outboxPublishDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("event_type", eventType)))
}

// RecordProcess 实现 consumer.Observer。
func (m *AsyncMetrics) RecordProcess(ctx context.Context, eventType, result string, duration time.Duration) {
	eventType = boundedEventType(eventType)
	attrs := metric.WithAttributes(attribute.String("event_type", eventType), attribute.String("result", result))
	m.consumerProcessTotal.Add(ctx, 1, attrs)
	// duration 不附加 result，避免每新增一种处理结果都成倍扩张 histogram 时序。
	m.consumerProcessDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("event_type", eventType)))
}

// RecordDLQ 实现 consumer.Observer。
func (m *AsyncMetrics) RecordDLQ(ctx context.Context, eventType, result string, failedAt time.Time) {
	eventType = boundedEventType(eventType)
	attrs := metric.WithAttributes(attribute.String("event_type", eventType), attribute.String("result", result))
	m.consumerDLQTotal.Add(ctx, 1, attrs)
	if result == "success" {
		// 只有 DLQ 已可靠持久化才推进时间戳；failedAt 表示原消息处理失败时间，
		// 不是 DLQ Relay 发布完成时间，避免告警把未接管的失败误判为已处理。
		m.consumerDLQLastFailure.Record(ctx, failedAt.Unix(), metric.WithAttributes(attribute.String("event_type", eventType)))
	}
}

// RecordLag 实现 consumer.Observer；Kafka 集群级 lag 由 Kafka exporter 提供。
func (m *AsyncMetrics) RecordLag(context.Context, string, int, int64) {}

// Initialize 在 HTTP 初始化阶段验证并注册 instruments；真正的指标记录由 AsyncRuntime
// 注入的 AsyncMetrics 完成，因此这里有意只检查构造错误。
func Initialize(_ context.Context, scope string) error {
	_, err := NewAsyncMetrics(otel.Meter(scope + ".metrics"))
	return err
}

func sanitizeMetricPrefix(scope string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_")
	prefix := strings.Trim(replacer.Replace(scope), "_")
	if prefix == "" {
		return "apiserver"
	}
	return prefix
}

func boundedEventType(t string) string {
	switch t {
	case "report.generated.v1", "report.generated.v1.dlq", "example.created.v1":
		return t
	default:
		return "unknown"
	}
}

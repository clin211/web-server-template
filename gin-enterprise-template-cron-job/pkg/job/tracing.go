package job

import (
	"context"
	"encoding/json"
	"maps"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// MetadataExecutionID 是任务元数据中执行记录 ID 的键名。
	MetadataExecutionID = "executionID"
	// MetadataScheduledTaskID 是任务元数据中定时任务 ID 的键名。
	MetadataScheduledTaskID = "scheduledTaskID"
)

type taskEnvelope struct {
	Payload  json.RawMessage   `json:"payload"`
	Trace    map[string]string `json:"trace,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// InjectTraceContext 将当前上下文中的链路追踪信息注入到字符串映射。
func InjectTraceContext(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}

	return maps.Clone(carrier)
}

// ExtractTraceContext 从字符串映射中提取链路追踪信息并注入到上下文中。
func ExtractTraceContext(ctx context.Context, traceContext map[string]string) context.Context {
	if len(traceContext) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(traceContext))
}

// UnmarshalEnvelope 解析队列中的任务信封负载。
func UnmarshalEnvelope(payload []byte) (taskEnvelope, error) {
	var envelope taskEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return taskEnvelope{}, err
	}
	if len(envelope.Payload) == 0 {
		envelope.Payload = json.RawMessage("{}")
	}
	if envelope.Metadata == nil {
		envelope.Metadata = map[string]string{}
	}
	return envelope, nil
}

// Package outbox 提供事务性 Outbox 事件及发布 Relay。
package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	errEventIDRequired         = errors.New("event ID is required")
	errEventTypeRequired       = errors.New("event type is required")
	errProducerRequired        = errors.New("producer is required")
	errAggregateTypeRequired   = errors.New("aggregate type is required")
	errAggregateIDRequired     = errors.New("aggregate ID is required")
	errOccurredAtRequired      = errors.New("event occurred time is required")
	errEnvelopePayloadRequired = errors.New("event envelope payload is required")
	errEnvelopeMalformed       = errors.New("event envelope is malformed")
)

// Event 是 Outbox Repository 与消息 Publisher 之间的稳定事件模型。
type Event struct {
	EventID       string
	EventType     string
	Producer      string
	AggregateType string
	AggregateID   string
	TraceID       string
	OccurredAt    time.Time
	Payload       []byte
	Attempts      int
}

// NewEventInput 包含创建事件 envelope 所需的字段。
type NewEventInput struct {
	EventID       string
	EventType     string
	Producer      string
	AggregateType string
	AggregateID   string
	TraceID       string
	OccurredAt    time.Time
	Payload       any
}

// Envelope 是生产者和消费者共享的版本化事件契约。
type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Producer      string          `json:"producer"`
	TraceID       string          `json:"trace_id,omitempty"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
}

// NewEvent 校验输入并创建包含版本化 envelope 的事件。
func NewEvent(input NewEventInput) (Event, error) {
	if input.EventID == "" {
		return Event{}, errEventIDRequired
	}
	if input.EventType == "" {
		return Event{}, errEventTypeRequired
	}
	if input.Producer == "" {
		return Event{}, errProducerRequired
	}
	if input.AggregateType == "" {
		return Event{}, errAggregateTypeRequired
	}
	if input.AggregateID == "" {
		return Event{}, errAggregateIDRequired
	}
	if input.Payload == nil {
		return Event{}, errEnvelopePayloadRequired
	}
	// 在创建边界固化业务发生时间，确保数据库重试或 Relay 延迟不会把发布时间误当作发生时间。
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}

	// 先序列化业务 payload 再嵌入 envelope，保证数据库持久化、Kafka 发布和消费者
	// 身份哈希使用完全相同的契约字节，避免各层分别编码导致幂等判断漂移。
	eventPayload, err := json.Marshal(input.Payload)
	if err != nil {
		return Event{}, fmt.Errorf("marshal event payload: %w", err)
	}
	payload, err := json.Marshal(Envelope{
		EventID:       input.EventID,
		EventType:     input.EventType,
		OccurredAt:    input.OccurredAt,
		Producer:      input.Producer,
		TraceID:       input.TraceID,
		AggregateType: input.AggregateType,
		AggregateID:   input.AggregateID,
		Payload:       eventPayload,
	})
	if err != nil {
		return Event{}, fmt.Errorf("marshal event envelope: %w", err)
	}

	return Event{
		EventID:       input.EventID,
		EventType:     input.EventType,
		Producer:      input.Producer,
		AggregateType: input.AggregateType,
		AggregateID:   input.AggregateID,
		TraceID:       input.TraceID,
		OccurredAt:    input.OccurredAt,
		Payload:       payload,
	}, nil
}

// ParseEnvelope 严格解析并校验事件 envelope 的必填字段。
// 在路由业务 handler 前拒绝不完整契约，避免无法稳定标识的消息绕过 Inbox 幂等边界。
func ParseEnvelope(payload []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("%w: %w", errEnvelopeMalformed, err)
	}
	if envelope.EventID == "" {
		return Envelope{}, errEventIDRequired
	}
	if envelope.EventType == "" {
		return Envelope{}, errEventTypeRequired
	}
	if envelope.Producer == "" {
		return Envelope{}, errProducerRequired
	}
	if envelope.AggregateType == "" {
		return Envelope{}, errAggregateTypeRequired
	}
	if envelope.AggregateID == "" {
		return Envelope{}, errAggregateIDRequired
	}
	if envelope.OccurredAt.IsZero() {
		return Envelope{}, errOccurredAtRequired
	}
	if len(envelope.Payload) == 0 || string(envelope.Payload) == "null" {
		return Envelope{}, errEnvelopePayloadRequired
	}
	return envelope, nil
}

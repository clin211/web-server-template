package outbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

var (
	errKafkaMessageWriterRequired = errors.New("kafka message writer is required")
	errMultiplePublisherTopics    = errors.New("at most one Kafka publisher topic may be specified")
)

// MessageWriter 定义 KafkaPublisher 所需的最小写入能力。
type MessageWriter interface {
	WriteMessages(ctx context.Context, messages ...kafka.Message) error
}

// KafkaPublisher 将 Outbox Event 映射为 Kafka Message。
type KafkaPublisher struct {
	writer      MessageWriter
	targetTopic string
}

// NewKafkaPublisher 使用同步 Kafka writer 创建 Publisher。
// targetTopic 仅用于需要显式记录目标 topic 的多 topic publisher；普通 writer
// 已经固定 topic 时可省略该参数。
func NewKafkaPublisher(writer MessageWriter, targetTopic ...string) (*KafkaPublisher, error) {
	if writer == nil {
		return nil, errKafkaMessageWriterRequired
	}
	if len(targetTopic) > 1 {
		return nil, errMultiplePublisherTopics
	}
	return &KafkaPublisher{writer: writer, targetTopic: firstTopic(targetTopic)}, nil
}

// Publish 使用 aggregate ID 作为 key 发布事件，并传播事件关联头。
func (p *KafkaPublisher) Publish(ctx context.Context, event Event) error {
	headers := []kafka.Header{
		{Key: "event_id", Value: []byte(event.EventID)},
		{Key: "event_type", Value: []byte(event.EventType)},
	}
	if event.TraceID != "" {
		headers = append(headers, kafka.Header{Key: "trace_id", Value: []byte(event.TraceID)})
	}
	// aggregate_id 作为 key 可让同一聚合的事件稳定落到同一 partition，保留聚合内
	// 相对顺序；改用随机值或 event_id 会破坏该顺序保证。
	message := kafka.Message{
		Key:     []byte(event.AggregateID),
		Value:   event.Payload,
		Headers: headers,
		Time:    event.OccurredAt,
	}
	if p.targetTopic != "" {
		message.Topic = p.targetTopic
	}
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write Kafka message: %w", err)
	}
	return nil
}

func firstTopic(topics []string) string {
	if len(topics) == 0 {
		return ""
	}
	return topics[0]
}

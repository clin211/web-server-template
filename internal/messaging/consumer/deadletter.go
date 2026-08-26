package consumer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	errDeadLetterRepositoryRequired = errors.New("dead-letter repository is required")
	errInboxFinalizerRequired       = errors.New("inbox finalizer is required")
	errDeadLetterTopicRequired      = errors.New("dead-letter topic is required")
)

// DeadLetter 保存原始消息、消费位置和最终失败原因。
type DeadLetter struct {
	EventID       string         `json:"event_id,omitempty"`
	EventType     string         `json:"event_type,omitempty"`
	Topic         string         `json:"topic"`
	Partition     int            `json:"partition"`
	Offset        int64          `json:"offset"`
	Key           []byte         `json:"key,omitempty"`
	Value         []byte         `json:"value"`
	Headers       []kafka.Header `json:"headers,omitempty"`
	Attempts      int            `json:"attempts"`
	FailureKind   string         `json:"failure_kind"`
	Error         string         `json:"error"`
	FailedAt      time.Time      `json:"failed_at"`
	FinalizeInbox bool           `json:"-"`
}

func deadLetterFromMessage(message kafka.Message, attempts int, processErr error, failedAt time.Time) DeadLetter {
	failureKind := "retry_exhausted"
	if IsPermanent(processErr) {
		failureKind = "permanent"
		if errors.Is(processErr, ErrUnsupportedEventType) {
			failureKind = "unsupported"
		}
	}
	if errors.Is(processErr, ErrEventIdentityConflict) {
		failureKind = "identity_conflict"
	}
	return DeadLetter{
		EventID:     headerValue(message.Headers, "event_id"),
		EventType:   headerValue(message.Headers, "event_type"),
		Topic:       message.Topic,
		Partition:   message.Partition,
		Offset:      message.Offset,
		Key:         append([]byte(nil), message.Key...),
		Value:       append([]byte(nil), message.Value...),
		Headers:     append([]kafka.Header(nil), message.Headers...),
		Attempts:    attempts,
		FailureKind: failureKind,
		Error:       processErr.Error(),
		FailedAt:    failedAt.UTC(),
		// 只有 Claim 成功后的失败才携带 Inbox attempt。解析或身份校验等 Claim 前错误
		// 也要进入 DLQ，但不能尝试终结一个尚不存在的 Inbox 行。
		FinalizeInbox: FailureAttempt(processErr) > 0,
	}
}

func headerValue(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

// MarshalJSONPayload 将 DeadLetter 编码为稳定的 JSON 载荷。
func (d DeadLetter) MarshalJSONPayload() ([]byte, error) {
	payload, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("marshal dead-letter payload: %w", err)
	}
	return payload, nil
}

// StoredDeadLetter 是持久化 DLQ Relay 所需的完整记录。
type StoredDeadLetter struct {
	DeadLetterID string
	ConsumerName string
	DLQTopic     string
	DeadLetter   DeadLetter
}

// DeadLetterRepository 持久化待发布的 DLQ 记录。
type DeadLetterRepository interface {
	Create(ctx context.Context, record StoredDeadLetter) error
}

// InboxFinalizer 将持有正确 attempt 的 Inbox 事件标记为 dead_lettered。
type InboxFinalizer interface {
	MarkDeadLettered(ctx context.Context, consumerName, eventID string, attempt int, processedAt time.Time) error
}

// PersistentDeadLetterPublisher 在一个数据库事务内保存 DLQ 并终结 Inbox。
type PersistentDeadLetterPublisher struct {
	transactioner Transactioner
	repository    DeadLetterRepository
	inbox         InboxFinalizer
	consumerName  string
	dlqTopic      string
}

// NewPersistentDeadLetterPublisher 创建数据库优先的 DLQ Publisher。
func NewPersistentDeadLetterPublisher(
	transactioner Transactioner,
	repository DeadLetterRepository,
	inbox InboxFinalizer,
	consumerName string,
	dlqTopic string,
) (*PersistentDeadLetterPublisher, error) {

	if transactioner == nil {
		return nil, errTransactionerRequired
	}
	if repository == nil {
		return nil, errDeadLetterRepositoryRequired
	}
	if inbox == nil {
		return nil, errInboxFinalizerRequired
	}
	if consumerName == "" {
		return nil, errConsumerNameRequired
	}
	if dlqTopic == "" {
		return nil, errDeadLetterTopicRequired
	}
	return &PersistentDeadLetterPublisher{
		transactioner: transactioner,
		repository:    repository,
		inbox:         inbox,
		consumerName:  consumerName,
		dlqTopic:      dlqTopic,
	}, nil
}

// Publish 原子保存 DLQ 记录，并在适用时把 Inbox 标记为 dead_lettered。
func (p *PersistentDeadLetterPublisher) Publish(ctx context.Context, record DeadLetter) error {
	// deadLetterID 只由稳定的源位置生成。offset 提交失败导致消息重投时会得到同一 ID，
	// 配合数据库唯一约束保证 DLQ 记录不会重复创建。
	stored := StoredDeadLetter{
		DeadLetterID: deadLetterID(p.consumerName, record),
		ConsumerName: p.consumerName,
		DLQTopic:     p.dlqTopic,
		DeadLetter:   record,
	}
	// DLQ 记录和 Inbox 终态必须原子提交。只有该事务成功后 Consumer 才允许提交源
	// offset，因此不存在 Inbox 已终结但失败证据没有落库的中间状态。
	if err := p.transactioner.TX(ctx, func(txCtx context.Context) error {
		if err := p.repository.Create(txCtx, stored); err != nil {
			return fmt.Errorf("create dead-letter event: %w", err)
		}
		if record.FinalizeInbox {
			if err := p.inbox.MarkDeadLettered(txCtx, p.consumerName, record.EventID, record.Attempts, record.FailedAt); err != nil {
				return fmt.Errorf("mark inbox event dead-lettered: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("persist dead-letter transaction: %w", err)
	}
	return nil
}

func deadLetterID(consumerName string, record DeadLetter) string {
	identity := fmt.Sprintf("%s\x00%s\x00%d\x00%d", consumerName, record.Topic, record.Partition, record.Offset)
	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("%x", hash[:])
}

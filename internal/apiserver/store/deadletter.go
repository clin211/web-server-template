package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
)

var errDeadLetterUpdate = errors.New("update dead-letter event state")

// DeadLetterStore 定义死信原子写入和 Relay 持久化操作。
type DeadLetterStore interface {
	consumer.DeadLetterRepository
	outbox.Repository
}

type deadLetterStore struct {
	core *datastore
}

var _ DeadLetterStore = (*deadLetterStore)(nil)

func newDeadLetterStore(store *datastore) *deadLetterStore {
	return &deadLetterStore{core: store}
}

// Create 幂等保存原始消息和失败证据。
func (s *deadLetterStore) Create(ctx context.Context, stored consumer.StoredDeadLetter) error {
	record := stored.DeadLetter
	// kafka.Header 只包含字节切片和字符串，JSON 编码不会失败。
	headers, err := json.Marshal(record.Headers) //nolint:errchkjson
	if err != nil {
		return fmt.Errorf("marshal dead-letter headers: %w", err)
	}
	now := time.Now().UTC()
	modelRecord := &model.DeadLetterEventM{
		DeadLetterID:      stored.DeadLetterID,
		ConsumerName:      stored.ConsumerName,
		OriginalEventID:   stringPointer(record.EventID),
		OriginalEventType: stringPointer(record.EventType),
		SourceTopic:       record.Topic,
		SourcePartition:   int32(record.Partition),
		SourceOffset:      record.Offset,
		MessageKey:        record.Key,
		Headers:           datatypes.JSON(headers),
		Payload:           record.Value,
		TraceID:           stringPointer(headerValueFromKafka(record.Headers, "trace_id")),
		FailureKind:       record.FailureKind,
		ConsumeAttempts:   int32(record.Attempts),
		LastError:         truncateRunes(record.Error, maxOutboxErrorRunes),
		DLQTopic:          stored.DLQTopic,
		Status:            known.DeadLetterStatusPending,
		AvailableAt:       now,
	}
	// 源 topic/partition/offset 是 Kafka 投递的稳定身份。提交 offset 失败后的重投只需
	// 复用已有 DLQ 记录，不能再次制造一条待发布死信。
	result := s.core.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "consumer_name"},
			{Name: "source_topic"},
			{Name: "source_partition"},
			{Name: "source_offset"},
		},
		DoNothing: true,
	}).Create(modelRecord)
	if result.Error != nil {
		return fmt.Errorf("create dead-letter event: %w", result.Error)
	}
	return nil
}

// Claim 通过行锁领取一批可发布的死信，并增加 publish attempt fencing token。
func (s *deadLetterStore) Claim(ctx context.Context, batchSize int, leaseDuration time.Duration) ([]outbox.Event, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(leaseDuration)
	var records []model.DeadLetterEventM
	err := s.core.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var claimErr error
		records, claimErr = claimDeadLetterRecords(tx, now, leaseUntil, batchSize)
		return claimErr
	})
	if err != nil {
		return nil, fmt.Errorf("claim dead-letter events: %w", err)
	}

	events := make([]outbox.Event, 0, len(records))
	for index := range records {
		// PostgreSQL RETURNING/SELECT 得到的是 SQL 自增前快照；同步内存值后，Relay 才能
		// 把正确的 publish_attempts 作为 fencing token 传给终态更新。
		records[index].PublishAttempts++
		event, err := deadLetterEventFromModel(records[index])
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

//nolint:dupl
func claimDeadLetterRecords(tx *gorm.DB, now, leaseUntil time.Time, batchSize int) ([]model.DeadLetterEventM, error) {
	var records []model.DeadLetterEventM
	query := `
SELECT id, dead_letter_id, consumer_name, original_event_id, original_event_type,
       source_topic, source_partition, source_offset, message_key, headers, payload,
       trace_id, failure_kind, consume_attempts, last_error, dlq_topic, status,
       publish_attempts, available_at, published_at, created_at, updated_at
FROM dead_letter_event
WHERE ((status IN (?, ?) AND available_at <= ?) OR (status = ? AND available_at <= ?))
ORDER BY id
LIMIT ?
FOR UPDATE SKIP LOCKED`
	if err := tx.Raw(
		query,
		known.DeadLetterStatusPending,
		known.DeadLetterStatusFailed,
		now,
		known.DeadLetterStatusProcessing,
		now,
		batchSize,
	).Scan(&records).Error; err != nil {
		return nil, fmt.Errorf("select dead-letter events: %w", err)
	}
	if len(records) == 0 {
		return records, nil
	}

	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.DeadLetterID)
	}
	// 网络发布不持有数据库行锁；短事务先写 processing 租约并递增 publish_attempts，
	// 使崩溃后的记录可被重新领取，同时阻止旧 Relay 的迟到结果覆盖新状态。
	updates := map[string]any{
		"status":           known.DeadLetterStatusProcessing,
		"available_at":     leaseUntil,
		"updated_at":       now,
		"publish_attempts": gorm.Expr("publish_attempts + 1"),
	}
	if err := tx.Model(&model.DeadLetterEventM{}).Where("dead_letter_id IN ?", ids).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("lease dead-letter events: %w", err)
	}
	return records, nil
}

// MarkPublished 将已发送到 Kafka DLQ 的事件标记为 published。
func (s *deadLetterStore) MarkPublished(ctx context.Context, eventID string, attempt int, publishedAt time.Time) error {
	updates := map[string]any{
		"status":       known.DeadLetterStatusPublished,
		"published_at": publishedAt,
		"last_error":   "",
		"updated_at":   publishedAt,
	}
	result := s.core.DB(ctx).Model(&model.DeadLetterEventM{}).
		Where("dead_letter_id = ? AND status = ? AND publish_attempts = ?", eventID, known.DeadLetterStatusProcessing, attempt).
		Updates(updates)
	return checkDeadLetterLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

// MarkFailed 记录 DLQ 发布错误和下次可用时间。
func (s *deadLetterStore) MarkFailed(ctx context.Context, eventID string, attempt int, reason string, nextAvailableAt time.Time) error {
	updates := map[string]any{
		"status":       known.DeadLetterStatusFailed,
		"last_error":   truncateRunes(reason, maxOutboxErrorRunes),
		"available_at": nextAvailableAt,
		"updated_at":   time.Now().UTC(),
	}
	result := s.core.DB(ctx).Model(&model.DeadLetterEventM{}).
		Where("dead_letter_id = ? AND status = ? AND publish_attempts = ?", eventID, known.DeadLetterStatusProcessing, attempt).
		Updates(updates)
	return checkDeadLetterLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

// MarkExhausted 将 DLQ 发布达到重试上限的记录转为终态，避免无限重试掩盖持续故障。
func (s *deadLetterStore) MarkExhausted(ctx context.Context, eventID string, attempt int, reason string, exhaustedAt time.Time) error {
	updates := map[string]any{
		"status":     known.DeadLetterStatusDead,
		"last_error": truncateRunes(reason, maxOutboxErrorRunes),
		"updated_at": exhaustedAt,
	}
	result := s.core.DB(ctx).Model(&model.DeadLetterEventM{}).
		Where("dead_letter_id = ? AND status = ? AND publish_attempts = ?", eventID, known.DeadLetterStatusProcessing, attempt).
		Updates(updates)
	return checkDeadLetterLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

func deadLetterEventFromModel(record model.DeadLetterEventM) (outbox.Event, error) {
	var headers []kafka.Header
	if err := json.Unmarshal(record.Headers, &headers); err != nil {
		return outbox.Event{}, fmt.Errorf("unmarshal dead-letter %s headers: %w", record.DeadLetterID, err)
	}
	deadLetter := consumer.DeadLetter{
		EventID:     stringValue(record.OriginalEventID),
		EventType:   stringValue(record.OriginalEventType),
		Topic:       record.SourceTopic,
		Partition:   int(record.SourcePartition),
		Offset:      record.SourceOffset,
		Key:         record.MessageKey,
		Value:       record.Payload,
		Headers:     headers,
		Attempts:    int(record.ConsumeAttempts),
		FailureKind: record.FailureKind,
		Error:       record.LastError,
		FailedAt:    record.CreatedAt,
	}
	payload, err := deadLetter.MarshalJSONPayload()
	if err != nil {
		return outbox.Event{}, fmt.Errorf("marshal dead-letter %s: %w", record.DeadLetterID, err)
	}
	eventType := deadLetter.EventType
	if eventType == "" {
		eventType = "unknown"
	}
	aggregateID := deadLetter.EventID
	if aggregateID == "" {
		aggregateID = record.DeadLetterID
	}
	// dead_letter_id 作为新消息 ID，使租约恢复后的重复发布具有稳定身份；原 event_id
	// 优先作为 aggregate key 维持分区稳定，无法解析原身份时才使用稳定 fallback。
	return outbox.Event{
		EventID:       record.DeadLetterID,
		EventType:     eventType + ".dlq",
		Producer:      record.ConsumerName,
		AggregateType: "dead_letter",
		AggregateID:   aggregateID,
		TraceID:       stringValue(record.TraceID),
		OccurredAt:    record.CreatedAt,
		Payload:       payload,
		Attempts:      int(record.PublishAttempts),
	}, nil
}

func checkDeadLetterLeaseUpdate(eventID string, attempt int, rowsAffected int64, updateErr error) error {
	if updateErr != nil {
		return fmt.Errorf("%w for dead-letter %s attempt %d: %w", errDeadLetterUpdate, eventID, attempt, updateErr)
	}
	// 零行更新表示 status 或 publish_attempts 已被新 Relay 推进；旧 Relay 的迟到 Kafka
	// 结果不能覆盖当前租约所有者写入的状态。
	if rowsAffected != 1 {
		return fmt.Errorf("dead-letter %s attempt %d: %w", eventID, attempt, outbox.ErrLeaseLost)
	}
	return nil
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func headerValueFromKafka(headers []kafka.Header, key string) string {
	for _, header := range headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

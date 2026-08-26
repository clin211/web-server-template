package store

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
)

const maxOutboxErrorRunes = 2000

var errOutboxUpdate = errors.New("update outbox event state")

// OutboxStore 定义 Outbox 事件的事务写入和 Relay 持久化操作。
type OutboxStore interface {
	outbox.Repository
	Create(ctx context.Context, event outbox.Event) error
}

type outboxStore struct {
	core *datastore
}

var _ OutboxStore = (*outboxStore)(nil)

func newOutboxStore(store *datastore) *outboxStore {
	return &outboxStore{core: store}
}

// Create 在当前数据库上下文中持久化 Outbox 事件。
func (s *outboxStore) Create(ctx context.Context, event outbox.Event) error {
	var traceID *string
	if event.TraceID != "" {
		traceID = &event.TraceID
	}
	now := time.Now().UTC()
	record := &model.OutboxEventM{
		EventID:       event.EventID,
		EventType:     event.EventType,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		Payload:       datatypes.JSON(event.Payload),
		TraceID:       traceID,
		Status:        known.OutboxStatusPending,
		AvailableAt:   now,
	}
	if err := s.core.DB(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}
	return nil
}

// Claim 通过行锁跳过其他 Relay 已领取的事件，并为当前批次设置租约。
func (s *outboxStore) Claim(ctx context.Context, batchSize int, leaseDuration time.Duration) ([]outbox.Event, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(leaseDuration)
	var records []model.OutboxEventM

	err := s.core.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var claimErr error
		records, claimErr = claimOutboxRecords(tx, now, leaseUntil, batchSize)
		return claimErr
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	for index := range records {
		// 查询返回的是 SQL `attempts + 1` 之前的快照；这里同步内存 token，否则后续
		// MarkPublished/MarkFailed 会因 attempt 不匹配而全部报告租约丢失。
		records[index].Attempts++
	}

	events := make([]outbox.Event, 0, len(records))
	for _, record := range records {
		event, err := outboxEventFromModel(record)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

//nolint:dupl
func claimOutboxRecords(tx *gorm.DB, now, leaseUntil time.Time, batchSize int) ([]model.OutboxEventM, error) {
	var records []model.OutboxEventM
	query := `
SELECT id, event_id, event_type, aggregate_type, aggregate_id, payload, trace_id,
       status, attempts, available_at, published_at, last_error, created_at, updated_at
FROM outbox_event
WHERE ((status IN (?, ?) AND available_at <= ?) OR (status = ? AND available_at <= ?))
ORDER BY id
LIMIT ?
FOR UPDATE SKIP LOCKED`
	if err := tx.Raw(
		query,
		known.OutboxStatusPending,
		known.OutboxStatusFailed,
		now,
		known.OutboxStatusProcessing,
		now,
		batchSize,
	).Scan(&records).Error; err != nil {
		return nil, fmt.Errorf("select outbox events: %w", err)
	}
	if len(records) == 0 {
		return records, nil
	}

	eventIDs := make([]string, 0, len(records))
	for _, record := range records {
		eventIDs = append(eventIDs, record.EventID)
	}
	// 领取只在短事务内完成，真正的 Kafka I/O 在事务提交后执行。processing 记录的
	// available_at 是租约截止时间，Relay 崩溃后其他实例可在到期时恢复处理。
	updates := map[string]any{
		"status":       known.OutboxStatusProcessing,
		"available_at": leaseUntil,
		"updated_at":   now,
		"attempts":     gorm.Expr("attempts + 1"),
	}
	if err := tx.Model(&model.OutboxEventM{}).Where("event_id IN ?", eventIDs).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("lease outbox events: %w", err)
	}
	return records, nil
}

// MarkPublished 将已成功发送的事件标记为 published。
func (s *outboxStore) MarkPublished(ctx context.Context, eventID string, attempt int, publishedAt time.Time) error {
	updates := map[string]any{
		"status":       known.OutboxStatusPublished,
		"published_at": publishedAt,
		"last_error":   nil,
		"updated_at":   publishedAt,
	}
	result := s.core.DB(ctx).Model(&model.OutboxEventM{}).
		Where("event_id = ? AND status = ? AND attempts = ?", eventID, known.OutboxStatusProcessing, attempt).
		Updates(updates)
	return checkOutboxLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

// MarkFailed 记录发布错误并设置下次可用时间。
func (s *outboxStore) MarkFailed(ctx context.Context, eventID string, attempt int, reason string, nextAvailableAt time.Time) error {
	reason = truncateRunes(reason, maxOutboxErrorRunes)
	updates := map[string]any{
		"status":       known.OutboxStatusFailed,
		"last_error":   reason,
		"available_at": nextAvailableAt,
		"updated_at":   time.Now().UTC(),
	}
	result := s.core.DB(ctx).Model(&model.OutboxEventM{}).
		Where("event_id = ? AND status = ? AND attempts = ?", eventID, known.OutboxStatusProcessing, attempt).
		Updates(updates)
	return checkOutboxLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

// MarkExhausted 将达到重试上限的事件转为终态，等待运维修复后重放。
func (s *outboxStore) MarkExhausted(ctx context.Context, eventID string, attempt int, reason string, exhaustedAt time.Time) error {
	updates := map[string]any{
		"status":     known.OutboxStatusDead,
		"last_error": truncateRunes(reason, maxOutboxErrorRunes),
		"updated_at": exhaustedAt,
	}
	result := s.core.DB(ctx).Model(&model.OutboxEventM{}).
		Where("event_id = ? AND status = ? AND attempts = ?", eventID, known.OutboxStatusProcessing, attempt).
		Updates(updates)
	return checkOutboxLeaseUpdate(eventID, attempt, result.RowsAffected, result.Error)
}

func checkOutboxLeaseUpdate(eventID string, attempt int, rowsAffected int64, updateErr error) error {
	if updateErr != nil {
		return fmt.Errorf("%w for event %s attempt %d: %w", errOutboxUpdate, eventID, attempt, updateErr)
	}
	// attempt 是领取时生成的 fencing token。零行更新表示事件已被更新的租约接管，
	// 旧 Relay 的迟到发布结果不得覆盖当前所有者写入的状态。
	if rowsAffected != 1 {
		return fmt.Errorf("event %s attempt %d: %w", eventID, attempt, outbox.ErrLeaseLost)
	}
	return nil
}

func outboxEventFromModel(record model.OutboxEventM) (outbox.Event, error) {
	traceID := ""
	if record.TraceID != nil {
		traceID = *record.TraceID
	}
	// Producer 和 OccurredAt 属于原始事件契约，必须从持久化 envelope 恢复；使用当前
	// 时间或 updated_at 会篡改 Kafka message time，并让重放事件失去原发生时间。
	envelope, err := outbox.ParseEnvelope(record.Payload)
	if err != nil {
		return outbox.Event{}, fmt.Errorf("parse outbox event %s envelope: %w", record.EventID, err)
	}
	return outbox.Event{
		EventID:       record.EventID,
		EventType:     record.EventType,
		Producer:      envelope.Producer,
		AggregateType: record.AggregateType,
		AggregateID:   record.AggregateID,
		TraceID:       traceID,
		OccurredAt:    envelope.OccurredAt,
		Payload:       []byte(record.Payload),
		Attempts:      int(record.Attempts),
	}, nil
}

func truncateRunes(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum])
}

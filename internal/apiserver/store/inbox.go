package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
)

var errInboxUpdate = errors.New("update inbox event state")
var errInboxUnknownStatus = errors.New("unknown inbox event status")

// InboxStore 定义 Kafka Consumer 的 Inbox 状态机操作。
type InboxStore interface {
	consumer.InboxRepository
	MarkDeadLettered(ctx context.Context, consumerName, eventID string, attempt int, processedAt time.Time) error
}

type inboxStore struct {
	core *datastore
}

var _ InboxStore = (*inboxStore)(nil)

func newInboxStore(store *datastore) *inboxStore {
	return &inboxStore{core: store}
}

// Claim 新建或重新领取 Inbox 事件，并返回持久化 attempt fencing token。
//
//nolint:gocognit,gocyclo
func (s *inboxStore) Claim(ctx context.Context, identity consumer.MessageIdentity, leaseDuration time.Duration) (consumer.Claim, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(leaseDuration)
	var claim consumer.Claim
	err := s.core.DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 先依赖唯一约束竞争首次插入，避免两个 worker 都认为自己创建了处理记录。
		// 冲突方随后锁定已有行，并依据终态或租约时间决定跳过、等待还是接管。
		record, inserted, err := insertInboxEvent(tx, identity, leaseUntil, now)
		if err != nil {
			return err
		}
		if inserted {
			claim = consumer.Claim{Disposition: consumer.ClaimAcquired, Attempt: int(record.Attempts)}
			return nil
		}

		record, err = lockInboxEvent(tx, identity)
		if err != nil {
			return err
		}
		// event_id 只能标识同一份业务事实；相同 ID 携带不同类型或 payload 不能静默
		// 去重，否则会掩盖生产者协议错误。
		if record.EventType != identity.EventType || !bytes.Equal(record.PayloadHash, identity.PayloadHash) {
			return fmt.Errorf("consumer %s event %s: %w", identity.ConsumerName, identity.EventID, consumer.ErrEventIdentityConflict)
		}
		// 源位置不参与同一 event_id 的一致性校验：生产者在确认不明确时可能用相同
		// event_id 和 payload 重新发布到新 offset，这是合法重投，应由 Inbox 终态吸收。
		switch record.Status {
		case known.InboxStatusCompleted:
			claim = consumer.Claim{Disposition: consumer.ClaimCompleted, Attempt: int(record.Attempts)}
			return nil
		case known.InboxStatusDeadLettered:
			claim = consumer.Claim{Disposition: consumer.ClaimDeadLettered, Attempt: int(record.Attempts)}
			return nil
		case known.InboxStatusProcessing, known.InboxStatusFailed:
			// available_at 在 processing 状态表示租约截止时间，在 failed 状态表示重试
			// 开放时间；两者到期前都不能重新领取或递增 attempt。
			if record.AvailableAt.After(now) {
				claim = consumer.Claim{Disposition: consumer.ClaimBusy, Attempt: int(record.Attempts)}
				return nil
			}
		default:
			return fmt.Errorf("%w: consumer %s event %s status %q", errInboxUnknownStatus, identity.ConsumerName, identity.EventID, record.Status)
		}

		// 每次重新领取都递增 attempt，它同时是 fencing token。后续状态更新必须携带
		// 当前 token，租约过期前启动的旧 worker 因 token 不匹配而无法写入终态。
		attempt := int(record.Attempts) + 1
		updates := map[string]any{
			"status":       known.InboxStatusProcessing,
			"attempts":     attempt,
			"available_at": leaseUntil,
			"last_error":   nil,
			"updated_at":   now,
		}
		if err := tx.Model(&model.InboxEventM{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("renew inbox event lease: %w", err)
		}
		claim = consumer.Claim{Disposition: consumer.ClaimAcquired, Attempt: attempt}
		return nil
	})
	if err != nil {
		return consumer.Claim{}, fmt.Errorf("claim inbox event: %w", err)
	}
	return claim, nil
}

func insertInboxEvent(tx *gorm.DB, identity consumer.MessageIdentity, leaseUntil, now time.Time) (model.InboxEventM, bool, error) {
	var record model.InboxEventM
	query := `
INSERT INTO inbox_event (
    consumer_name, event_id, event_type, source_topic, source_partition, source_offset,
    payload_hash, status, attempts, available_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)
ON CONFLICT DO NOTHING
RETURNING id, consumer_name, event_id, event_type, source_topic, source_partition, source_offset,
          payload_hash, status, attempts, available_at, processed_at, last_error, created_at, updated_at`
	result := tx.Raw(
		query,
		identity.ConsumerName,
		identity.EventID,
		identity.EventType,
		identity.SourceTopic,
		identity.SourcePartition,
		identity.SourceOffset,
		identity.PayloadHash,
		known.InboxStatusProcessing,
		leaseUntil,
		now,
		now,
	).Scan(&record)
	if result.Error != nil {
		return model.InboxEventM{}, false, fmt.Errorf("insert inbox event: %w", result.Error)
	}
	return record, record.ID != 0, nil
}

func lockInboxEvent(tx *gorm.DB, identity consumer.MessageIdentity) (model.InboxEventM, error) {
	var record model.InboxEventM
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("consumer_name = ? AND event_id = ?", identity.ConsumerName, identity.EventID).
		Take(&record).Error
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.InboxEventM{}, fmt.Errorf("select inbox event: %w", err)
	}

	// event_id 未命中但源位置唯一约束发生过冲突，说明同一个 Kafka offset 被声明成了
	// 另一事件。按源位置再次查询是为了把数据库冲突转换成明确的协议错误。
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"consumer_name = ? AND source_topic = ? AND source_partition = ? AND source_offset = ?",
			identity.ConsumerName,
			identity.SourceTopic,
			identity.SourcePartition,
			identity.SourceOffset,
		).
		Take(&record).Error
	if err != nil {
		return model.InboxEventM{}, fmt.Errorf("select inbox event by source position: %w", err)
	}
	return model.InboxEventM{}, fmt.Errorf(
		"source position belongs to event %s instead of %s: %w",
		record.EventID,
		identity.EventID,
		consumer.ErrEventIdentityConflict,
	)
}

// MarkCompleted 将当前 attempt 的事件标记为 completed。
//
// 调用方必须传入业务事务派生的 txCtx，使业务副作用和 completed 状态只有一个提交点；
// 使用普通 context 会拆开两个事务，破坏 Inbox 对重复副作用的最终屏障。
func (s *inboxStore) MarkCompleted(ctx context.Context, consumerName, eventID string, attempt int, processedAt time.Time) error {
	updates := map[string]any{
		"status":       known.InboxStatusCompleted,
		"processed_at": processedAt,
		"last_error":   nil,
		"updated_at":   processedAt,
	}
	result := s.core.DB(ctx).Model(&model.InboxEventM{}).
		Where("consumer_name = ? AND event_id = ? AND status = ? AND attempts = ?", consumerName, eventID, known.InboxStatusProcessing, attempt).
		Updates(updates)
	return checkInboxLeaseUpdate(consumerName, eventID, attempt, result.RowsAffected, result.Error)
}

// MarkFailed 记录处理错误并释放当前 attempt 的租约。
func (s *inboxStore) MarkFailed(ctx context.Context, consumerName, eventID string, attempt int, reason string, nextAvailableAt time.Time) error {
	updates := map[string]any{
		"status":       known.InboxStatusFailed,
		"last_error":   truncateRunes(reason, maxOutboxErrorRunes),
		"available_at": nextAvailableAt,
		"updated_at":   time.Now().UTC(),
	}
	result := s.core.DB(ctx).Model(&model.InboxEventM{}).
		Where("consumer_name = ? AND event_id = ? AND status = ? AND attempts = ?", consumerName, eventID, known.InboxStatusProcessing, attempt).
		Updates(updates)
	return checkInboxLeaseUpdate(consumerName, eventID, attempt, result.RowsAffected, result.Error)
}

// MarkDeadLettered 将已可靠写入 DLQ 的当前 attempt 标记为终态。
//
// processing 和 failed 都允许进入该终态：失败原因可能已先用同一 attempt 持久化，随后
// DLQ 事务再完成终结；attempt 条件仍会拒绝已经被新 worker 接管的旧结果。
func (s *inboxStore) MarkDeadLettered(ctx context.Context, consumerName, eventID string, attempt int, processedAt time.Time) error {
	updates := map[string]any{
		"status":       known.InboxStatusDeadLettered,
		"processed_at": processedAt,
		"updated_at":   processedAt,
	}
	result := s.core.DB(ctx).Model(&model.InboxEventM{}).
		Where("consumer_name = ? AND event_id = ? AND status IN ? AND attempts = ?", consumerName, eventID, []string{known.InboxStatusProcessing, known.InboxStatusFailed}, attempt).
		Updates(updates)
	return checkInboxLeaseUpdate(consumerName, eventID, attempt, result.RowsAffected, result.Error)
}

func checkInboxLeaseUpdate(consumerName, eventID string, attempt int, rowsAffected int64, updateErr error) error {
	if updateErr != nil {
		return fmt.Errorf("%w for consumer %s event %s attempt %d: %w", errInboxUpdate, consumerName, eventID, attempt, updateErr)
	}
	// 零行更新不是普通的“记录不存在”：它通常表示状态或 attempt 已被新 worker 推进，
	// 必须作为租约丢失返回，阻止旧 worker 提交 Kafka offset。
	if rowsAffected != 1 {
		return fmt.Errorf("consumer %s event %s attempt %d: %w", consumerName, eventID, attempt, consumer.ErrInboxLeaseLost)
	}
	return nil
}

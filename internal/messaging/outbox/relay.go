package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

const defaultMaxRetryDelay = 5 * time.Minute
const defaultMaxAttempts = 20

var (
	errRepositoryRequired       = errors.New("outbox repository is required")
	errPublisherRequired        = errors.New("outbox publisher is required")
	errBatchSizeNotPositive     = errors.New("outbox batch size must be positive")
	errLeaseDurationNotPositive = errors.New("outbox lease duration must be positive")
	errRetryDelayNotPositive    = errors.New("outbox retry delay must be positive")
)

// ErrLeaseLost 表示事件租约已过期或已被其他 Relay 重新领取。
var ErrLeaseLost = errors.New("outbox event lease lost")

// Repository 定义 Relay 对 Outbox 持久层的最小依赖。
type Repository interface {
	Claim(ctx context.Context, batchSize int, leaseDuration time.Duration) ([]Event, error)
	MarkPublished(ctx context.Context, eventID string, attempt int, publishedAt time.Time) error
	MarkFailed(ctx context.Context, eventID string, attempt int, reason string, nextAvailableAt time.Time) error
}

// ExhaustionRepository 可选地把达到重试上限的事件转为终态。旧仓储若未实现该
// 接口仍会保留 failed 状态并记录告警，因此不会破坏已有实现。
type ExhaustionRepository interface {
	MarkExhausted(ctx context.Context, eventID string, attempt int, reason string, exhaustedAt time.Time) error
}

// Publisher 定义可靠发布单个事件的能力。
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Observer 记录 Relay 的发布结果，不携带事件 ID 等高基数字段。
type Observer interface {
	RecordPublish(ctx context.Context, eventType, result string, duration time.Duration)
}

// RelayOptions 配置 Outbox Relay 的批次和重试节奏。
type RelayOptions struct {
	BatchSize     int
	PollInterval  time.Duration
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
	MaxAttempts   int
	Now           func() time.Time
	Jitter        func(time.Duration) time.Duration
	Observer      Observer
}

// Relay 领取并发布待处理的 Outbox 事件。
type Relay struct {
	repository Repository
	publisher  Publisher
	options    RelayOptions
}

// NewRelay 校验依赖和选项并创建 Relay。
//
//nolint:gocognit,gocyclo
func NewRelay(repository Repository, publisher Publisher, options RelayOptions) (*Relay, error) {
	if repository == nil {
		return nil, errRepositoryRequired
	}
	if publisher == nil {
		return nil, errPublisherRequired
	}
	if options.BatchSize <= 0 {
		return nil, errBatchSizeNotPositive
	}
	if options.LeaseDuration <= 0 {
		return nil, errLeaseDurationNotPositive
	}
	if options.RetryDelay <= 0 {
		return nil, errRetryDelayNotPositive
	}
	if options.PollInterval <= 0 {
		options.PollInterval = time.Second
	}
	if options.MaxRetryDelay <= 0 {
		options.MaxRetryDelay = defaultMaxRetryDelay
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = defaultMaxAttempts
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Jitter == nil {
		options.Jitter = equalJitter
	}
	if options.Observer == nil {
		options.Observer = noopObserver{}
	}

	return &Relay{repository: repository, publisher: publisher, options: options}, nil
}

// Run 持续发布事件，直到 ctx 被取消。
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.options.PollInterval)
	defer ticker.Stop()

	for {
		// 单轮故障只记录而不杀死 Relay：失败记录已进入退避或保留 processing 租约，
		// 后续轮次可以自行恢复，避免瞬时 Kafka/数据库故障必须依赖进程重启。
		if err := r.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "Outbox relay iteration failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// RunOnce 领取并尝试发布一个批次的事件。
func (r *Relay) RunOnce(ctx context.Context) error {
	events, err := r.repository.Claim(ctx, r.options.BatchSize, r.options.LeaseDuration)
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}

	var errs []error
	for _, event := range events {
		if err := r.publish(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

//nolint:gocognit,nestif // 发布、终态转换和租约回写必须保持在同一控制流中处理。
func (r *Relay) publish(ctx context.Context, event Event) error {
	startedAt := r.options.Now()
	// Kafka 发布发生在租约事务之外，避免网络等待长期占用数据库锁。发布失败时释放为
	// failed 并退避；若状态更新失败则保留 processing，待租约过期后重新领取。
	if err := r.publisher.Publish(ctx, event); err != nil {
		r.options.Observer.RecordPublish(ctx, event.EventType, "failure", r.options.Now().Sub(startedAt))
		if event.Attempts >= r.options.MaxAttempts {
			if exhausted, ok := r.repository.(ExhaustionRepository); ok {
				if markErr := exhausted.MarkExhausted(ctx, event.EventID, event.Attempts, err.Error(), r.options.Now()); markErr != nil {
					return errors.Join(
						fmt.Errorf("publish outbox event %s exhausted after %d attempts: %w", event.EventID, event.Attempts, err),
						fmt.Errorf("mark outbox event %s exhausted: %w", event.EventID, markErr),
					)
				}
				r.options.Observer.RecordPublish(ctx, event.EventType, "exhausted", r.options.Now().Sub(startedAt))
				return fmt.Errorf("publish outbox event %s exhausted after %d attempts: %w", event.EventID, event.Attempts, err)
			}
			slog.ErrorContext(ctx, "Outbox event reached retry limit; repository has no terminal state", "event_type", event.EventType, "attempts", event.Attempts)
		}
		delay := r.options.Jitter(r.retryDelay(event.Attempts - 1))
		nextAvailableAt := r.options.Now().Add(delay)
		if markErr := r.repository.MarkFailed(ctx, event.EventID, event.Attempts, err.Error(), nextAvailableAt); markErr != nil {
			return errors.Join(
				fmt.Errorf("publish outbox event %s: %w", event.EventID, err),
				fmt.Errorf("mark outbox event %s failed: %w", event.EventID, markErr),
			)
		}
		return fmt.Errorf("publish outbox event %s: %w", event.EventID, err)
	}

	// Kafka 成功但数据库标记失败时不能假装完成。租约过期后的再次发布可能产生重复，
	// 这是 at-least-once 的预期代价；消费者 Inbox 负责吸收重复，优先保证不丢消息。
	if err := r.repository.MarkPublished(ctx, event.EventID, event.Attempts, r.options.Now()); err != nil {
		r.options.Observer.RecordPublish(ctx, event.EventType, "failure", r.options.Now().Sub(startedAt))
		return fmt.Errorf("mark outbox event %s published: %w", event.EventID, err)
	}
	r.options.Observer.RecordPublish(ctx, event.EventType, "success", r.options.Now().Sub(startedAt))
	return nil
}

type noopObserver struct{}

func (noopObserver) RecordPublish(context.Context, string, string, time.Duration) {}

func equalJitter(delay time.Duration) time.Duration {
	if delay <= 1 {
		return delay
	}
	// equal jitter 保留一半退避下限并随机化另一半，既避免过早重试，也防止多个 Relay
	// 在 Kafka 恢复时按相同指数节奏同步唤醒形成流量尖峰。
	half := delay / 2
	return half + time.Duration(rand.Int64N(int64(delay-half)))
}

func (r *Relay) retryDelay(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	delay := r.options.RetryDelay
	for range attempts {
		if delay >= r.options.MaxRetryDelay/2 {
			return r.options.MaxRetryDelay
		}
		delay *= 2
	}
	if delay > r.options.MaxRetryDelay {
		return r.options.MaxRetryDelay
	}
	return delay
}

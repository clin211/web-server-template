package consumer

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
)

var (
	// ErrEventIdentityConflict 表示 Kafka header、envelope 或已有 Inbox 身份不一致。
	ErrEventIdentityConflict = errors.New("event identity conflict")
	// ErrInboxLeaseLost 表示 Inbox 处理租约已被其他 worker 接管。
	ErrInboxLeaseLost = errors.New("inbox event lease lost")
	// ErrInboxBusy 表示另一个 worker 仍持有 Inbox 处理租约。
	ErrInboxBusy = errors.New("inbox event is being processed")

	errTransactionerRequired   = errors.New("inbox transactioner is required")
	errInboxRepositoryRequired = errors.New("inbox repository is required")
	errEventHandlerRequired    = errors.New("event handler is required")
	errConsumerNameRequired    = errors.New("consumer name is required")
	errInboxLeaseNotPositive   = errors.New("inbox lease duration must be positive")
	errInboxRetryNotPositive   = errors.New("inbox retry delay must be positive")
	errInboxUnknownDisposition = errors.New("unknown inbox claim disposition")
)

// ClaimDisposition 描述 Inbox 事件的领取结果。
type ClaimDisposition string

const (
	// ClaimAcquired 表示当前 worker 获得处理租约。
	ClaimAcquired ClaimDisposition = "acquired"
	// ClaimCompleted 表示事件已经成功处理。
	ClaimCompleted ClaimDisposition = "completed"
	// ClaimDeadLettered 表示事件已经可靠进入 DLQ。
	ClaimDeadLettered ClaimDisposition = "dead_lettered"
	// ClaimBusy 表示其他 worker 仍持有有效租约。
	ClaimBusy ClaimDisposition = "busy"
)

// MessageIdentity 是 Inbox 幂等和冲突检测所需的稳定消息身份。
type MessageIdentity struct {
	ConsumerName    string
	EventID         string
	EventType       string
	SourceTopic     string
	SourcePartition int
	SourceOffset    int64
	PayloadHash     []byte
}

// Claim 包含 Inbox 领取状态及作为 fencing token 的 attempt。
type Claim struct {
	Disposition ClaimDisposition
	Attempt     int
}

// InboxRepository 定义 Consumer 对 Inbox 状态机的持久化依赖。
type InboxRepository interface {
	Claim(ctx context.Context, identity MessageIdentity, leaseDuration time.Duration) (Claim, error)
	MarkCompleted(ctx context.Context, consumerName, eventID string, attempt int, processedAt time.Time) error
	MarkFailed(ctx context.Context, consumerName, eventID string, attempt int, reason string, nextAvailableAt time.Time) error
}

// Transactioner 将业务副作用与 Inbox 完成状态放入同一事务。
type Transactioner interface {
	TX(ctx context.Context, fn func(context.Context) error) error
}

// Handler 处理已验证的事件 envelope。
type Handler interface {
	Handle(ctx context.Context, envelope outbox.Envelope) error
}

// HandlerFunc 将函数适配为 Handler。
type HandlerFunc func(context.Context, outbox.Envelope) error

// Handle 调用事件处理函数。
func (f HandlerFunc) Handle(ctx context.Context, envelope outbox.Envelope) error {
	return f(ctx, envelope)
}

// InboxOptions 定义逻辑消费者名称、租约和失败退避。
type InboxOptions struct {
	ConsumerName  string
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	Now           func() time.Time
}

// InboxProcessor 使用数据库 Inbox 屏障幂等处理事件。
type InboxProcessor struct {
	transactioner Transactioner
	repository    InboxRepository
	handler       Handler
	options       InboxOptions
}

// NewInboxProcessor 校验依赖并创建 InboxProcessor。
//
//nolint:gocognit,gocyclo
func NewInboxProcessor(transactioner Transactioner, repository InboxRepository, handler Handler, options InboxOptions) (*InboxProcessor, error) {
	if transactioner == nil {
		return nil, errTransactionerRequired
	}
	if repository == nil {
		return nil, errInboxRepositoryRequired
	}
	if handler == nil {
		return nil, errEventHandlerRequired
	}
	if options.ConsumerName == "" {
		return nil, errConsumerNameRequired
	}
	if options.LeaseDuration <= 0 {
		return nil, errInboxLeaseNotPositive
	}
	if options.RetryDelay <= 0 {
		return nil, errInboxRetryNotPositive
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &InboxProcessor{transactioner: transactioner, repository: repository, handler: handler, options: options}, nil
}

// Process 校验事件身份、领取 Inbox 租约并原子完成业务处理。
//
//nolint:gocognit,gocyclo
func (p *InboxProcessor) Process(ctx context.Context, message kafka.Message) error {
	envelope, err := outbox.ParseEnvelope(message.Value)
	if err != nil {
		return Permanent(fmt.Errorf("parse event envelope: %w", err))
	}
	// Header 用于 broker 侧路由和观测，body 是业务契约；两者不一致意味着消息身份
	// 已被错误构造或篡改，不能按普通业务失败重试。
	if headerValue(message.Headers, "event_id") != envelope.EventID || headerValue(message.Headers, "event_type") != envelope.EventType {
		return Permanent(fmt.Errorf("%w for event %s", ErrEventIdentityConflict, envelope.EventID))
	}

	payloadHash := sha256.Sum256(message.Value)
	claim, err := p.repository.Claim(ctx, MessageIdentity{
		ConsumerName:    p.options.ConsumerName,
		EventID:         envelope.EventID,
		EventType:       envelope.EventType,
		SourceTopic:     message.Topic,
		SourcePartition: message.Partition,
		SourceOffset:    message.Offset,
		PayloadHash:     payloadHash[:],
	}, p.options.LeaseDuration)
	if err != nil {
		if errors.Is(err, ErrEventIdentityConflict) {
			return Permanent(err)
		}
		return fmt.Errorf("claim inbox event %s: %w", envelope.EventID, err)
	}
	// 终态事件说明业务副作用或 DLQ 已经可靠落库。直接成功返回仅用于补交此前失败的
	// Kafka offset，不能再次调用业务 handler。
	if claim.Disposition == ClaimCompleted || claim.Disposition == ClaimDeadLettered {
		return nil
	}
	if claim.Disposition == ClaimBusy {
		return fmt.Errorf("event %s: %w", envelope.EventID, ErrInboxBusy)
	}
	if claim.Disposition != ClaimAcquired {
		return fmt.Errorf("%w: event %s disposition %q", errInboxUnknownDisposition, envelope.EventID, claim.Disposition)
	}

	eventCtx := ctx
	if envelope.TraceID != "" {
		eventCtx = contextx.WithTraceID(eventCtx, envelope.TraceID)
	}
	// 业务副作用与 MarkCompleted 必须共享同一数据库事务：任何一步失败都整体回滚，
	// 从而避免“业务已生效但 Inbox 仍可重试”造成重复副作用。
	processErr := p.transactioner.TX(eventCtx, func(txCtx context.Context) error {
		if err := p.handler.Handle(txCtx, envelope); err != nil {
			return fmt.Errorf("handle event %s: %w", envelope.EventID, err)
		}
		if err := p.repository.MarkCompleted(txCtx, p.options.ConsumerName, envelope.EventID, claim.Attempt, p.options.Now().UTC()); err != nil {
			return fmt.Errorf("complete inbox event %s: %w", envelope.EventID, err)
		}
		return nil
	})
	if processErr == nil {
		return nil
	}

	// 业务事务此时已经回滚，失败状态在独立事务上下文中持久化，以保存跨进程重试次数。
	// MarkFailed 仍校验 attempt；若租约已被接管，旧 worker 不能覆盖新 worker 的状态。
	nextAvailableAt := p.options.Now().UTC().Add(p.options.RetryDelay)
	if markErr := p.repository.MarkFailed(eventCtx, p.options.ConsumerName, envelope.EventID, claim.Attempt, processErr.Error(), nextAvailableAt); markErr != nil {
		return attemptedError{attempt: claim.Attempt, err: errors.Join(processErr, fmt.Errorf("mark inbox event %s failed: %w", envelope.EventID, markErr))}
	}
	return attemptedError{attempt: claim.Attempt, err: processErr}
}

type permanentError struct {
	err error
}

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent 标记不应重试的消息协议或业务错误。
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent 报告错误是否属于不可重试错误。
func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

type attemptedError struct {
	attempt int
	err     error
}

func (e attemptedError) Error() string { return e.err.Error() }
func (e attemptedError) Unwrap() error { return e.err }

// FailureAttempt 返回持久化 Inbox 错误携带的 attempt fencing token。
func FailureAttempt(err error) int {
	var target attemptedError
	if errors.As(err, &target) {
		return target.attempt
	}
	return 0
}

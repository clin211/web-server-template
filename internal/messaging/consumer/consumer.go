// Package consumer 提供手动提交 offset 的 Kafka 事件消费能力。
package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	errReaderRequired            = errors.New("consumer reader is required")
	errProcessorRequired         = errors.New("consumer processor is required")
	errDLQPublisherRequired      = errors.New("consumer DLQ publisher is required")
	errMaxAttemptsNotPositive    = errors.New("consumer max attempts must be positive")
	errProcessTimeoutNotPositive = errors.New("consumer process timeout must be positive")
)

// MessageReader 定义 Consumer 所需的 Kafka 拉取和手动提交能力。
type MessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, messages ...kafka.Message) error
}

// Processor 在提交 Kafka offset 前处理一条消息。
type Processor interface {
	Process(ctx context.Context, message kafka.Message) error
}

// DeadLetterPublisher 可靠发布无法处理的消息。
type DeadLetterPublisher interface {
	Publish(ctx context.Context, record DeadLetter) error
}

// Observer 记录 Consumer 处理、DLQ 和 lag 状态。
type Observer interface {
	RecordProcess(ctx context.Context, eventType, result string, duration time.Duration)
	RecordDLQ(ctx context.Context, eventType, result string, failedAt time.Time)
	RecordLag(ctx context.Context, topic string, partition int, lag int64)
}

// Options 定义消息重试和在途处理的时间边界。
type Options struct {
	MaxAttempts    int
	RetryDelay     time.Duration
	MaxRetryDelay  time.Duration
	ProcessTimeout time.Duration
	Observer       Observer
	Now            func() time.Time
	Wait           func(context.Context, time.Duration) error
}

// Consumer 串行处理消息，并仅在处理或 DLQ 成功后提交 offset。
type Consumer struct {
	reader    MessageReader
	processor Processor
	dlq       DeadLetterPublisher
	options   Options
}

// New 校验依赖和选项并创建 Consumer。
//
//nolint:gocognit,gocyclo
func New(reader MessageReader, processor Processor, dlq DeadLetterPublisher, options Options) (*Consumer, error) {
	if reader == nil {
		return nil, errReaderRequired
	}
	if processor == nil {
		return nil, errProcessorRequired
	}
	if dlq == nil {
		return nil, errDLQPublisherRequired
	}
	if options.MaxAttempts <= 0 {
		return nil, errMaxAttemptsNotPositive
	}
	if options.ProcessTimeout <= 0 {
		return nil, errProcessTimeoutNotPositive
	}
	if options.RetryDelay <= 0 {
		options.RetryDelay = time.Second
	}
	if options.MaxRetryDelay < options.RetryDelay {
		options.MaxRetryDelay = options.RetryDelay
	}
	if options.Observer == nil {
		options.Observer = noopObserver{}
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Wait == nil {
		options.Wait = waitContext
	}
	return &Consumer{reader: reader, processor: processor, dlq: dlq, options: options}, nil
}

// Run 持续消费消息，直到 ctx 被取消或 Reader 返回不可恢复错误。
func (c *Consumer) Run(ctx context.Context) error {
	// 首版故意串行处理。消费者组提交某分区的较高 offset 会同时越过此前 offset；若直接
	// goroutine fan-out，同分区后完成的高 offset 可能掩盖仍失败的低 offset。未来扩展
	// 并行度时必须按 partition 建立独立保序队列。
	for ctx.Err() == nil {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				break
			}
			return fmt.Errorf("fetch Kafka message: %w", err)
		}

		if err := c.consume(ctx, message); err != nil {
			return err
		}
	}
	return context.Canceled
}

//nolint:gocognit,gocyclo
func (c *Consumer) consume(runCtx context.Context, message kafka.Message) error {
	// FetchMessage 返回后，这条消息已经成为当前 worker 的在途工作。关闭信号只停止继续
	// Fetch；这里保留原 context 中的 values 并设置独立超时，让业务处理、DLQ 落库或 offset 提交有机会完成。
	// 超时导致提交失败时 Kafka 会再次投递，Inbox 会屏蔽重复副作用，因此语义仍是 at-least-once。
	processCtx, cancel := context.WithTimeout(context.WithoutCancel(runCtx), c.options.ProcessTimeout)
	defer cancel()

	eventType := headerValue(message.Headers, "event_type")
	// HighWaterMark 只能估算本次 fetch 时刻的分区积压，不代表消费者组的权威 lag；
	// 集群级告警和容量判断应使用 Kafka exporter 或托管平台指标。
	lag := message.HighWaterMark - message.Offset - 1
	lag = max(lag, 0)
	c.options.Observer.RecordLag(processCtx, message.Topic, message.Partition, lag)

	// Inbox attempt 持久化在数据库中，进程重启后必须沿用该次数；否则 poison message
	// 会因本地计数归零而永远无法达到 DLQ 阈值。
	var processErr error
	attempts := 0
	for attempts < c.options.MaxAttempts {
		startedAt := c.options.Now()
		processErr = c.processor.Process(processCtx, message)
		result := "success"
		if processErr != nil {
			result = "failure"
		}
		c.options.Observer.RecordProcess(processCtx, eventType, result, c.options.Now().Sub(startedAt))
		if processErr == nil {
			return c.commitWithRetry(processCtx, message)
		}
		// permanent error 无需重试；普通失败使用 Inbox 持久化 attempt；Busy 表示其他
		// worker 持有租约且本次没有执行 handler，因此不能消耗失败次数。
		if IsPermanent(processErr) {
			attempts++
			break
		}
		if persistedAttempt := FailureAttempt(processErr); persistedAttempt > 0 {
			attempts = persistedAttempt
		} else if !errors.Is(processErr, ErrInboxBusy) {
			attempts++
		}
		if attempts >= c.options.MaxAttempts {
			break
		}
		if err := c.options.Wait(processCtx, c.retryDelay(max(attempts-1, 0))); err != nil {
			return fmt.Errorf("wait to retry Kafka message: %w", err)
		}
	}

	record := deadLetterFromMessage(message, max(attempts, 1), processErr, c.options.Now())
	// 只有 DLQ 已可靠持久化后才能提交源 offset。反过来的顺序会在进程崩溃时永久丢失
	// poison message；提交失败则只会造成可由 Inbox/DLQ 唯一约束吸收的重复投递。
	if err := c.publishDLQWithRetry(processCtx, record); err != nil {
		return err
	}
	return c.commitWithRetry(processCtx, message)
}

func (c *Consumer) publishDLQWithRetry(ctx context.Context, record DeadLetter) error {
	// DLQ 是源消息的最终证据，不能因为短暂数据库故障而放弃；循环只受在途处理
	// context 的超时约束，超时后不提交源 offset，交由 Kafka 重新投递。
	for attempt := 0; ; attempt++ {
		err := c.dlq.Publish(ctx, record)
		result := "success"
		if err != nil {
			result = "failure"
		}
		c.options.Observer.RecordDLQ(ctx, record.EventType, result, record.FailedAt)
		if err == nil {
			return nil
		}
		slog.ErrorContext(ctx, "Failed to publish dead-letter message",
			"event.name", "consumer.dlq.publish_failed",
			"messaging.message.id", record.EventID,
			"messaging.destination", record.Topic,
			"error", err,
		)
		if waitErr := c.options.Wait(ctx, c.retryDelay(attempt)); waitErr != nil {
			return errors.Join(fmt.Errorf("publish dead-letter message: %w", err), fmt.Errorf("wait to retry dead-letter message: %w", waitErr))
		}
	}
}

func (c *Consumer) commitWithRetry(ctx context.Context, message kafka.Message) error {
	// CommitMessages 使用同步提交。提交成功是本条消息处理完成的唯一边界，因此失败时
	// 不继续抓取同分区的后续消息，避免较高 offset 越过尚未确认的消息。
	for attempt := 0; ; attempt++ {
		if err := c.reader.CommitMessages(ctx, message); err == nil {
			return nil
		} else {
			slog.ErrorContext(ctx, "Failed to commit Kafka message",
				"event.name", "consumer.commit_failed",
				"messaging.destination", message.Topic,
				"messaging.kafka.partition", message.Partition,
				"messaging.kafka.offset", message.Offset,
				"error", err,
			)
			if waitErr := c.options.Wait(ctx, c.retryDelay(attempt)); waitErr != nil {
				return errors.Join(fmt.Errorf("commit Kafka message: %w", err), fmt.Errorf("wait to retry Kafka commit: %w", waitErr))
			}
		}
	}
}

func (c *Consumer) retryDelay(attempt int) time.Duration {
	delay := c.options.RetryDelay
	for range attempt {
		if delay >= c.options.MaxRetryDelay/2 {
			return c.options.MaxRetryDelay
		}
		delay *= 2
	}
	if delay > c.options.MaxRetryDelay {
		return c.options.MaxRetryDelay
	}
	return delay
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type noopObserver struct{}

func (noopObserver) RecordProcess(context.Context, string, string, time.Duration) {}
func (noopObserver) RecordDLQ(context.Context, string, string, time.Time)         {}
func (noopObserver) RecordLag(context.Context, string, int, int64)                {}

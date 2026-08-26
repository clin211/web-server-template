package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*ConsumerOptions)(nil)

var (
	errConsumerMaxAttemptsNotPositive    = errors.New("consumer max attempts must be positive")
	errConsumerRetryDelayNotPositive     = errors.New("consumer retry delay must be positive")
	errConsumerMaxRetryDelayTooShort     = errors.New("consumer max retry delay must not be less than retry delay")
	errConsumerProcessTimeoutNotPositive = errors.New("consumer process timeout must be positive")
	errConsumerInboxLeaseNotPositive     = errors.New("consumer inbox lease duration must be positive")
)

// ConsumerOptions 定义 Kafka Consumer、Inbox 和 DLQ 的运行参数。
type ConsumerOptions struct {
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	// MaxAttempts 基于 Inbox 中持久化的 attempt，而不是进程内计数，因此重启不会重置重试预算。
	MaxAttempts   int           `json:"max-attempts" mapstructure:"max-attempts"`
	RetryDelay    time.Duration `json:"retry-delay" mapstructure:"retry-delay"`
	MaxRetryDelay time.Duration `json:"max-retry-delay" mapstructure:"max-retry-delay"`
	// ProcessTimeout 是停机后排空已 Fetch 消息的总预算，应小于 InboxLeaseDuration。
	ProcessTimeout time.Duration `json:"process-timeout" mapstructure:"process-timeout"`
	// InboxLeaseDuration 必须覆盖最坏业务事务耗时和依赖抖动，否则旧 worker 尚未结束时
	// 另一实例就可能重新领取同一事件；attempt fencing 只能阻止迟到写，不能撤销已发生的外部副作用。
	InboxLeaseDuration time.Duration `json:"inbox-lease-duration" mapstructure:"inbox-lease-duration"`
	// DLQTopic 必须与源 topic 分离，避免死信重新进入原 Consumer 形成消费循环。
	DLQTopic string `json:"dlq-topic" mapstructure:"dlq-topic"`
}

// NewConsumerOptions 创建具有保守默认值的 ConsumerOptions。
func NewConsumerOptions() *ConsumerOptions {
	return &ConsumerOptions{
		Enabled:            false,
		MaxAttempts:        5,
		RetryDelay:         time.Second,
		MaxRetryDelay:      time.Minute,
		ProcessTimeout:     30 * time.Second,
		InboxLeaseDuration: time.Minute,
	}
}

// Validate 验证 Consumer 参数。
func (o *ConsumerOptions) Validate() []error {
	var errs []error
	if o.MaxAttempts <= 0 {
		errs = append(errs, errConsumerMaxAttemptsNotPositive)
	}
	if o.RetryDelay <= 0 {
		errs = append(errs, errConsumerRetryDelayNotPositive)
	}
	if o.MaxRetryDelay < o.RetryDelay {
		errs = append(errs, errConsumerMaxRetryDelayTooShort)
	}
	if o.ProcessTimeout <= 0 {
		errs = append(errs, errConsumerProcessTimeoutNotPositive)
	}
	if o.InboxLeaseDuration <= 0 {
		errs = append(errs, errConsumerInboxLeaseNotPositive)
	}
	return errs
}

// AddFlags 将 Consumer 参数注册到 FlagSet。
func (o *ConsumerOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.Enabled, fullPrefix+".enabled", o.Enabled, "Enable the Kafka event consumer.")
	fs.IntVar(&o.MaxAttempts, fullPrefix+".max-attempts", o.MaxAttempts, "Maximum processing attempts before moving a message to the DLQ.")
	fs.DurationVar(&o.RetryDelay, fullPrefix+".retry-delay", o.RetryDelay, "Base delay after a message processing failure.")
	fs.DurationVar(&o.MaxRetryDelay, fullPrefix+".max-retry-delay", o.MaxRetryDelay, "Maximum delay between message processing attempts.")
	fs.DurationVar(&o.ProcessTimeout, fullPrefix+".process-timeout", o.ProcessTimeout, "Maximum time to drain an already fetched message.")
	fs.DurationVar(&o.InboxLeaseDuration, fullPrefix+".inbox-lease-duration", o.InboxLeaseDuration, "Duration of an Inbox processing lease.")
	fs.StringVar(&o.DLQTopic, fullPrefix+".dlq-topic", o.DLQTopic, "Kafka topic used for dead-letter messages.")
}

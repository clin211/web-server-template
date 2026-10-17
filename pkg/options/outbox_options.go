package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*OutboxOptions)(nil)

var (
	errOutboxBatchSizeNotPositive     = errors.New("outbox batch size must be positive")
	errOutboxPollIntervalNotPositive  = errors.New("outbox poll interval must be positive")
	errOutboxLeaseDurationNotPositive = errors.New("outbox lease duration must be positive")
	errOutboxRetryDelayNotPositive    = errors.New("outbox retry delay must be positive")
	errOutboxMaxRetryDelayTooShort    = errors.New("outbox max retry delay must not be less than retry delay")
	errOutboxMaxAttemptsNotPositive   = errors.New("outbox max attempts must be positive")
)

// OutboxOptions 定义事务性 Outbox Relay 的运行参数。
type OutboxOptions struct {
	Enabled      bool          `json:"enabled" mapstructure:"enabled"`
	BatchSize    int           `json:"batch-size" mapstructure:"batch-size"`
	PollInterval time.Duration `json:"poll-interval" mapstructure:"poll-interval"`
	// Relay 会先领取整个批次再串行发布，LeaseDuration 必须覆盖最坏批次发布时间。
	// 租约过短会让其他实例重复领取；系统仍不丢消息，但 at-least-once 的重复率会升高。
	LeaseDuration time.Duration `json:"lease-duration" mapstructure:"lease-duration"`
	// RetryDelay 和 MaxRetryDelay 构成带抖动的有界指数退避，防止多实例在 Kafka
	// 恢复时同时重试并形成流量尖峰。
	RetryDelay    time.Duration `json:"retry-delay" mapstructure:"retry-delay"`
	MaxRetryDelay time.Duration `json:"max-retry-delay" mapstructure:"max-retry-delay"`
	MaxAttempts   int           `json:"max-attempts" mapstructure:"max-attempts"`
}

// NewOutboxOptions 创建具有保守默认值的 OutboxOptions。
func NewOutboxOptions() *OutboxOptions {
	return &OutboxOptions{
		Enabled:       false,
		BatchSize:     100,
		PollInterval:  time.Second,
		LeaseDuration: 30 * time.Second,
		RetryDelay:    time.Second,
		MaxRetryDelay: 5 * time.Minute,
		MaxAttempts:   20,
	}
}

// Validate 验证 Outbox Relay 参数。
func (o *OutboxOptions) Validate() []error {
	var errs []error
	if o.BatchSize <= 0 {
		errs = append(errs, errOutboxBatchSizeNotPositive)
	}
	if o.PollInterval <= 0 {
		errs = append(errs, errOutboxPollIntervalNotPositive)
	}
	if o.LeaseDuration <= 0 {
		errs = append(errs, errOutboxLeaseDurationNotPositive)
	}
	if o.RetryDelay <= 0 {
		errs = append(errs, errOutboxRetryDelayNotPositive)
	}
	if o.MaxRetryDelay < o.RetryDelay {
		errs = append(errs, errOutboxMaxRetryDelayTooShort)
	}
	if o.MaxAttempts <= 0 {
		errs = append(errs, errOutboxMaxAttemptsNotPositive)
	}
	return errs
}

// AddFlags 将 Outbox Relay 参数注册到 FlagSet。
func (o *OutboxOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.Enabled, fullPrefix+".enabled", o.Enabled, "Enable the transactional outbox relay.")
	fs.IntVar(&o.BatchSize, fullPrefix+".batch-size", o.BatchSize, "Maximum events claimed per relay iteration.")
	fs.DurationVar(&o.PollInterval, fullPrefix+".poll-interval", o.PollInterval, "Delay between relay iterations.")
	fs.DurationVar(&o.LeaseDuration, fullPrefix+".lease-duration", o.LeaseDuration, "Duration of an outbox processing lease.")
	o.addRetryFlags(fs, fullPrefix)
	fs.IntVar(&o.MaxAttempts, fullPrefix+".max-attempts", o.MaxAttempts, "Maximum publish attempts before an event enters the terminal dead state.")
}

func (o *OutboxOptions) addRetryFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.DurationVar(&o.RetryDelay, fullPrefix+".retry-delay", o.RetryDelay, "Base delay after a publish failure.")
	fs.DurationVar(&o.MaxRetryDelay, fullPrefix+".max-retry-delay", o.MaxRetryDelay, "Maximum delay after repeated publish failures.")
}

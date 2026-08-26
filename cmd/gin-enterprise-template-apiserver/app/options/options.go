// nolint: err113
package options

import (
	"fmt"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/clin211/gin-enterprise-template/internal/apiserver"
)

// ServerOptions 包含服务器的配置选项。
type ServerOptions struct {
	// JWTOptions 包含 JWT 认证配置选项。
	JWTOptions *genericoptions.JWTOptions `json:"jwt" mapstructure:"jwt"`
	// TLSOptions 包含 TLS 配置选项。
	TLSOptions *genericoptions.TLSOptions `json:"tls" mapstructure:"tls"`
	// HTTPOptions 包含 HTTP 配置选项。
	HTTPOptions *genericoptions.HTTPOptions `json:"http" mapstructure:"http"`
	// PostgreSQLOptions 包含 PostgreSQL 配置选项。
	PostgreSQLOptions *genericoptions.PostgreSQLOptions `json:"postgresql" mapstructure:"postgresql"`
	// RedisOptions 包含 Redis 配置选项。
	RedisOptions *genericoptions.RedisOptions `json:"redis" mapstructure:"redis"`
	// OTelOptions 用于指定 OpenTelemetry 选项。
	OTelOptions *genericoptions.OTelOptions `json:"otel" mapstructure:"otel"`
	// KafkaOptions 包含 Outbox Relay 使用的 Kafka 配置。
	KafkaOptions *genericoptions.KafkaOptions `json:"kafka" mapstructure:"kafka"`
	// OutboxOptions 包含事务性 Outbox Relay 配置。
	OutboxOptions *genericoptions.OutboxOptions `json:"outbox" mapstructure:"outbox"`
	// ConsumerOptions 包含 Kafka Consumer、Inbox 和 DLQ 配置。
	ConsumerOptions *genericoptions.ConsumerOptions `json:"consumer" mapstructure:"consumer"`
	// LoginSecurityOptions 包含 Redis 支撑的登录暴力破解防护配置。
	LoginSecurityOptions *genericoptions.LoginSecurityOptions `json:"login-security" mapstructure:"login-security"`
}

// NewServerOptions 创建一个使用默认值的 ServerOptions 实例。
func NewServerOptions() *ServerOptions {
	opts := &ServerOptions{
		JWTOptions:           genericoptions.NewJWTOptions(),
		TLSOptions:           genericoptions.NewTLSOptions(),
		HTTPOptions:          genericoptions.NewHTTPOptions(),
		PostgreSQLOptions:    genericoptions.NewPostgreSQLOptions(),
		RedisOptions:         genericoptions.NewRedisOptions(),
		OTelOptions:          genericoptions.NewOTelOptions(),
		KafkaOptions:         genericoptions.NewKafkaOptions(),
		OutboxOptions:        genericoptions.NewOutboxOptions(),
		ConsumerOptions:      genericoptions.NewConsumerOptions(),
		LoginSecurityOptions: genericoptions.NewLoginSecurityOptions(),
	}
	opts.HTTPOptions.Addr = ":5555"

	return opts
}

// AddFlags 将 ServerOptions 中的选项绑定到命令行标志。
func (o *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	// 添加 JWT 选项标志
	o.JWTOptions.AddFlags(fs, "jwt")
	// 为子选项添加命令行标志。
	o.TLSOptions.AddFlags(fs, "tls")
	o.HTTPOptions.AddFlags(fs, "http")
	o.PostgreSQLOptions.AddFlags(fs, "postgresql")
	o.RedisOptions.AddFlags(fs, "redis")
	o.OTelOptions.AddFlags(fs, "otel")
	o.KafkaOptions.AddFlags(fs, "kafka")
	o.OutboxOptions.AddFlags(fs, "outbox")
	o.ConsumerOptions.AddFlags(fs, "consumer")
	o.LoginSecurityOptions.AddFlags(fs, "login-security")
}

// Complete 完成所有必需的选项。
func (o *ServerOptions) Complete() error {
	// TODO: 如果需要，添加完成逻辑。
	return nil
}

// Validate 检查 ServerOptions 中的选项是否有效。
func (o *ServerOptions) Validate() error {
	errs := []error{}

	// 验证 JWT 选项
	errs = append(errs, o.JWTOptions.Validate()...)
	// 验证子选项。
	errs = append(errs, o.TLSOptions.Validate()...)
	errs = append(errs, o.HTTPOptions.Validate()...)
	errs = append(errs, o.PostgreSQLOptions.Validate()...)
	errs = append(errs, o.RedisOptions.Validate()...)
	errs = append(errs, o.OTelOptions.Validate()...)
	errs = append(errs, o.OutboxOptions.Validate()...)
	errs = append(errs, o.ConsumerOptions.Validate()...)
	errs = append(errs, o.LoginSecurityOptions.Validate()...)
	// Refresh token rotation always requires Redis, regardless of login limiter state.
	if o.RedisOptions == nil || o.RedisOptions.Addr == "" {
		errs = append(errs, fmt.Errorf("redis is required for refresh token rotation"))
	}
	// Kafka 仅在异步链路启用时是强依赖；Outbox 和 Consumer 共用主 topic、连接及
	// Writer 配置，Consumer 还额外依赖稳定 group、同步提交和独立 DLQ topic。
	if o.OutboxOptions.Enabled {
		errs = append(errs, o.KafkaOptions.Validate()...)
		if o.KafkaOptions.Topic == "" {
			errs = append(errs, fmt.Errorf("kafka topic is required when outbox relay is enabled"))
		}
	}
	//nolint:nestif
	if o.ConsumerOptions.Enabled {
		if o.KafkaOptions == nil {
			errs = append(errs, fmt.Errorf("kafka options are required when consumer is enabled"))
		} else {
			errs = append(errs, o.KafkaOptions.Validate()...)
			if o.KafkaOptions.ReaderOptions.Topic == "" {
				errs = append(errs, fmt.Errorf("kafka reader topic is required when consumer is enabled"))
			}
			if o.KafkaOptions.ReaderOptions.GroupID == "" {
				errs = append(errs, fmt.Errorf("kafka reader group-id is required when consumer is enabled"))
			}
			// 非零 CommitInterval 会让 kafka-go 异步批量提交；这不是普通调优项，可能在
			// Inbox 完成或 DLQ 持久化前确认 offset，从而造成不可恢复的消息丢失。
			if o.KafkaOptions.ReaderOptions.CommitInterval != 0 {
				errs = append(errs, fmt.Errorf("kafka reader commit-interval must be zero when consumer is enabled"))
			}
			if o.ConsumerOptions.DLQTopic == "" {
				errs = append(errs, fmt.Errorf("consumer dlq topic is required when consumer is enabled"))
			}
		}
	}

	// 汇总所有错误并返回。
	return utilerrors.NewAggregate(errs)
}

// Config 基于 ServerOptions 构建 apiserver.Config。
func (o *ServerOptions) Config() (*apiserver.Config, error) {
	return &apiserver.Config{
		JWTOptions:           o.JWTOptions,
		TLSOptions:           o.TLSOptions,
		HTTPOptions:          o.HTTPOptions,
		PostgreSQLOptions:    o.PostgreSQLOptions,
		RedisOptions:         o.RedisOptions,
		OTelOptions:          o.OTelOptions,
		KafkaOptions:         o.KafkaOptions,
		OutboxOptions:        o.OutboxOptions,
		ConsumerOptions:      o.ConsumerOptions,
		LoginSecurityOptions: o.LoginSecurityOptions,
	}, nil
}

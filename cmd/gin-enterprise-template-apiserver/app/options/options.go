// nolint: err113
package options

import (
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
}

// NewServerOptions 创建一个使用默认值的 ServerOptions 实例。
func NewServerOptions() *ServerOptions {
	opts := &ServerOptions{
		JWTOptions:        genericoptions.NewJWTOptions(),
		TLSOptions:        genericoptions.NewTLSOptions(),
		HTTPOptions:       genericoptions.NewHTTPOptions(),
		PostgreSQLOptions: genericoptions.NewPostgreSQLOptions(),
		RedisOptions:      genericoptions.NewRedisOptions(),
		OTelOptions:       genericoptions.NewOTelOptions(),
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

	// 汇总所有错误并返回。
	return utilerrors.NewAggregate(errs)
}

// Config 基于 ServerOptions 构建 apiserver.Config。
func (o *ServerOptions) Config() (*apiserver.Config, error) {
	return &apiserver.Config{
		JWTOptions:        o.JWTOptions,
		TLSOptions:        o.TLSOptions,
		HTTPOptions:       o.HTTPOptions,
		PostgreSQLOptions: o.PostgreSQLOptions,
		RedisOptions:      o.RedisOptions,
		OTelOptions:       o.OTelOptions,
	}, nil
}

package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

var (
	errLoginRateLimitNotPositive  = errors.New("login security max attempts must be positive")
	errLoginRateWindowNotPositive = errors.New("login security window must be positive")
)

// LoginSecurityOptions 配置跨实例登录暴力破解防护。
type LoginSecurityOptions struct {
	Enabled     bool          `json:"enabled" mapstructure:"enabled"`
	MaxAttempts int64         `json:"max-attempts" mapstructure:"max-attempts"`
	Window      time.Duration `json:"window" mapstructure:"window"`
}

// NewLoginSecurityOptions 创建登录限流默认配置。
func NewLoginSecurityOptions() *LoginSecurityOptions {
	return &LoginSecurityOptions{Enabled: true, MaxAttempts: 10, Window: 5 * time.Minute}
}

// Validate 校验登录限流配置。
func (o *LoginSecurityOptions) Validate() []error {
	if o == nil || !o.Enabled {
		return nil
	}
	var errs []error
	if o.MaxAttempts <= 0 {
		errs = append(errs, errLoginRateLimitNotPositive)
	}
	if o.Window <= 0 {
		errs = append(errs, errLoginRateWindowNotPositive)
	}
	return errs
}

// AddFlags 注册登录安全参数。
func (o *LoginSecurityOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.Enabled, fullPrefix+".enabled", o.Enabled, "Enable Redis-backed login rate limiting.")
	fs.Int64Var(&o.MaxAttempts, fullPrefix+".max-attempts", o.MaxAttempts, "Maximum login attempts per direct client address in one window.")
	fs.DurationVar(&o.Window, fullPrefix+".window", o.Window, "Login rate-limit fixed window duration.")
}

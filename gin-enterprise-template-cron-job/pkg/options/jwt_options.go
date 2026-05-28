package options

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*JWTOptions)(nil)

// JWTOptions 包含与 JWT 认证相关的配置项。
type JWTOptions struct {
	// Secret 是用于签名 JWT 令牌的私钥。
	Secret string `json:"secret" mapstructure:"secret"`
	// AccessExpiration 是访问令牌的过期时间。
	AccessExpiration time.Duration `json:"access-expiration" mapstructure:"access-expiration"`
	// RefreshExpiration 是刷新令牌的过期时间。
	RefreshExpiration time.Duration `json:"refresh-expiration" mapstructure:"refresh-expiration"`

	fullPrefix string
}

// NewJWTOptions 创建一个 JWTOptions 实例。
// Secret 字段必须由配置文件 / 环境变量 / 命令行显式提供，
// 这里特意不再写死任何默认值，避免泄漏到生产环境。
func NewJWTOptions() *JWTOptions {
	return &JWTOptions{
		Secret:            "",
		AccessExpiration:  2 * time.Hour,
		RefreshExpiration: 168 * time.Hour, // 7 days
	}
}

// Validate 用于解析和验证 JWT 参数。
func (o *JWTOptions) Validate() []error {
	var errs []error

	if o.Secret == "" {
		errs = append(errs, fmt.Errorf(
			"--%s.secret must be specified (set via env APP_JWT_SECRET, --jwt.secret flag, or config file)",
			o.fullPrefix,
		))
	}
	if o.Secret != "" && IsPlaceholderSecret(o.Secret) {
		errs = append(errs, fmt.Errorf(
			"--%s.secret looks like a placeholder/known-weak value (%q); please generate a real one (e.g. `openssl rand -hex 32`)",
			o.fullPrefix, o.Secret,
		))
	}
	if o.Secret != "" && len(o.Secret) < 32 {
		errs = append(errs, fmt.Errorf(
			"--%s.secret must be at least 32 characters long (current: %d)",
			o.fullPrefix, len(o.Secret),
		))
	}
	if o.AccessExpiration <= 0 {
		errs = append(errs, fmt.Errorf("--%s.access-expiration must be positive", o.fullPrefix))
	}
	if o.RefreshExpiration <= 0 {
		errs = append(errs, fmt.Errorf("--%s.refresh-expiration must be positive", o.fullPrefix))
	}
	if o.RefreshExpiration < o.AccessExpiration {
		errs = append(errs, fmt.Errorf("--%s.refresh-expiration must be greater than or equal to access-expiration", o.fullPrefix))
	}

	return errs
}

// AddFlags 将与 JWT 配置相关的标志添加到指定的 FlagSet。
func (o *JWTOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	if fs == nil {
		return
	}

	o.fullPrefix = fullPrefix
	fs.StringVar(&o.Secret, fullPrefix+".secret", o.Secret, "Private key used to sign JWT tokens.")
	fs.DurationVar(&o.AccessExpiration, fullPrefix+".access-expiration", o.AccessExpiration, "JWT access token expiration time.")
	fs.DurationVar(&o.RefreshExpiration, fullPrefix+".refresh-expiration", o.RefreshExpiration, "JWT refresh token expiration time.")
}

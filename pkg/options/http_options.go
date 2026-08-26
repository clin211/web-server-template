package options

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*HTTPOptions)(nil)

var (
	errHTTPTimeoutNotPositive = errors.New("http timeout must be positive")
	errWildcardCORSOrigin     = errors.New("wildcard CORS origin is not allowed")
)

// HTTPOptions 包含与 HTTP 服务器启动相关的配置项。
type HTTPOptions struct {
	// 服务器的网络类型。
	Network string `json:"network" mapstructure:"network"`

	// 服务器地址。
	Addr string `json:"addr" mapstructure:"addr"`

	// 服务器读取请求、写入响应和保持空闲连接的超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// AllowedOrigins 是允许跨域访问 API 的精确 Origin 列表。空列表表示禁用 CORS。
	AllowedOrigins []string `json:"allowed-origins" mapstructure:"allowed-origins"`
}

// NewHTTPOptions 创建带有默认参数的 HTTPOptions 对象。
func NewHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:38443",
		Timeout: 30 * time.Second,
	}
}

// Validate 用于解析和验证用户在程序启动时在命令行输入的参数。
func (o *HTTPOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}

	if err := ValidateAddress(o.Addr); err != nil {
		errs = append(errs, err)
	}
	if o.Timeout <= 0 {
		errs = append(errs, errHTTPTimeoutNotPositive)
	}
	for _, origin := range o.AllowedOrigins {
		if origin == "*" {
			errs = append(errs, errWildcardCORSOrigin)
			continue
		}
		if err := validateCORSOrigin(origin); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// AddFlagsWithPrefix 将 HTTP 服务器相关标志注册到指定的 FlagSet，
// 使用 fullPrefix 作为标志名称的完整前缀。
//
// 示例：
//
//	o.AddFlagsWithPrefix(fs, "apiserver.http")  // --apiserver.http.network, --apiserver.http.addr, etc.
//	o.AddFlagsWithPrefix(fs, "gateway.http")    // --gateway.http.network, --gateway.http.addr, etc.
func (o *HTTPOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Network, fullPrefix+".network", o.Network,
		"Network type for the HTTP server (e.g., tcp, tcp4, tcp6).")
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr,
		"Listen address for the HTTP server (e.g., :8080, 0.0.0.0:8443).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout,
		"Timeout for incoming HTTP connections.")
	fs.StringSliceVar(&o.AllowedOrigins, fullPrefix+".allowed-origins", o.AllowedOrigins,
		"Exact origins allowed to make cross-origin requests. Empty disables CORS.")
}

// Complete 填充未设置且需要具有有效数据的字段。
func (s *HTTPOptions) Complete() error {
	return nil
}

func validateCORSOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("invalid CORS origin %q", origin)
	}
	return nil
}

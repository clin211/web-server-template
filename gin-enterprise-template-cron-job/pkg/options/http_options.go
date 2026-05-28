package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*HTTPOptions)(nil)

// HTTPOptions 包含与 HTTP 服务器启动相关的配置项。
type HTTPOptions struct {
	// 服务器的网络类型。
	Network string `json:"network" mapstructure:"network"`

	// 服务器地址。
	Addr string `json:"addr" mapstructure:"addr"`

	// 服务器超时时间。由 HTTP 客户端使用。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
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

	errors := []error{}

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
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
}

// Complete 填充未设置且需要具有有效数据的字段。
func (s *HTTPOptions) Complete() error {
	return nil
}

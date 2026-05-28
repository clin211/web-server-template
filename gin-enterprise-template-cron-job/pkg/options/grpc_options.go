package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*GRPCOptions)(nil)

// GRPCOptions 用于创建未认证、未授权、不安全的端口。
// 不应该再使用这些选项。
type GRPCOptions struct {
	// 服务器网络类型。
	Network string `json:"network" mapstructure:"network"`

	// 服务器地址。
	Addr string `json:"addr" mapstructure:"addr"`

	// 服务器超时时间。由 gRPC 客户端使用。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewGRPCOptions 用于创建未认证、未授权、不安全的端口。
// 不应该再使用这些选项。
func NewGRPCOptions() *GRPCOptions {
	return &GRPCOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:39090",
		Timeout: 30 * time.Second,
	}
}

// Validate 用于解析和验证用户在程序启动时在命令行输入的参数。
func (o *GRPCOptions) Validate() []error {
	var errors []error

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// AddFlags 将与特定 API 服务器的功能相关的标志添加到
// 指定的 FlagSet。
func (o *GRPCOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Network, fullPrefix+".network", o.Network, "Specify the network for the gRPC server.")
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, "Specify the gRPC server bind address and port.")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout for server connections.")
}

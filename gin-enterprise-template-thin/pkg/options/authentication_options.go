package options

import (
	"github.com/spf13/pflag"
)

var _ IOptions = (*ClientCertAuthenticationOptions)(nil)

// ClientCertAuthenticationOptions 提供客户端证书认证的不同选项。
type ClientCertAuthenticationOptions struct {
	// ClientCA 是您将识别的传入客户端证书的所有签名者的证书捆绑包
	ClientCA string `json:"client-ca-file" mapstructure:"client-ca-file"`
}

// NewClientCertAuthenticationOptions 创建带有默认参数的 ClientCertAuthenticationOptions 对象。
func NewClientCertAuthenticationOptions() *ClientCertAuthenticationOptions {
	return &ClientCertAuthenticationOptions{
		ClientCA: "",
	}
}

// Validate 用于解析和验证用户在程序启动时在命令行输入的参数。
func (o *ClientCertAuthenticationOptions) Validate() []error {
	return []error{}
}

// AddFlags 将与特定服务器的 ClientCertAuthenticationOptions 相关的标志添加到
// 指定的 FlagSet。
func (o *ClientCertAuthenticationOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.ClientCA, fullPrefix+".ca-file", o.ClientCA, ""+
		"If set, any request presenting a client certificate signed by one of "+
		"the authorities in the client-ca-file is authenticated with an identity "+
		"corresponding to the CommonName of the client certificate.")
}

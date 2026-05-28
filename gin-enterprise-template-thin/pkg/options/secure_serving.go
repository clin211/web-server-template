package options

import (
	"fmt"
	"path"

	"github.com/spf13/pflag"
)

var _ IOptions = (*SecureServingOptions)(nil)

// SecureServingOptions 包含与 HTTPS 服务器启动相关的配置项。
type SecureServingOptions struct {
	BindAddress string `json:"bind-address"`
	// BindPort 在设置 Listener 时被忽略，即使为 0 也会提供 HTTPS。
	BindPort int `json:"bind-port"`
	// Required 设置为 true 意味着 BindPort 不能为零。
	Required bool
	// ServerCert 是用于提供安全流量的 TLS 证书信息
	ServerCert GeneratableKeyCert `json:"tls"`
	// AdvertiseAddress net.IP

	fullPrefix string
}

// CertKey 包含与证书相关的配置项。
type CertKey struct {
	// CertFile 是包含 PEM 编码证书的文件，可能还包含完整的证书链
	CertFile string `json:"cert-file"`
	// KeyFile 是包含 CertFile 指定的证书的 PEM 编码私钥的文件
	KeyFile string `json:"private-key-file"`
}

// GeneratableKeyCert 包含与证书相关的配置项。
type GeneratableKeyCert struct {
	// CertKey 允许设置要使用的显式证书/密钥文件。
	CertKey CertKey `json:"cert-key"`

	// 如果未显式设置 CertFile/KeyFile，则指定写入生成证书的目录。
	// PairName 用于确定 CertDirectory 中的文件名。
	// 如果未设置 CertDirectory 和 PairName，将生成内存中的证书。
	CertDirectory string `json:"cert-dir"`
	// PairName 是将与 CertDirectory 一起使用的名称，用于制作证书和密钥文件名。
	// 它变成 CertDirectory/PairName.crt 和 CertDirectory/PairName.key
	PairName string `json:"pair-name"`
}

// NewSecureServingOptions 创建带有默认参数的 SecureServingOptions 对象。
func NewSecureServingOptions() *SecureServingOptions {
	return &SecureServingOptions{
		BindAddress: "0.0.0.0",
		BindPort:    8443,
		Required:    true,
		ServerCert: GeneratableKeyCert{
			PairName:      "onex",
			CertDirectory: "/var/run/onex",
		},
	}
}

// Validate 用于解析和验证用户在程序启动时在命令行输入的参数。
func (s *SecureServingOptions) Validate() []error {
	if s == nil {
		return nil
	}

	errors := []error{}

	if s.Required && s.BindPort < 1 || s.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--"+s.fullPrefix+".bind-port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0", s.BindPort))
	} else if s.BindPort < 0 || s.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--"+s.fullPrefix+".bind-port %v must be between 0 and 65535, inclusive. 0 for turning off secure port", s.BindPort))
	}

	return errors
}

// AddFlags 将与特定 API 服务器的 HTTPS 服务器相关的标志添加到
// 指定的 FlagSet。
func (s *SecureServingOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&s.BindAddress, fullPrefix+".bind-address", s.BindAddress, ""+
		"The IP address on which to listen for the --"+fullPrefix+".bind-port port. The "+
		"associated interface(s) must be reachable by the rest of the engine, and by CLI/web "+
		"clients. If blank, all interfaces will be used (0.0.0.0 for all IPv4 interfaces and :: for all IPv6 interfaces).")
	desc := "The port on which to serve HTTPS with authentication and authorization."
	if s.Required {
		desc += " It cannot be switched off with 0."
	} else {
		desc += " If 0, don't serve HTTPS at all."
	}
	fs.IntVar(&s.BindPort, fullPrefix+".bind-port", s.BindPort, desc)

	fs.StringVar(&s.ServerCert.CertDirectory, fullPrefix+".tls.cert-dir", s.ServerCert.CertDirectory, ""+
		"The directory where the TLS certs are located. "+
		"If --"+fullPrefix+".tls.cert-key.cert-file and --"+fullPrefix+".tls.cert-key.private-key-file are provided, "+
		"this flag will be ignored.")

	fs.StringVar(&s.ServerCert.PairName, fullPrefix+".tls.pair-name", s.ServerCert.PairName, ""+
		"The name which will be used with --"+fullPrefix+".tls.cert-dir to make a cert and key filenames. "+
		"It becomes <cert-dir>/<pair-name>.crt and <cert-dir>/<pair-name>.key")

	fs.StringVar(&s.ServerCert.CertKey.CertFile, fullPrefix+".tls.cert-key.cert-file", s.ServerCert.CertKey.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")

	fs.StringVar(&s.ServerCert.CertKey.KeyFile, fullPrefix+".tls.cert-key.private-key-file",
		s.ServerCert.CertKey.KeyFile, ""+
			"File containing the default x509 private key matching --"+fullPrefix+".tls.cert-key.cert-file.")
}

// Complete 填充未设置且需要具有有效数据的字段。
func (s *SecureServingOptions) Complete() error {
	if s == nil || s.BindPort == 0 {
		return nil
	}

	keyCert := &s.ServerCert.CertKey
	if len(keyCert.CertFile) != 0 || len(keyCert.KeyFile) != 0 {
		return nil
	}

	if len(s.ServerCert.CertDirectory) > 0 {
		if len(s.ServerCert.PairName) == 0 {
			return fmt.Errorf("--" + s.fullPrefix + ".tls.pair-name is required if --" + s.fullPrefix + ".tls.cert-dir is set")
		}
		keyCert.CertFile = path.Join(s.ServerCert.CertDirectory, s.ServerCert.PairName+".crt")
		keyCert.KeyFile = path.Join(s.ServerCert.CertDirectory, s.ServerCert.PairName+".key")
	}

	return nil
}

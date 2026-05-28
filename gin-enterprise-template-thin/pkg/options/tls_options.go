package options

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var _ IOptions = (*TLSOptions)(nil)

// TLSOptions 是用于提供安全流量的 TLS 证书信息。
type TLSOptions struct {
	// UseTLS 指定如果可能是否应使用 TLS 加密。
	UseTLS             bool   `json:"use-tls" mapstructure:"use-tls"`
	InsecureSkipVerify bool   `json:"insecure-skip-verify" mapstructure:"insecure-skip-verify"`
	CaCert             string `json:"ca-cert" mapstructure:"ca-cert"`
	Cert               string `json:"cert" mapstructure:"cert"`
	Key                string `json:"key" mapstructure:"key"`
}

// NewTLSOptions 创建一个`零值`实例。
func NewTLSOptions() *TLSOptions {
	return &TLSOptions{}
}

// Validate 验证传递给 TLSOptions 的标志。
func (o *TLSOptions) Validate() []error {
	errs := []error{}

	if !o.UseTLS {
		return errs
	}

	if (o.Cert != "" && o.Key == "") || (o.Cert == "" && o.Key != "") {
		errs = append(errs, fmt.Errorf("only one of cert and key configuration option is setted, you should set both to enable tls"))
	}

	return errs
}

// AddFlags 将与特定 API 服务器的 redis 存储相关的标志添加到指定的 FlagSet。
func (o *TLSOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.UseTLS, fullPrefix+".use-tls", o.UseTLS, "Use tls transport to connect the server.")
	fs.BoolVar(&o.InsecureSkipVerify, fullPrefix+".insecure-skip-verify", o.InsecureSkipVerify, ""+
		"Controls whether a client verifies the server's certificate chain and host name.")
	fs.StringVar(&o.CaCert, fullPrefix+".ca-cert", o.CaCert, "Path to ca cert for connecting to the server.")
	fs.StringVar(&o.Cert, fullPrefix+".cert", o.Cert, "Path to cert file for connecting to the server.")
	fs.StringVar(&o.Key, fullPrefix+".key", o.Key, "Path to key file for connecting to the server.")
}

func (o *TLSOptions) MustTLSConfig() *tls.Config {
	tlsConf, err := o.TLSConfig()
	if err != nil {
		return &tls.Config{}
	}

	return tlsConf
}

func (o *TLSOptions) TLSConfig() (*tls.Config, error) {
	if !o.UseTLS {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.InsecureSkipVerify,
	}

	if o.Cert != "" && o.Key != "" {
		var cert tls.Certificate
		cert, err := tls.LoadX509KeyPair(o.Cert, o.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to loading tls certificates: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if o.CaCert != "" {
		data, err := os.ReadFile(o.CaCert)
		if err != nil {
			return nil, err
		}

		capool := x509.NewCertPool()
		for {
			var block *pem.Block
			block, _ = pem.Decode(data)
			if block == nil {
				break
			}
			cacert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			capool.AddCert(cacert)
		}

		tlsConfig.RootCAs = capool
	}

	return tlsConfig, nil
}

// Scheme returns the URL scheme based on the TLS configuration.
func (o *TLSOptions) Scheme() string {
	if o.UseTLS {
		return "https"
	}
	return "http"
}

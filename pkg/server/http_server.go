package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

var (
	errHTTPOptionsRequired = errors.New("HTTP options are required")
	errHTTPTLSRequired     = errors.New("HTTP TLS certificate and key are required")
)

// HTTPServer 代表一个 HTTP 服务器.
type HTTPServer struct {
	srv     *http.Server
	network string
	useTLS  bool
	listen  func(network, address string) (net.Listener, error)
}

// NewHTTPServer 创建一个新的 HTTP 服务器实例.
func NewHTTPServer(httpOptions *genericoptions.HTTPOptions, tlsOptions *genericoptions.TLSOptions, handler http.Handler) (*HTTPServer, error) {
	if httpOptions == nil {
		return nil, errHTTPOptionsRequired
	}

	var tlsConfig *tls.Config
	if tlsOptions != nil && tlsOptions.UseTLS {
		if tlsOptions.Cert == "" || tlsOptions.Key == "" {
			return nil, errHTTPTLSRequired
		}
		var err error
		tlsConfig, err = tlsOptions.TLSConfig()
		if err != nil {
			return nil, fmt.Errorf("create HTTP TLS config: %w", err)
		}
	}

	return &HTTPServer{
		srv: &http.Server{
			Addr:              httpOptions.Addr,
			Handler:           handler,
			TLSConfig:         tlsConfig,
			ReadHeaderTimeout: httpOptions.Timeout,
			ReadTimeout:       httpOptions.Timeout,
			WriteTimeout:      httpOptions.Timeout,
			IdleTimeout:       httpOptions.Timeout,
		},
		network: httpOptions.Network,
		useTLS:  tlsConfig != nil,
		listen:  net.Listen,
	}, nil
}

// Run 启动 HTTP 服务器，并把监听或服务错误返回给调用方。
func (s *HTTPServer) Run() error {
	listener, err := s.listen(s.network, s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s %s: %w", s.network, s.srv.Addr, err)
	}

	slog.Info("Start to listening the incoming requests",
		"protocol", protocolName(s.srv),
		"addr", listener.Addr().String())
	serveFn := func() error { return s.srv.Serve(listener) }
	if s.useTLS {
		serveFn = func() error { return s.srv.ServeTLS(listener, "", "") }
	}

	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP server: %w", err)
	}
	return nil
}

// GracefulStop 优雅地关闭 HTTP 服务器.
func (s *HTTPServer) GracefulStop(ctx context.Context) error {
	slog.Info("Gracefully stop HTTP(s) server")
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP(s) server forced to shutdown", "err", err)
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}

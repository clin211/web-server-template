package server

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

// HTTPServer 代表一个 HTTP 服务器.
type HTTPServer struct {
	srv *http.Server
}

// NewHTTPServer 创建一个新的 HTTP 服务器实例.
func NewHTTPServer(httpOptions *genericoptions.HTTPOptions, tlsOptions *genericoptions.TLSOptions, handler http.Handler) *HTTPServer {
	var tlsConfig *tls.Config
	if tlsOptions != nil && tlsOptions.UseTLS {
		tlsConfig = tlsOptions.MustTLSConfig()
	}

	return &HTTPServer{
		srv: &http.Server{
			Addr:      httpOptions.Addr,
			Handler:   handler,
			TLSConfig: tlsConfig,
		},
	}
}

// RunOrDie 启动 HTTP 服务器并在出错时记录致命错误.
func (s *HTTPServer) RunOrDie() {
	slog.Info("Start to listening the incoming requests",
		"protocol", protocolName(s.srv),
		"addr", s.srv.Addr)
	// 默认启动 HTTP 服务器
	serveFn := func() error { return s.srv.ListenAndServe() }
	if s.srv.TLSConfig != nil {
		serveFn = func() error { return s.srv.ListenAndServeTLS("", "") }
	}

	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to server HTTP(s) server", "err", err)
	}
}

// GracefulStop 优雅地关闭 HTTP 服务器.
func (s *HTTPServer) GracefulStop(ctx context.Context) {
	slog.Info("Gracefully stop HTTP(s) server")
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP(s) server forced to shutdown", "err", err)
	}
}

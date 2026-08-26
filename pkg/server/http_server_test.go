package server

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	timeout := 17 * time.Second
	srv, err := NewHTTPServer(&genericoptions.HTTPOptions{
		Network: "tcp",
		Addr:    "127.0.0.1:0",
		Timeout: timeout,
	}, nil, http.NotFoundHandler())
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}

	if srv.srv.ReadHeaderTimeout != timeout {
		t.Errorf("ReadHeaderTimeout = %s, want %s", srv.srv.ReadHeaderTimeout, timeout)
	}
	if srv.srv.ReadTimeout != timeout {
		t.Errorf("ReadTimeout = %s, want %s", srv.srv.ReadTimeout, timeout)
	}
	if srv.srv.WriteTimeout != timeout {
		t.Errorf("WriteTimeout = %s, want %s", srv.srv.WriteTimeout, timeout)
	}
	if srv.srv.IdleTimeout != timeout {
		t.Errorf("IdleTimeout = %s, want %s", srv.srv.IdleTimeout, timeout)
	}
}

func TestHTTPServerRunReturnsListenError(t *testing.T) {
	srv, err := NewHTTPServer(&genericoptions.HTTPOptions{
		Network: "tcp",
		Addr:    "127.0.0.1:8080",
		Timeout: time.Second,
	}, nil, http.NotFoundHandler())
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	wantErr := &net.OpError{Op: "listen", Net: "tcp", Err: errors.New("address already in use")}
	srv.listen = func(string, string) (net.Listener, error) {
		return nil, wantErr
	}

	err = srv.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want listen error")
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Fatalf("Run() error = %T %v, want wrapped *net.OpError", err, err)
	}
	if opErr != wantErr {
		t.Fatalf("Run() wrapped error = %v, want %v", opErr, wantErr)
	}
}

func TestNewHTTPServerReturnsTLSConfigError(t *testing.T) {
	_, err := NewHTTPServer(&genericoptions.HTTPOptions{
		Network: "tcp",
		Addr:    "127.0.0.1:0",
		Timeout: time.Second,
	}, &genericoptions.TLSOptions{
		UseTLS: true,
		Cert:   "testdata/missing.crt",
		Key:    "testdata/missing.key",
	}, http.NotFoundHandler())
	if err == nil {
		t.Fatal("NewHTTPServer() error = nil, want TLS config error")
	}
	if got := err.Error(); !strings.Contains(got, "create HTTP TLS config") {
		t.Fatalf("NewHTTPServer() error = %q, want TLS config context", got)
	}
}

func TestNewHTTPServerRequiresTLSCertificate(t *testing.T) {
	_, err := NewHTTPServer(&genericoptions.HTTPOptions{
		Network: "tcp",
		Addr:    "127.0.0.1:0",
		Timeout: time.Second,
	}, &genericoptions.TLSOptions{UseTLS: true}, http.NotFoundHandler())
	if err == nil {
		t.Fatal("NewHTTPServer() error = nil, want missing TLS certificate error")
	}
	if got := err.Error(); !strings.Contains(got, "certificate and key are required") {
		t.Fatalf("NewHTTPServer() error = %q, want missing certificate context", got)
	}
}

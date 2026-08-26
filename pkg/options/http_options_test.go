package options

import (
	"strings"
	"testing"
	"time"
)

func TestHTTPOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*HTTPOptions)
		wantErr string
	}{
		{name: "valid defaults"},
		{
			name:    "non-positive timeout",
			mutate:  func(o *HTTPOptions) { o.Timeout = 0 },
			wantErr: "http timeout must be positive",
		},
		{
			name: "wildcard CORS origin",
			mutate: func(o *HTTPOptions) {
				o.AllowedOrigins = []string{"*"}
			},
			wantErr: "wildcard CORS origin is not allowed",
		},
		{
			name: "origin with path",
			mutate: func(o *HTTPOptions) {
				o.AllowedOrigins = []string{"https://example.com/api"}
			},
			wantErr: "invalid CORS origin",
		},
		{
			name: "explicit origins",
			mutate: func(o *HTTPOptions) {
				o.AllowedOrigins = []string{"https://console.example.com", "http://localhost:3000"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewHTTPOptions()
			if tt.mutate != nil {
				tt.mutate(opts)
			}

			errs := opts.Validate()
			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("Validate() errors = %v, want none", errs)
				}
				return
			}
			for _, err := range errs {
				if strings.Contains(err.Error(), tt.wantErr) {
					return
				}
			}
			t.Fatalf("Validate() errors = %v, want error containing %q", errs, tt.wantErr)
		})
	}
}

func TestNewHTTPOptionsDefaults(t *testing.T) {
	opts := NewHTTPOptions()
	if opts.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %s, want 30s", opts.Timeout)
	}
	if len(opts.AllowedOrigins) != 0 {
		t.Fatalf("AllowedOrigins = %v, want default deny", opts.AllowedOrigins)
	}
}

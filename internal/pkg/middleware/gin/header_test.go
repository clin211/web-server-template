package gin

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	ginframework "github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		method         string
		wantStatus     int
		wantOrigin     string
		wantVary       string
		wantHandled    bool
	}{
		{
			name:        "same origin request continues",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantHandled: true,
		},
		{
			name:        "untrusted origin continues without CORS permission",
			origin:      "https://untrusted.example.com",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantVary:    "Origin",
			wantHandled: true,
		},
		{
			name:           "allowed origin receives CORS permission",
			allowedOrigins: []string{"https://console.example.com"},
			origin:         "https://console.example.com",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "https://console.example.com",
			wantVary:       "Origin",
			wantHandled:    true,
		},
		{
			name:           "allowed preflight stops before handler",
			allowedOrigins: []string{"https://console.example.com"},
			origin:         "https://console.example.com",
			method:         http.MethodOptions,
			wantStatus:     http.StatusNoContent,
			wantOrigin:     "https://console.example.com",
			wantVary:       "Origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := false
			engine := ginframework.New()
			engine.Use(CORS(tt.allowedOrigins))
			engine.Any("/resource", func(c *ginframework.Context) {
				handled = true
				c.Status(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, "/resource", nil)
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			engine.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if got := recorder.Header().Get("Vary"); got != tt.wantVary {
				t.Errorf("Vary = %q, want %q", got, tt.wantVary)
			}
			if handled != tt.wantHandled {
				t.Errorf("handler called = %v, want %v", handled, tt.wantHandled)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	tests := []struct {
		name     string
		tls      *tls.ConnectionState
		wantHSTS string
	}{
		{name: "plain HTTP omits HSTS"},
		{name: "TLS enables HSTS", tls: &tls.ConnectionState{}, wantHSTS: "max-age=31536000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := ginframework.New()
			engine.Use(SecurityHeaders)
			engine.GET("/resource", func(c *ginframework.Context) { c.Status(http.StatusOK) })

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/resource", nil)
			request.TLS = tt.tls
			engine.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
				t.Errorf("X-Frame-Options = %q, want DENY", got)
			}
			if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			if got := recorder.Header().Get("Strict-Transport-Security"); got != tt.wantHSTS {
				t.Errorf("Strict-Transport-Security = %q, want %q", got, tt.wantHSTS)
			}
		})
	}
}

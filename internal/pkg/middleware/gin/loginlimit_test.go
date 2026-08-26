package gin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ginframework "github.com/gin-gonic/gin"
)

type fakeLoginLimiter struct {
	allowed  bool
	err      error
	identity string
}

func (l *fakeLoginLimiter) AllowLogin(_ context.Context, identity string, _ int64, _ time.Duration) (bool, error) {
	l.identity = identity
	return l.allowed, l.err
}

func TestLoginRateLimitMiddleware(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	tests := []struct {
		name       string
		limiter    *fakeLoginLimiter
		wantStatus int
		wantNext   bool
	}{
		{name: "allowed", limiter: &fakeLoginLimiter{allowed: true}, wantStatus: http.StatusNoContent, wantNext: true},
		{name: "limited", limiter: &fakeLoginLimiter{}, wantStatus: http.StatusTooManyRequests},
		{name: "redis unavailable", limiter: &fakeLoginLimiter{err: errors.New("redis down")}, wantStatus: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			engine := ginframework.New()
			engine.POST("/login", LoginRateLimitMiddleware(tt.limiter, 10, time.Minute), func(c *ginframework.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/login", nil)
			request.RemoteAddr = "192.0.2.8:12345"
			engine.ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatus || called != tt.wantNext {
				t.Fatalf("status = %d, called = %v", recorder.Code, called)
			}
			if tt.limiter.identity != "192.0.2.8" {
				t.Fatalf("identity = %q, want direct IP", tt.limiter.identity)
			}
		})
	}
}

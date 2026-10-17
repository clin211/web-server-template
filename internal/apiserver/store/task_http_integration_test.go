package store_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/biz"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/handler"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/gin-gonic/gin"
)

type managementLimiter struct {
	allowed bool
	err     error
	calls   int
}

func (l *managementLimiter) AllowLogin(ctx context.Context, key string, max int64, window time.Duration) (bool, error) {
	l.calls++
	return l.allowed, l.err
}
func TestIntegrationDeadLetterHTTPPolicy(t *testing.T) {
	db, core := testDatabase(t)
	bootstrapAdmin(t, db, "owner")
	for _, tc := range []struct {
		name, actor   string
		limiter       *managementLimiter
		status, calls int
	}{
		{"unprivileged", "other", &managementLimiter{allowed: true}, http.StatusForbidden, 0},
		{"limited", "owner", &managementLimiter{}, http.StatusTooManyRequests, 1},
		{"redis unavailable", "owner", &managementLimiter{err: errors.New("redis unavailable")}, http.StatusServiceUnavailable, 1},
		{"limiter missing", "owner", nil, http.StatusServiceUnavailable, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := handler.NewHandler(biz.NewBiz(core, nil), nil, func(c *gin.Context) {
				c.Request = c.Request.WithContext(contextx.WithUserID(c.Request.Context(), tc.actor))
			})
			if tc.limiter != nil {
				h.SetReplayLimiter(tc.limiter)
			}
			engine := gin.New()
			h.InstallAll(engine.Group("/v1"))
			req := httptest.NewRequest(http.MethodPost, "/v1/dead-letters/dead/replay", strings.NewReader(`{"password":"must-not-leak"}`))
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			if strings.Contains(response.Body.String(), "must-not-leak") {
				t.Fatal("response leaked request")
			}
			if tc.limiter != nil && tc.limiter.calls != tc.calls {
				t.Fatal("limiter used before authorization")
			}
		})
	}
	var n int64
	if err := db.Table("audit_log").Where("action='dlq.rate_limited' AND outcome='denied'").Count(&n).Error; err != nil || n != 1 {
		t.Fatal("missing rate limit audit", err)
	}
}

package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeReadinessChecker struct{ err error }

func (c fakeReadinessChecker) Check(context.Context) error { return c.err }

func TestInstallGenericAPIDoesNotExposePprof(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := InstallGenericAPI(engine, "test-service"); err != nil {
		t.Fatalf("InstallGenericAPI() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /debug/pprof/ status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestReadyzReflectsDependencyHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "ready", wantStatus: http.StatusOK},
		{name: "database unavailable", err: errors.New("database unavailable"), wantStatus: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			cfg := &ServerConfig{readiness: fakeReadinessChecker{err: tt.err}}
			engine.GET("/readyz", cfg.Readyz)

			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("GET /readyz status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}

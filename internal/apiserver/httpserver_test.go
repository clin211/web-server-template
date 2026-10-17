package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"

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

func TestNotFoundTranslations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(mw.I18n(localization.New()))
	if err := InstallGenericAPI(engine, "test-service"); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ language, message string }{
		{"", "Page not found."}, {"zh-CN", "页面未找到。"},
	} {
		t.Run(tt.language, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/missing", nil)
			req.Header.Set("Accept-Language", tt.language)
			engine.ServeHTTP(w, req)
			var response errorsx.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != 404 || response.Code != int(errorsx.CodeUserNotFound) || response.Reason != "Common.PageNotFound" || response.Message != tt.message {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}

package gin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	ginframework "github.com/gin-gonic/gin"
)

func TestI18nNegotiationAndCORS(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	engine := ginframework.New()
	engine.Use(I18n(localization.New()), CORS([]string{"https://example.com"}))
	engine.GET("/", func(c *ginframework.Context) {
		c.String(http.StatusOK, i18n.FromContext(c.Request.Context()).T("User.NotFound"))
	})
	for _, tt := range []struct{ language, wantLanguage, wantBody string }{
		{"", "en", "User not found."}, {"zh-CN", "zh-Hans", "用户未找到。"},
	} {
		t.Run(tt.language, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Language", tt.language)
			req.Header.Set("Origin", "https://example.com")
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Body.String() != tt.wantBody || w.Header().Get("Content-Language") != tt.wantLanguage {
				t.Fatalf("headers = %v, body = %s", w.Header(), w.Body.String())
			}
			vary := strings.Join(w.Header().Values("Vary"), ",")
			if !strings.Contains(vary, "Origin") || !strings.Contains(vary, "Accept-Language") {
				t.Fatalf("Vary = %q", vary)
			}
		})
	}
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Headers", "accept-language")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent || !strings.Contains(strings.ToLower(w.Header().Get("Access-Control-Allow-Headers")), "accept-language") {
		t.Fatalf("preflight: status = %d, headers = %v", w.Code, w.Header())
	}
}

func TestLocalizedEarlyErrors(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	translator := localization.New()
	for _, tt := range []struct {
		name           string
		middleware     ginframework.HandlerFunc
		status, code   int
		reason, en, zh string
	}{
		{"authentication", AuthnMiddleware(&fakeUserRetriever{}, authnTestTokenManager()), 401, 50102, "Auth.TokenInvalid", "Invalid token.", "令牌无效。"},
		{"rate limit", LoginRateLimitMiddleware(&fakeLoginLimiter{}, 10, time.Minute), 429, 10104, "Service.TooManyRequests", "Too many requests. Please try again later.", "请求过多，请稍后再试。"},
		{"unavailable", LoginRateLimitMiddleware(nil, 10, time.Minute), 503, 10102, "Service.Unavailable", "Service temporarily unavailable.", "服务暂时不可用。"},
	} {
		for _, lang := range []string{"", "zh-CN"} {
			t.Run(tt.name+lang, func(t *testing.T) {
				engine := ginframework.New()
				engine.Use(I18n(translator), tt.middleware)
				engine.GET("/", func(c *ginframework.Context) { t.Error("handler reached after rejection") })
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept-Language", lang)
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				var response struct {
					Code            int
					Message, Reason string
				}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				want := tt.en
				if lang == "zh-CN" {
					want = tt.zh
				}
				if w.Code != tt.status || response.Code != tt.code || response.Reason != tt.reason || response.Message != want {
					t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
				}
				if tt.status == 429 && w.Header().Get("Retry-After") != "60" {
					t.Fatal("Retry-After changed")
				}
			})
		}
	}
}

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	"github.com/gin-gonic/gin"
)

func TestTaskResponseTranslations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(mw.I18n(localization.New()))
	engine.GET("/accepted", func(c *gin.Context) { reply(c, 202, gin.H{"task_id": "task-1"}, nil) })
	engine.GET("/created", func(c *gin.Context) { reply(c, 201, nil, nil) })
	engine.GET("/conflict", func(c *gin.Context) { reply(c, 0, nil, store.ErrIdempotencyConflict) })
	for _, tt := range []struct {
		path, lang      string
		status, code    int
		message, reason string
	}{
		{"/accepted", "en", 202, 0, "success", ""},
		{"/accepted", "zh-CN", 202, 0, "成功", ""},
		{"/created", "zh-CN", 201, 0, "成功", ""},
		{"/conflict", "", 409, 409, "resource version, state or idempotency conflict", "Task.Conflict"},
		{"/conflict", "zh-CN", 409, 409, "资源版本、状态或幂等键冲突。", "Task.Conflict"},
	} {
		t.Run(tt.path+tt.lang, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Accept-Language", tt.lang)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			var response struct {
				Code            int
				Message, Reason string
				Data            map[string]any
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != tt.status || response.Code != tt.code || response.Message != tt.message || response.Reason != tt.reason {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			if tt.path == "/accepted" && response.Data["task_id"] != "task-1" {
				t.Fatalf("data = %v", response.Data)
			}
			if tt.path == "/created" && response.Data == nil {
				t.Fatal("empty data object missing")
			}
		})
	}
}

package gin_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/localization"
	mw "github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/core"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/clin211/gin-enterprise-template/pkg/i18n"
	"github.com/gin-gonic/gin"
)

func TestLocalizedHTTPResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	base := errorsx.ErrInvalidArgument.WithMessageID("Validation.EmailInvalid")
	original := base.Message
	engine := gin.New()
	engine.Use(mw.I18n(localization.New()))
	engine.GET("/plain", func(c *gin.Context) { core.WriteResponse(c, gin.H{"value": "unchanged"}, nil) })
	engine.GET("/proto", func(c *gin.Context) {
		core.WriteResponse(c, &v1.HealthzResponse{Status: v1.ServiceStatus_Healthy}, nil)
	})
	engine.GET("/error", func(c *gin.Context) { core.WriteResponse(c, nil, base) })
	engine.GET("/direct", func(c *gin.Context) { core.WriteBizError(c, base) })
	engine.GET("/system", func(c *gin.Context) { core.WriteResponse(c, nil, errors.New("private database failure")) })
	engine.GET("/marshal", func(c *gin.Context) { core.WriteResponse(c, &v1.HealthzResponse{Message: "\xff"}, nil) })
	engine.POST("/bind", func(c *gin.Context) {
		var request struct {
			Value int `json:"value"`
		}
		core.WriteResponse(c, nil, core.ShouldBindJSON(c, &request))
	})
	t.Cleanup(func() {
		if base.Message != original {
			t.Error("shared error mutated")
		}
	})
	for _, tt := range []struct {
		path           string
		status, code   int
		reason, en, zh string
	}{
		{"/plain", 200, 0, "", "success", "成功"},
		{"/proto", 200, 0, "", "success", "成功"},
		{"/error", 400, int(errorsx.CodeUserInvalidCredentials), "Common.InvalidArgument", "invalid email format", "邮箱格式无效。"},
		{"/direct", 400, int(errorsx.CodeUserInvalidCredentials), "Common.InvalidArgument", "invalid email format", "邮箱格式无效。"},
		{"/system", 500, int(errorsx.CodeInternalServer), "Common.InternalError", "Internal server error.", "内部服务器错误。"},
		{"/marshal", 500, int(errorsx.CodeInternalServer), "Common.InternalError", "Internal server error.", "内部服务器错误。"},
		{"/bind", 400, int(errorsx.CodeUserInvalidCredentials), "Common.BindError", "Failed to bind request body.", "请求体绑定失败。"},
	} {
		for _, lang := range []string{"en", "zh-CN"} {
			t.Run(tt.path+"/"+lang, func(t *testing.T) {
				t.Parallel()
				method := http.MethodGet
				if tt.path == "/bind" {
					method = http.MethodPost
				}
				req := httptest.NewRequest(method, tt.path, nil)
				req.Header.Set("Accept-Language", lang)
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
				want := tt.en
				if lang == "zh-CN" {
					want = tt.zh
				}
				if w.Code != tt.status || response.Code != tt.code || response.Reason != tt.reason || response.Message != want {
					t.Fatalf("status = %d, body = %s; want status=%d code=%d reason=%s message=%s", w.Code, w.Body.String(), tt.status, tt.code, tt.reason, want)
				}
				if tt.path == "/plain" && response.Data["value"] != "unchanged" {
					t.Fatalf("data = %v", response.Data)
				}
				if tt.path == "/proto" && (response.Data["status"] != "Healthy" || response.Data["version"] != "") {
					t.Fatalf("proto data = %v", response.Data)
				}
			})
		}
	}
}

func TestParameterizedErrorResponsesConcurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	translator := localization.New()
	base := errorsx.ErrInvalidArgument.WithMessageID("Validation.NicknameTooLong")
	engine := gin.New()
	engine.Use(mw.I18n(translator))
	engine.GET("/limit/:max", func(c *gin.Context) {
		max, err := strconv.Atoi(c.Param("max"))
		if err != nil {
			core.WriteResponse(c, nil, errorsx.ErrInvalidArgument)
			return
		}
		core.WriteResponse(c, nil, base.WithMessageParams(i18n.Params{"MaxLength": max}))
	})
	for max := 45; max < 77; max++ {
		for _, lang := range []string{"en", "zh-CN"} {
			t.Run(fmt.Sprintf("%s/%d", lang, max), func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/limit/%d", max), nil)
				req.Header.Set("Accept-Language", lang)
				w := httptest.NewRecorder()
				engine.ServeHTTP(w, req)
				var response map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				want := fmt.Sprintf("nickname must be less than %d characters", max)
				if lang == "zh-CN" {
					want = fmt.Sprintf("昵称长度必须小于 %d 个字符。", max)
				}
				if w.Code != 400 || response["code"] != float64(errorsx.CodeUserInvalidCredentials) || response["reason"] != "Common.InvalidArgument" || response["message"] != want {
					t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
				}
				if len(response) != 3 {
					t.Fatalf("template parameters leaked into response: %s", w.Body.String())
				}
			})
		}
	}
	t.Cleanup(func() {
		ctx := i18n.WithContext(context.Background(), translator)
		if got := base.Localize(ctx).Message; got != "Invalid request parameters." {
			t.Errorf("shared error changed: %q", got)
		}
	})
}

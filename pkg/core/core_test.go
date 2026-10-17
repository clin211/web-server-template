package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestOnInitializeUsesSearchPathWhenConfigFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte("jwt:\n  secret: from-file\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	configFile := ""
	OnInitialize(&configFile, "APP", []string{dir}, "service.yaml")()
	if got := viper.GetString("jwt.secret"); got != "from-file" {
		t.Fatalf("jwt.secret = %q, want from-file", got)
	}
	if err := ConfigError(); err != nil {
		t.Fatalf("ConfigError() = %v, want nil", err)
	}
}

func TestOnInitializeReportsExplicitConfigError(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "missing.yaml")
	OnInitialize(&configFile, "APP", nil, "service.yaml")()
	if err := ConfigError(); err == nil {
		t.Fatal("ConfigError() = nil, want missing config error")
	}
}

func TestOnInitializeEnvironmentOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte("jwt:\n  secret: from-file\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("APP_JWT_SECRET", "from-env")
	configFile := ""
	OnInitialize(&configFile, "APP", []string{dir}, "service.yaml")()
	if got := viper.GetString("jwt.secret"); got != "from-env" {
		t.Fatalf("jwt.secret = %q, want from-env", got)
	}
}

func TestWriteResponseBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/test", func(c *gin.Context) {
		WriteResponse(c, nil, errorsx.NewBizError(errorsx.CodeUserNotFound, "User.NotFound", "用户不存在"))
	})
	req, _ := http.NewRequest("POST", "/test", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(errorsx.CodeUserNotFound), resp["code"])
	assert.Equal(t, "User.NotFound", resp["reason"])
	assert.Equal(t, "用户不存在", resp["message"])
}

// TestWriteResponseProtoMessageEnvelope 回归契约行为：proto 指针响应必须走
// protojson（EmitUnpopulated + 枚举名），输出与 OpenAPI 契约中的
// XxxResponseEnvelope 声明一致（例如 healthz 的 status 为字符串枚举）。
func TestWriteResponseProtoMessageEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.GET("/test", func(c *gin.Context) {
		WriteResponse(c, &v1.HealthzResponse{
			Status:    v1.ServiceStatus_Healthy,
			Timestamp: "2026-08-28T00:00:00Z",
			Message:   "ok",
			Version:   "v0.0.0",
		}, nil)
	})
	req, _ := http.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{
		"code": 0,
		"message": "success",
		"data": {"status": "Healthy", "timestamp": "2026-08-28T00:00:00Z", "message": "ok", "version": "v0.0.0"}
	}`, w.Body.String())
}

func TestWriteResponseSystemError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/test", func(c *gin.Context) {
		WriteResponse(c, nil, errorsx.NewBizError(errorsx.CodeDatabaseReadFailed, "Infra.DatabaseReadFailed", "数据库读取失败"))
	})
	req, _ := http.NewRequest("POST", "/test", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestWriteResponseProtocolError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/test", func(c *gin.Context) {
		WriteResponse(c, nil, errorsx.NewBizError(errorsx.CodeAuthTokenInvalid, "Auth.TokenInvalid", "令牌无效"))
	})
	req, _ := http.NewRequest("POST", "/test", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

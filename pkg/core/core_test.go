package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWriteResponseBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	engine.POST("/test", func(c *gin.Context) {
		WriteResponse(c, nil, errorsx.NewBizError(errorsx.CodeUserNotFound, "User.NotFound", "用户不存在"))
	})
	req, _ := http.NewRequest("POST", "/test", nil)
	engine.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(errorsx.CodeUserNotFound), resp["code"])
	assert.Equal(t, "User.NotFound", resp["reason"])
	assert.Equal(t, "用户不存在", resp["message"])
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

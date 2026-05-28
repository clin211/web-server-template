package token

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

// TestInit 测试 Init 函数
func TestInit(t *testing.T) {
	// 重置配置以确保测试环境干净
	Reset()

	// 测试默认配置
	assert.Equal(t, "Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", config.key)
	assert.Equal(t, "identityKey", config.identityKey)
	assert.Equal(t, 2*time.Hour, config.accessExpiration)
	assert.Equal(t, 7*24*time.Hour, config.refreshExpiration)

	// 测试自定义配置
	Reset()
	Init("newKey", 3*time.Hour, 14*24*time.Hour, WithIdentityKey("newIdentityKey"))

	assert.Equal(t, "newKey", config.key)
	assert.Equal(t, "newIdentityKey", config.identityKey)

	// 再次调用 Init，确保配置不会被覆盖（因为使用了 once.Do）
	Init("anotherKey", 1*time.Hour, 7*24*time.Hour, WithIdentityKey("anotherIdentityKey"))

	assert.Equal(t, "newKey", config.key)                 // 仍然是 "newKey"
	assert.Equal(t, "newIdentityKey", config.identityKey) // 仍然是 "newIdentityKey"

	// 为后续测试重置配置
	Reset()
	Init("Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", 2*time.Hour, 7*24*time.Hour, WithIdentityKey("identityKey"))
}

// TestSign 测试 Sign 函数（双Token）
func TestSign(t *testing.T) {
	identityKey := "testUser"
	accessToken, refreshToken, accessExpireAt, refreshExpireAt, err := Sign(identityKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// 验证过期时间
	assert.True(t, accessExpireAt.After(time.Now()))
	assert.True(t, refreshExpireAt.After(accessExpireAt))

	// 解析 Access Token
	parsedIdentityKey, err := ParseIdentity(accessToken, config.key)
	assert.NoError(t, err)
	assert.Equal(t, identityKey, parsedIdentityKey)

	// 验证 token 类型
	tokenType, err := GetTokenType(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeAccess, tokenType)

	tokenType, err = GetTokenType(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeRefresh, tokenType)

	// 验证 IsAccessToken 和 IsRefreshToken
	assert.True(t, IsAccessToken(accessToken))
	assert.False(t, IsRefreshToken(accessToken))
	assert.True(t, IsRefreshToken(refreshToken))
	assert.False(t, IsAccessToken(refreshToken))
}

// TestParseRefreshToken 测试解析 Refresh Token
func TestParseRefreshToken(t *testing.T) {
	identityKey := "testUser"
	_, refreshToken, _, _, err := Sign(identityKey)
	assert.NoError(t, err)

	// 解析 Refresh Token
	parsedIdentityKey, err := ParseRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, identityKey, parsedIdentityKey)

	// 使用 Access Token 调用 ParseRefreshToken 应该失败
	accessToken, _, _, _, _ := Sign(identityKey)
	_, err = ParseRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Equal(t, ErrNotRefreshToken, err)
}

// TestParseInvalidToken 测试解析无效的 token
func TestParseInvalidToken(t *testing.T) {
	invalidToken := "invalid.token.string"
	identityKey, err := ParseIdentity(invalidToken, config.key)

	assert.Error(t, err)
	assert.Empty(t, identityKey)
}

// TestParseRequestWithGin 测试从 Gin 上下文解析 token
func TestParseRequestWithGin(t *testing.T) {
	// 设置 Gin 上下文
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ")

	// 创建 Gin 上下文
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// 测试解析请求
	identityKey, err := ParseRequest(c)

	assert.Error(t, err)
	assert.Empty(t, identityKey)

	// 测试有效的 Access Token
	accessToken, _, _, _, _ := Sign("testUser")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	c.Request = req

	identityKey, err = ParseRequest(c)
	assert.NoError(t, err)
	assert.Equal(t, "testUser", identityKey)

	// 测试使用 Refresh Token 应该失败
	_, refreshToken, _, _, _ := Sign("testUser")
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	c.Request = req

	identityKey, err = ParseRequest(c)
	assert.Error(t, err)
	assert.Equal(t, ErrNotAccessToken, err)
}

// TestParseRequestWithGRPC 测试从 gRPC 上下文解析 token
func TestParseRequestWithGRPC(t *testing.T) {
	// 创建 gRPC 上下文
	md := metadata.New(map[string]string{"Authorization": "Bearer "})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// 测试解析请求
	identityKey, err := ParseRequest(ctx)

	assert.Error(t, err)
	assert.Empty(t, identityKey)

	// 测试有效的 Access Token
	accessToken, _, _, _, _ := Sign("testUser")
	md = metadata.New(map[string]string{"Authorization": "Bearer " + accessToken})
	ctx = metadata.NewIncomingContext(context.Background(), md)

	identityKey, err = ParseRequest(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "testUser", identityKey)
}

// TestGetTokenType 测试获取 token 类型
func TestGetTokenType(t *testing.T) {
	accessToken, refreshToken, _, _, err := Sign("testUser")
	assert.NoError(t, err)

	// 测试 Access Token
	tokenType, err := GetTokenType(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeAccess, tokenType)

	// 测试 Refresh Token
	tokenType, err = GetTokenType(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, TokenTypeRefresh, tokenType)

	// 测试无效 token
	tokenType, err = GetTokenType("invalid.token")
	assert.Error(t, err)
	assert.Empty(t, tokenType)

	// 测试没有 token_type 的 token（使用 SignWithClaims 创建）
	claimsWithoutType := map[string]interface{}{
		"user_id": "123",
	}
	oldToken, _, _ := SignWithClaims(claimsWithoutType)
	tokenType, err = GetTokenType(oldToken)
	assert.Error(t, err)
	assert.Equal(t, ErrMissingTokenType, err)
}

// TestGetAccessExpiration 测试获取 Access Token 过期时间
func TestGetAccessExpiration(t *testing.T) {
	duration := GetAccessExpiration()
	assert.Equal(t, 2*time.Hour, duration)
}

// TestGetRefreshExpiration 测试获取 Refresh Token 过期时间
func TestGetRefreshExpiration(t *testing.T) {
	duration := GetRefreshExpiration()
	assert.Equal(t, 7*24*time.Hour, duration)
}

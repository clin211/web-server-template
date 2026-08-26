package token

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

type fakeRefreshTokenStore struct {
	used map[string]struct{}
}

func (s *fakeRefreshTokenStore) Consume(_ context.Context, tokenID string, _ time.Duration) (bool, error) {
	if _, exists := s.used[tokenID]; exists {
		return false, nil
	}
	s.used[tokenID] = struct{}{}
	return true, nil
}

func initTestConfig() {
	Reset()
	Init("Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", 2*time.Hour, 7*24*time.Hour, WithIdentityKey("identityKey"))
}

// TestInit 测试 Init 函数
func TestInit(t *testing.T) {
	// 重置配置以确保测试环境干净
	Reset()

	// 测试默认配置
	cfg := GetConfig()
	assert.Equal(t, "Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", cfg.key)
	assert.Equal(t, "identityKey", cfg.identityKey)
	assert.Equal(t, 2*time.Hour, cfg.accessExpiration)
	assert.Equal(t, 7*24*time.Hour, cfg.refreshExpiration)

	// 测试自定义配置
	Reset()
	Init("newKey", 3*time.Hour, 14*24*time.Hour, WithIdentityKey("newIdentityKey"))

	cfg = GetConfig()
	assert.Equal(t, "newKey", cfg.key)
	assert.Equal(t, "newIdentityKey", cfg.identityKey)

	// 再次调用 Init，确保配置不会被覆盖（因为使用了 once.Do）
	Init("anotherKey", 1*time.Hour, 7*24*time.Hour, WithIdentityKey("anotherIdentityKey"))

	cfg = GetConfig()
	assert.Equal(t, "newKey", cfg.key)                 // 仍然是 "newKey"
	assert.Equal(t, "newIdentityKey", cfg.identityKey) // 仍然是 "newIdentityKey"

	// 为后续测试重置配置
	Reset()
	Init("Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5", 2*time.Hour, 7*24*time.Hour, WithIdentityKey("identityKey"))
}

// TestSign 测试 Sign 函数（双Token）
func TestSign(t *testing.T) {
	initTestConfig()

	identityKey := "testUser"
	accessToken, refreshToken, accessExpireAt, refreshExpireAt, err := Sign(identityKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// 验证过期时间
	assert.True(t, accessExpireAt.After(time.Now()))
	assert.True(t, refreshExpireAt.After(accessExpireAt))

	// 解析 Access Token
	cfg := GetConfig()
	parsedIdentityKey, err := ParseIdentity(accessToken, cfg.key)
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

func TestSignWithVersionCarriesTokenVersion(t *testing.T) {
	initTestConfig()

	accessToken, refreshToken, _, _, err := SignWithVersion("user-1", 7)
	assert.NoError(t, err)
	for _, tokenString := range []string{accessToken, refreshToken} {
		version, versionErr := GetTokenVersion(tokenString)
		assert.NoError(t, versionErr)
		assert.Equal(t, int64(7), version)
	}
}

func TestManagersKeepSigningConfigurationIsolated(t *testing.T) {
	first := NewManager("first-manager-key-with-sufficient-length", time.Hour, 24*time.Hour, WithIdentityKey("user_id"))
	second := NewManager("second-manager-key-with-sufficient-length", time.Hour, 24*time.Hour, WithIdentityKey("user_id"))

	_, refreshToken, _, _, err := first.SignWithVersion("user-1", 3)
	assert.NoError(t, err)
	identity, version, err := first.ParseRefreshTokenWithVersion(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", identity)
	assert.Equal(t, int64(3), version)
	_, _, err = second.ParseRefreshTokenWithVersion(refreshToken)
	assert.Error(t, err)
}

func TestManagerRotatesRefreshTokenOnce(t *testing.T) {
	store := &fakeRefreshTokenStore{used: make(map[string]struct{})}
	manager := NewManagerWithRefreshStore(store, "rotation-test-key-with-sufficient-length", time.Hour, 24*time.Hour, WithIdentityKey("user_id"))
	_, oldRefreshToken, _, _, err := manager.SignWithVersion("user-1", 4)
	assert.NoError(t, err)

	accessToken, newRefreshToken, _, _, err := manager.RotateRefreshToken(context.Background(), oldRefreshToken, "user-1", 4)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, oldRefreshToken, newRefreshToken)

	_, _, _, _, err = manager.RotateRefreshToken(context.Background(), oldRefreshToken, "user-1", 4)
	assert.True(t, errors.Is(err, ErrRefreshTokenReused))
}

// TestParseRefreshToken 测试解析 Refresh Token
func TestParseRefreshToken(t *testing.T) {
	initTestConfig()

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
	initTestConfig()

	invalidToken := "invalid.token.string"
	cfg := GetConfig()
	identityKey, err := ParseIdentity(invalidToken, cfg.key)

	assert.Error(t, err)
	assert.Empty(t, identityKey)
}

// TestParseRequestWithGin 测试从 Gin 上下文解析 token
func TestParseRequestWithGin(t *testing.T) {
	initTestConfig()

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

func TestParseAuthorizationHeaderRejectsTrailingContent(t *testing.T) {
	t.Parallel()

	for _, header := range []string{
		"Bearer token trailing",
		"Bearer  token",
		"bearer token",
		"Bearer\ttoken",
	} {
		if _, err := ParseAuthorizationHeader(header); err == nil {
			t.Errorf("ParseAuthorizationHeader(%q) error = nil", header)
		}
	}
	if tokenString, err := ParseAuthorizationHeader("Bearer token"); err != nil || tokenString != "token" {
		t.Fatalf("ParseAuthorizationHeader(valid) = %q, %v", tokenString, err)
	}
}

func TestParseWithKeyRejectsNonHS256HMAC(t *testing.T) {
	t.Parallel()

	key := "algorithm-test-key-with-sufficient-length"
	tokenObject := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
	tokenString, err := tokenObject.SignedString([]byte(key))
	assert.NoError(t, err)
	_, err = ParseWithKey(tokenString, key)
	assert.Error(t, err)
}

// TestParseRequestWithGRPC 测试从 gRPC 上下文解析 token
func TestParseRequestWithGRPC(t *testing.T) {
	initTestConfig()

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
	initTestConfig()

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
	initTestConfig()

	duration := GetAccessExpiration()
	assert.Equal(t, 2*time.Hour, duration)
}

// TestGetRefreshExpiration 测试获取 Refresh Token 过期时间
func TestGetRefreshExpiration(t *testing.T) {
	initTestConfig()

	duration := GetRefreshExpiration()
	assert.Equal(t, 7*24*time.Hour, duration)
}

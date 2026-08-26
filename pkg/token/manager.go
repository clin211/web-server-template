package token

import (
	"context"
	"fmt"
	"math"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

// Manager 持有单个服务实例不可变的 JWT 签发和校验配置。
type Manager struct {
	cfg           Config
	refreshTokens RefreshTokenStore
}

// RefreshTokenStore 原子记录已消费的 refresh token ID。
type RefreshTokenStore interface {
	Consume(ctx context.Context, tokenID string, expiration time.Duration) (bool, error)
}

// NewManager 创建不依赖包级可变状态的 JWT 管理器。
func NewManager(key string, accessExpiration, refreshExpiration time.Duration, opts ...Option) *Manager {
	return NewManagerWithRefreshStore(nil, key, accessExpiration, refreshExpiration, opts...)
}

// NewManagerWithRefreshStore 创建启用 refresh token 单次消费的 JWT 管理器。
func NewManagerWithRefreshStore(store RefreshTokenStore, key string, accessExpiration, refreshExpiration time.Duration, opts ...Option) *Manager {
	cfg := defaultConfig()
	cfg.key = key
	if accessExpiration > 0 {
		cfg.accessExpiration = accessExpiration
	}
	if refreshExpiration > 0 {
		cfg.refreshExpiration = refreshExpiration
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Manager{cfg: cloneConfig(cfg), refreshTokens: store}
}

// SignWithVersion 签发携带账号令牌版本的 Access Token 和 Refresh Token。
func (m *Manager) SignWithVersion(identity string, version int64) (string, string, time.Time, time.Time, error) {
	if m == nil {
		return "", "", time.Time{}, time.Time{}, jwt.ErrInvalidKey
	}
	return signWithConfig(m.cfg, identity, version)
}

// ParseRequestWithVersion 从请求中校验 Access Token 并返回身份和令牌版本。
func (m *Manager) ParseRequestWithVersion(ctx context.Context) (string, int64, error) {
	tokenString, err := extractTokenFromRequest(ctx)
	if err != nil {
		return "", 0, err
	}
	return m.parseWithVersion(tokenString, TokenTypeAccess)
}

// ParseRefreshTokenWithVersion 校验 Refresh Token 并返回身份和令牌版本。
func (m *Manager) ParseRefreshTokenWithVersion(tokenString string) (string, int64, error) {
	return m.parseWithVersion(tokenString, TokenTypeRefresh)
}

// RotateRefreshToken 原子消费旧 Refresh Token，并签发新的令牌对。
func (m *Manager) RotateRefreshToken(ctx context.Context, tokenString, expectedIdentity string, expectedVersion int64) (string, string, time.Time, time.Time, error) {
	if m == nil || m.refreshTokens == nil {
		return "", "", time.Time{}, time.Time{}, ErrRefreshTokenStoreRequired
	}
	claims, identity, version, err := m.parseClaimsWithVersion(tokenString, TokenTypeRefresh)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	if identity != expectedIdentity {
		return "", "", time.Time{}, time.Time{}, ErrInvalidIdentityKey
	}
	if version != expectedVersion {
		return "", "", time.Time{}, time.Time{}, ErrInvalidTokenVersion
	}
	tokenID, ok := claims["jti"].(string)
	if !ok || tokenID == "" {
		return "", "", time.Time{}, time.Time{}, ErrMissingTokenID
	}
	expValue, ok := claims["exp"].(float64)
	if !ok || expValue <= 0 || expValue > math.MaxInt64 {
		return "", "", time.Time{}, time.Time{}, ErrInvalidTokenClaims
	}
	expiresAt := time.Unix(int64(expValue), 0)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return "", "", time.Time{}, time.Time{}, jwt.ErrTokenExpired
	}

	accessToken, refreshToken, accessExpireAt, refreshExpireAt, err := m.SignWithVersion(identity, version)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, err
	}
	consumed, err := m.refreshTokens.Consume(ctx, tokenID, ttl)
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("consume refresh token: %w", err)
	}
	if !consumed {
		return "", "", time.Time{}, time.Time{}, ErrRefreshTokenReused
	}
	return accessToken, refreshToken, accessExpireAt, refreshExpireAt, nil
}

func (m *Manager) parseWithVersion(tokenString, expectedType string) (string, int64, error) {
	_, identity, version, err := m.parseClaimsWithVersion(tokenString, expectedType)
	return identity, version, err
}

func (m *Manager) parseClaimsWithVersion(tokenString, expectedType string) (jwt.MapClaims, string, int64, error) {
	if m == nil || m.cfg.key == "" {
		return nil, "", 0, jwt.ErrInvalidKey
	}
	claims, err := ParseWithKey(tokenString, m.cfg.key)
	if err != nil {
		return nil, "", 0, err
	}
	tokenType, ok := claims["token_type"].(string)
	if !ok {
		return nil, "", 0, ErrInvalidTokenType
	}
	if tokenType != expectedType {
		if expectedType == TokenTypeRefresh {
			return nil, "", 0, ErrNotRefreshToken
		}
		return nil, "", 0, ErrNotAccessToken
	}
	identity, err := extractIdentityForKey(claims, m.cfg.identityKey)
	if err != nil {
		return nil, "", 0, err
	}
	version, err := tokenVersionFromClaims(claims)
	if err != nil {
		return nil, "", 0, err
	}
	return claims, identity, version, nil
}

func extractIdentityForKey(claims jwt.MapClaims, identityKey string) (string, error) {
	if identityKey == "" {
		return "", nil
	}
	value, exists := claims[identityKey]
	if !exists {
		return "", ErrMissingIdentityKey
	}
	identity, ok := value.(string)
	if !ok || identity == "" {
		return "", ErrInvalidIdentityKey
	}
	return identity, nil
}

func tokenVersionFromClaims(claims jwt.MapClaims) (int64, error) {
	value, exists := claims["token_version"]
	if !exists {
		return 0, ErrMissingTokenVersion
	}
	version, ok := value.(float64)
	if !ok || version < 0 || version > float64(^uint64(0)>>1) || version != float64(int64(version)) {
		return 0, ErrInvalidTokenVersion
	}
	return int64(version), nil
}

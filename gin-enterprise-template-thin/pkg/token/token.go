package token

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Config 包括 token 包的配置选项.
type Config struct {
	// key 用于签发和解析 token 的密钥.
	key string
	// identityKey 是 token 中用户身份的键.
	identityKey string
	// accessExpiration 是 Access Token 的过期时间
	accessExpiration time.Duration
	// refreshExpiration 是 Refresh Token 的过期时间
	refreshExpiration time.Duration
	// skipPaths 需要跳过认证的路径列表
	skipPaths []string
}

// Option 用于配置 token 包的选项
type Option func(*Config)

// Token 类型常量
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	config = Config{
		key:               "",
		identityKey:       "",
		accessExpiration:  2 * time.Hour,
		refreshExpiration: 7 * 24 * time.Hour,
		skipPaths:         []string{}, // 默认不跳过任何路径
	}
	once sync.Once // 确保配置只被初始化一次
)

// 预定义错误
var (
	ErrMissingIdentityKey  = errors.New("missing identity key in token")
	ErrInvalidIdentityKey  = errors.New("invalid identity key in token")
	ErrEmptyToken          = errors.New("token is empty")
	ErrInvalidAuthHeader   = errors.New("invalid authorization header")
	ErrEmptyAuthHeader     = errors.New("authorization header is empty")
	ErrMalformedAuthHeader = errors.New("malformed authorization header")
	ErrInvalidTokenClaims  = errors.New("invalid token claims")
	ErrPathSkipped         = errors.New("path is skipped for authentication")
	ErrInvalidTokenType    = errors.New("invalid token type")
	ErrNotRefreshToken     = errors.New("token is not a refresh token")
	ErrNotAccessToken      = errors.New("token is not an access token")
	ErrMissingTokenType    = errors.New("missing token type in claims")
)

// WithKey 设置签名密钥
func WithKey(key string) Option {
	return func(c *Config) {
		if key != "" {
			c.key = key
		}
	}
}

// WithIdentityKey 设置身份键名称
func WithIdentityKey(identityKey string) Option {
	return func(c *Config) {
		c.identityKey = identityKey // 允许设置为空字符串来禁用身份键验证
	}
}

// WithSkipPaths 设置需要跳过认证的路径列表
// 支持精确匹配和通配符匹配
func WithSkipPaths(paths ...string) Option {
	return func(c *Config) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

// WithCommonSkipPaths 添加常见的跳过路径（健康检查、监控等）
func WithCommonSkipPaths() Option {
	commonPaths := []string{
		"/health",
		"/healthz",
		"/health/*",
		"/ready",
		"/readiness",
		"/live",
		"/liveness",
		"/metrics",
		"/prometheus",
		"/status",
		"/ping",
		"/version",
		"/info",
		"/favicon.ico",
		"/robots.txt",
	}
	return func(c *Config) {
		c.skipPaths = append(c.skipPaths, commonPaths...)
	}
}

// WithSkipPathsPattern 使用模式匹配添加跳过路径
func WithSkipPathsPattern(patterns ...string) Option {
	return func(c *Config) {
		c.skipPaths = append(c.skipPaths, patterns...)
	}
}

// Init 设置包级别的配置 config, config 会用于本包后面的 token 签发和解析.
func Init(key string, accessExpiration, refreshExpiration time.Duration, opts ...Option) {
	once.Do(func() {
		if key != "" {
			config.key = key // 设置密钥
		}
		if accessExpiration > 0 {
			config.accessExpiration = accessExpiration
		}
		if refreshExpiration > 0 {
			config.refreshExpiration = refreshExpiration
		}

		// 应用所有配置选项
		for _, opt := range opts {
			opt(&config)
		}
	})
}

// Reset 重置配置（主要用于测试）
func Reset() {
	once = sync.Once{}
	config = Config{
		key:               "Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5",
		identityKey:       "identityKey",
		accessExpiration:  2 * time.Hour,
		refreshExpiration: 7 * 24 * time.Hour,
		skipPaths:         []string{},
	}
}

// shouldSkipPath 检查路径是否应该跳过认证
func shouldSkipPath(requestPath string) bool {
	for _, skipPath := range config.skipPaths {
		if matchPath(requestPath, skipPath) {
			return true
		}
	}
	return false
}

// matchPath 路径匹配函数，支持通配符
func matchPath(requestPath, pattern string) bool {
	// 精确匹配
	if requestPath == pattern {
		return true
	}

	// 通配符匹配
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(requestPath, prefix)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// 支持中间通配符（简单实现）
	if strings.Contains(pattern, "*") {
		return matchWildcard(requestPath, pattern)
	}

	return false
}

// matchWildcard 通配符匹配（简单实现）
func matchWildcard(str, pattern string) bool {
	// 将模式分割为非通配符部分
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return str == pattern
	}

	// 检查开头
	if !strings.HasPrefix(str, parts[0]) {
		return false
	}

	// 检查结尾
	if len(parts[len(parts)-1]) > 0 && !strings.HasSuffix(str, parts[len(parts)-1]) {
		return false
	}

	// 简单的中间匹配检查
	currentPos := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		if part == "" {
			continue
		}

		index := strings.Index(str[currentPos:], part)
		if index == -1 {
			return false
		}
		currentPos += index + len(part)
	}

	return true
}

// ParseIdentity 使用指定的密钥 key 解析 token，解析成功返回 token 上下文，否则报错.
func ParseIdentity(tokenString string, key string) (string, error) {
	if tokenString == "" {
		return "", ErrEmptyToken
	}

	if key == "" {
		return "", jwt.ErrInvalidKey
	}

	// 解析 token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 确保 token 加密算法符合预期的加密算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})
	if err != nil {
		return "", err
	}

	// 验证 token 有效性
	if !token.Valid {
		return "", jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidTokenClaims
	}

	return extractIdentity(claims)
}

// extractIdentity 从 claims 中提取身份信息
func extractIdentity(claims jwt.MapClaims) (string, error) {
	// 如果没有配置身份键，返回空字符串（表示不需要身份验证）
	if config.identityKey == "" {
		return "", nil
	}

	// 检查身份键是否存在
	value, exists := claims[config.identityKey]
	if !exists {
		return "", ErrMissingIdentityKey
	}

	// 验证身份键的值
	identity, ok := value.(string)
	if !ok {
		return "", ErrInvalidIdentityKey
	}

	// 身份键不能为空（如果配置了身份键）
	if identity == "" {
		return "", ErrInvalidIdentityKey
	}

	return identity, nil
}

// ParseRequest 从请求头中获取令牌，并将其传递递给 Parse 函数以解析令牌
// 验证 token_type 必须为 "access"
func ParseRequest(ctx context.Context) (string, error) {
	// 检查是否应该跳过认证
	if shouldSkipRequestPath(ctx) {
		return "", nil // 返回特殊错误表示路径被跳过
	}

	tokenString, err := extractTokenFromRequest(ctx)
	if err != nil {
		return "", err
	}

	// 验证 token 类型
	tokenType, err := GetTokenType(tokenString)
	if err != nil {
		return "", err
	}
	if tokenType != TokenTypeAccess {
		return "", ErrNotAccessToken
	}

	return ParseIdentity(tokenString, config.key)
}

// shouldSkipRequestPath 检查请求路径是否应该跳过认证
func shouldSkipRequestPath(ctx context.Context) bool {
	switch typed := ctx.(type) {
	case *gin.Context:
		return shouldSkipPath(typed.Request.URL.Path)
	default:
		// 对于gRPC，可以从metadata中获取method信息
		// 这里简化处理，如果需要可以扩展
		return false
	}
}

// ParseRequestIgnoreSkip 强制解析请求，忽略跳过路径设置
// 验证 token_type 必须为 "access"
func ParseRequestIgnoreSkip(ctx context.Context) (string, error) {
	tokenString, err := extractTokenFromRequest(ctx)
	if err != nil {
		return "", err
	}

	// 验证 token 类型
	tokenType, err := GetTokenType(tokenString)
	if err != nil {
		return "", err
	}
	if tokenType != TokenTypeAccess {
		return "", ErrNotAccessToken
	}

	return ParseIdentity(tokenString, config.key)
}

// extractTokenFromRequest 从不同类型的请求上下文中提取 token
func extractTokenFromRequest(ctx context.Context) (string, error) {
	switch typed := ctx.(type) {
	case *gin.Context:
		return extractTokenFromGin(typed)
	default:
		return extractTokenFromGRPC(ctx)
	}
}

// extractTokenFromGin 从 Gin Context 中提取 token
func extractTokenFromGin(c *gin.Context) (string, error) {
	header := c.Request.Header.Get("Authorization")
	if header == "" {
		return "", ErrEmptyAuthHeader
	}

	return parseAuthorizationHeader(header)
}

// extractTokenFromGRPC 从 gRPC Context 中提取 token
func extractTokenFromGRPC(ctx context.Context) (string, error) {
	token, err := auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}
	return token, nil
}

// parseAuthorizationHeader 解析 Authorization 头部
func parseAuthorizationHeader(header string) (string, error) {
	var token string
	n, err := fmt.Sscanf(header, "Bearer %s", &token)
	if err != nil || n != 1 {
		return "", ErrMalformedAuthHeader
	}

	if token == "" {
		return "", ErrEmptyToken
	}

	return token, nil
}

// Sign 使用 jwtSecret 签发 Access Token 和 Refresh Token 对
func Sign(identityValue string) (accessToken, refreshToken string, accessExpireAt, refreshExpireAt time.Time, err error) {
	if config.key == "" {
		return "", "", time.Time{}, time.Time{}, jwt.ErrInvalidKey
	}

	now := time.Now()
	accessExpireAt = now.Add(config.accessExpiration)
	refreshExpireAt = now.Add(config.refreshExpiration)

	// 签发 Access Token
	accessClaims := jwt.MapClaims{
		"token_type": TokenTypeAccess,
		"nbf":        now.Unix(),
		"iat":        now.Unix(),
		"exp":        accessExpireAt.Unix(),
	}
	if config.identityKey != "" && identityValue != "" {
		accessClaims[config.identityKey] = identityValue
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(config.key))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	// 签发 Refresh Token
	refreshClaims := jwt.MapClaims{
		"token_type": TokenTypeRefresh,
		"nbf":        now.Unix(),
		"iat":        now.Unix(),
		"exp":        refreshExpireAt.Unix(),
	}
	if config.identityKey != "" && identityValue != "" {
		refreshClaims[config.identityKey] = identityValue
	}

	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(config.key))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, accessExpireAt, refreshExpireAt, nil
}

// SignWithClaims 使用自定义 claims 签发 token（使用 accessExpiration）
func SignWithClaims(customClaims jwt.MapClaims) (string, time.Time, error) {
	if config.key == "" {
		return "", time.Time{}, jwt.ErrInvalidKey
	}

	now := time.Now()
	expireAt := now.Add(config.accessExpiration)

	// 合并自定义 claims 和必要的时间字段
	claims := make(jwt.MapClaims)
	for k, v := range customClaims {
		claims[k] = v
	}

	// 确保必要的时间字段存在
	if _, exists := claims["nbf"]; !exists {
		claims["nbf"] = now.Unix()
	}
	if _, exists := claims["iat"]; !exists {
		claims["iat"] = now.Unix()
	}
	if _, exists := claims["exp"]; !exists {
		claims["exp"] = expireAt.Unix()
	}

	// 创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签发 token
	tokenString, err := token.SignedString([]byte(config.key))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expireAt, nil
}

// Parse 验证 token 字符串的有效性（不解析身份信息）
func Parse(tokenString string) error {
	if tokenString == "" {
		return ErrEmptyToken
	}

	if config.key == "" {
		return jwt.ErrInvalidKey
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.key), nil
	})
	if err != nil {
		return err
	}

	if !token.Valid {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

// GetClaims 获取 token 中的所有 claims
func GetClaims(tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	if config.key == "" {
		return nil, jwt.ErrInvalidKey
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.key), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidTokenClaims
	}

	return claims, nil
}

// GetConfig 获取当前配置（用于调试和测试）
func GetConfig() Config {
	return config
}

// IsIdentityRequired 检查是否需要身份验证
func IsIdentityRequired() bool {
	return config.identityKey != ""
}

// GetAccessExpiration 获取 Access Token 过期时间
func GetAccessExpiration() time.Duration {
	return config.accessExpiration
}

// GetRefreshExpiration 获取 Refresh Token 过期时间
func GetRefreshExpiration() time.Duration {
	return config.refreshExpiration
}

// GetSkipPaths 获取跳过认证的路径列表
func GetSkipPaths() []string {
	return append([]string{}, config.skipPaths...) // 返回副本
}

// IsPathSkipped 检查指定路径是否被跳过认证
func IsPathSkipped(path string) bool {
	return shouldSkipPath(path)
}

// ParseWithKey 使用自定义密钥解析 token
func ParseWithKey(tokenString, key string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, ErrEmptyToken
	}

	if key == "" {
		return nil, jwt.ErrInvalidKey
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidTokenClaims
	}

	return claims, nil
}

// ParseRefreshToken 解析并验证 Refresh Token（验证 token_type="refresh"）
func ParseRefreshToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", ErrEmptyToken
	}

	// 验证 token 类型
	tokenType, err := GetTokenType(tokenString)
	if err != nil {
		return "", err
	}
	if tokenType != TokenTypeRefresh {
		return "", ErrNotRefreshToken
	}

	return ParseIdentity(tokenString, config.key)
}

// GetTokenType 获取 token 类型
func GetTokenType(tokenString string) (string, error) {
	if tokenString == "" {
		return "", ErrEmptyToken
	}

	claims, err := GetClaims(tokenString)
	if err != nil {
		return "", err
	}

	tokenType, exists := claims["token_type"]
	if !exists {
		return "", ErrMissingTokenType
	}

	typeStr, ok := tokenType.(string)
	if !ok {
		return "", ErrInvalidTokenType
	}

	return typeStr, nil
}

// IsAccessToken 检查是否为 Access Token
func IsAccessToken(tokenString string) bool {
	tokenType, err := GetTokenType(tokenString)
	if err != nil {
		return false
	}
	return tokenType == TokenTypeAccess
}

// IsRefreshToken 检查是否为 Refresh Token
func IsRefreshToken(tokenString string) bool {
	tokenType, err := GetTokenType(tokenString)
	if err != nil {
		return false
	}
	return tokenType == TokenTypeRefresh
}

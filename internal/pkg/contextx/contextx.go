package contextx

import (
	"context"
)

// 定义 context 的键。
type (
	// usernameKey 定义用户名的 context 键。
	usernameKey struct{}
	// userIDKey 定义用户 ID 的 context 键。
	userIDKey struct{}
	// accessTokenKey 定义访问令牌的 context 键。
	accessTokenKey struct{}
	// tokenVersionKey 定义令牌版本的 context 键。
	tokenVersionKey struct{}
	// requestIDKey 定义请求 ID 的 context 键。
	requestIDKey struct{}
	// traceIDKey 是用于在 context 中存储追踪 ID 的键
	traceIDKey struct{}
	// clientAddressKey 保存未信任代理头参与前的直连客户端地址。
	clientAddressKey struct{}
	// userAgentKey 保存登录请求 User-Agent，用于安全审计。
	userAgentKey struct{}
)

// WithUserID 将用户 ID 存储到 context 中。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID 从 context 中检索用户 ID。
func UserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey{}).(string)
	return userID
}

// WithUsername 将用户名存储到 context 中。
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username 从 context 中检索用户名。
func Username(ctx context.Context) string {
	username, _ := ctx.Value(usernameKey{}).(string)
	return username
}

// WithAccessToken 将访问令牌存储到 context 中。
func WithAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, accessToken)
}

// AccessToken 从 context 中检索访问令牌。
func AccessToken(ctx context.Context) string {
	accessToken, _ := ctx.Value(accessTokenKey{}).(string)
	return accessToken
}

// WithTokenVersion 将已验证的令牌版本存储到 context 中。
func WithTokenVersion(ctx context.Context, version int64) context.Context {
	return context.WithValue(ctx, tokenVersionKey{}, version)
}

// TokenVersion 从 context 中检索已验证的令牌版本。
func TokenVersion(ctx context.Context) int64 {
	version, _ := ctx.Value(tokenVersionKey{}).(int64)
	return version
}

// WithRequestID 将请求 ID 存储到 context 中。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID 从 context 中检索请求 ID。
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// WithTraceID 将追踪 ID 存储到 context 中。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceID 从 context 中检索追踪 ID。
func TraceID(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}

// WithClientAddress 将客户端网络地址存储到 context 中。
func WithClientAddress(ctx context.Context, address string) context.Context {
	return context.WithValue(ctx, clientAddressKey{}, address)
}

// ClientAddress 从 context 中检索客户端网络地址。
func ClientAddress(ctx context.Context) string {
	address, _ := ctx.Value(clientAddressKey{}).(string)
	return address
}

// WithUserAgent 将 User-Agent 存储到 context 中。
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, userAgentKey{}, userAgent)
}

// UserAgent 从 context 中检索 User-Agent。
func UserAgent(ctx context.Context) string {
	userAgent, _ := ctx.Value(userAgentKey{}).(string)
	return userAgent
}

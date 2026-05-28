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
	// requestIDKey 定义请求 ID 的 context 键。
	requestIDKey struct{}
	// traceIDKey 是用于在 context 中存储追踪 ID 的键
	traceIDKey struct{}
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

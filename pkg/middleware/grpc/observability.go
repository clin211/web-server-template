package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// --- 追踪头常量 ---
const (
	TraceParentHeaderKey = "traceparent"
	TraceIDHeaderKey     = "X-Trace-Id"
	RequestIDHeaderKey   = "X-Request-Id"
	TraceStateHeaderKey  = "tracestate"
)

// TraceInjectionMode 定义 gRPC 元数据的追踪注入方式。
type TraceInjectionMode int

const (
	InjectW3CTraceContext TraceInjectionMode = iota
	InjectTraceIDOnly
	InjectBoth
	InjectNone
)

// ObservabilityOptions 配置 gRPC 可观测性。
type ObservabilityOptions struct {
	TraceInjectionMode TraceInjectionMode
	CustomTraceHeader  string
}

// Option 修改 gRPC 可观测性配置。
type Option func(*ObservabilityOptions)

func WithTraceInjection(mode TraceInjectionMode) Option {
	return func(o *ObservabilityOptions) { o.TraceInjectionMode = mode }
}

func WithCustomTraceHeader(header string) Option {
	return func(o *ObservabilityOptions) { o.CustomTraceHeader = header }
}

// Observability 返回记录 RPC 请求和追踪信息的服务端拦截器。
func Observability(opts ...Option) grpc.UnaryServerInterceptor {
	cfg := &ObservabilityOptions{TraceInjectionMode: InjectTraceIDOnly}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		spanCtx := trace.SpanFromContext(ctx).SpanContext()

		// 注入追踪头
		md, _ := metadata.FromIncomingContext(ctx)
		md = md.Copy()
		md = injectTraceMetadata(ctx, md, spanCtx, cfg)
		ctx = metadata.NewIncomingContext(ctx, md)
		isDebugLevel := isDebugEnabled(ctx)

		requestBody := debugBody(ctx, req, isDebugLevel)
		resp, err := handler(ctx, req)
		responseBody := debugBody(ctx, resp, isDebugLevel)

		duration := time.Since(start).Seconds()

		// 对端信息
		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		status := "OK"
		if err != nil {
			status = "ERROR"
		}

		rpcData := rpcLogData(info.FullMethod, status, isDebugLevel, requestBody, responseBody)

		logLevel := slog.LevelInfo
		if isDebugLevel {
			logLevel = slog.LevelDebug
		}

		slog.Log(ctx, logLevel, "gRPC request completed",
			"duration_sec", duration,
			"source", map[string]any{"ip": clientIP},
			"rpc", rpcData,
			"trace", map[string]any{"id": spanCtx.TraceID().String()},
			"span", map[string]any{"id": spanCtx.SpanID().String()},
		)

		return resp, err
	}
}

// --- 注入追踪元数据 ---
func injectTraceMetadata(ctx context.Context, md metadata.MD, spanCtx trace.SpanContext, cfg *ObservabilityOptions) metadata.MD {
	if !spanCtx.IsValid() {
		return md
	}

	switch cfg.TraceInjectionMode {
	case InjectW3CTraceContext:
		md.Set(TraceParentHeaderKey, traceParent(spanCtx))
	case InjectTraceIDOnly:
		setTraceID(md, spanCtx, cfg.CustomTraceHeader)
	case InjectBoth:
		md.Set(TraceParentHeaderKey, traceParent(spanCtx))
		setTraceID(md, spanCtx, cfg.CustomTraceHeader)
	case InjectNone:
	}

	// 将请求 ID 设置到响应的 Header Metadata 中
	// grpc.SetHeader 会在 gRPC 方法响应中添加元数据（Metadata），
	// 此处将包含请求 ID 的 Metadata 设置到 Header 中。
	// 注意：grpc.SetHeader 仅设置数据，它不会立即发送给客户端。
	// Header Metadata 会在 RPC 响应返回时一并发送。
	if err := grpc.SetHeader(ctx, md); err != nil {
		slog.DebugContext(ctx, "Cannot set gRPC trace headers", "err", err)
	}

	return md
}

func isDebugEnabled(ctx context.Context) bool {
	return slog.Default().Enabled(ctx, slog.LevelDebug)
}

// ObservabilityWithW3CTraceContext 使用 W3C 追踪头创建拦截器。
func ObservabilityWithW3CTraceContext() grpc.UnaryServerInterceptor {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

func ObservabilityWithTraceID() grpc.UnaryServerInterceptor {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

func ObservabilityWithCustomHeader(headerName string) grpc.UnaryServerInterceptor {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}

func debugBody(ctx context.Context, value any, enabled bool) string {
	if !enabled || value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		slog.DebugContext(ctx, "Cannot serialize gRPC payload for logging", "err", err)
		return ""
	}
	return string(data)
}

func rpcLogData(method, status string, debug bool, requestBody, responseBody string) map[string]any {
	request := map[string]any{"method": method}
	response := map[string]any{"status": status}
	if debug {
		request["body"] = map[string]any{"content": requestBody, "bytes": len(requestBody)}
		response["body"] = map[string]any{"content": responseBody, "bytes": len(responseBody)}
	}
	return map[string]any{"request": request, "response": response}
}

func traceParent(spanCtx trace.SpanContext) string {
	flags := "00"
	if spanCtx.IsSampled() {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", spanCtx.TraceID().String(), spanCtx.SpanID().String(), flags)
}
func setTraceID(md metadata.MD, spanCtx trace.SpanContext, header string) {
	if header == "" {
		header = TraceIDHeaderKey
	}
	md.Set(header, spanCtx.TraceID().String())
}

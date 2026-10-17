package grpc

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

func TestTraceMetadataModes(t *testing.T) {
	span := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
	})
	for _, tt := range []struct {
		name       string
		mode       TraceInjectionMode
		parent, id bool
	}{
		{"w3c", InjectW3CTraceContext, true, false},
		{"id", InjectTraceIDOnly, false, true},
		{"both", InjectBoth, true, true},
		{"none", InjectNone, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			md := injectTraceMetadata(t.Context(), metadata.MD{}, span, &ObservabilityOptions{TraceInjectionMode: tt.mode, CustomTraceHeader: "custom-trace"})
			if got := len(md.Get(TraceParentHeaderKey)) != 0; got != tt.parent {
				t.Errorf("traceparent present = %v, want %v", got, tt.parent)
			}
			if got := len(md.Get("custom-trace")) != 0; got != tt.id {
				t.Errorf("custom trace ID present = %v, want %v", got, tt.id)
			}
			if tt.parent && md.Get(TraceParentHeaderKey)[0] != "00-"+span.TraceID().String()+"-"+span.SpanID().String()+"-01" {
				t.Errorf("unexpected traceparent: %v", md.Get(TraceParentHeaderKey))
			}
		})
	}
}

func TestRPCLogDataDebugRetainsMethodAndStatus(t *testing.T) {
	got := rpcLogData("/example.Service/Method", "ERROR", true, "request", "response")
	want := map[string]any{
		"request":  map[string]any{"method": "/example.Service/Method", "body": map[string]any{"content": "request", "bytes": 7}},
		"response": map[string]any{"status": "ERROR", "body": map[string]any{"content": "response", "bytes": 8}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RPC log data = %#v, want %#v", got, want)
	}
}

func TestDebugEnabledUsesRequestContext(t *testing.T) {
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	ctx := context.WithValue(t.Context(), contextKey{}, true)
	slog.SetDefault(slog.New(contextLevelHandler{Handler: slog.NewTextHandler(io.Discard, nil)}))
	if !isDebugEnabled(ctx) {
		t.Fatal("request-specific debug level was ignored")
	}
	if isDebugEnabled(t.Context()) {
		t.Fatal("debug unexpectedly enabled without request context")
	}
}

type contextKey struct{}
type contextLevelHandler struct{ slog.Handler }

func (h contextLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	enabled, ok := ctx.Value(contextKey{}).(bool)
	return ok && enabled
}

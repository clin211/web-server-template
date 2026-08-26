package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
)

func TestContextHandlerAddsCorrelationFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		want map[string]string
	}{
		{
			name: "request and user context",
			ctx: contextx.WithUserID(
				contextx.WithRequestID(context.Background(), "request-1"),
				"user-1",
			),
			want: map[string]string{"request_id": "request-1", "user_id": "user-1"},
		},
		{
			name: "otel span context",
			ctx: trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
				TraceID: trace.TraceID{1},
				SpanID:  trace.SpanID{2},
			})),
			want: map[string]string{
				"trace_id": "01000000000000000000000000000000",
				"span_id":  "0200000000000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			logger := slog.New(NewContextHandler(slog.NewJSONHandler(&output, nil)))
			logger.InfoContext(tt.ctx, "test event")

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode log record: %v", err)
			}
			for key, want := range tt.want {
				if got := record[key]; got != want {
					t.Errorf("field %s = %v, want %q", key, got, want)
				}
			}
		})
	}
}

func TestContextHandlerPreservesLoggerAttributes(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(&output, nil))).With("component", "worker")
	logger.InfoContext(context.Background(), "test event")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	if got := record["component"]; got != "worker" {
		t.Errorf("component = %v, want worker", got)
	}
}

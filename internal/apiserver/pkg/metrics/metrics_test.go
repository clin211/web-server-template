package metrics

import (
	"context"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAsyncMetricsRecordsLowCardinalitySignals(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics, err := NewAsyncMetrics(meterProvider.Meter("test"))
	if err != nil {
		t.Fatalf("NewAsyncMetrics() error = %v", err)
	}
	metrics.RecordPublish(context.Background(), "example.created.v1", "success", 10*time.Millisecond)
	metrics.RecordProcess(context.Background(), "example.created.v1", "failure", 20*time.Millisecond)
	metrics.RecordDLQ(context.Background(), "example.created.v1", "success", time.Unix(100, 0))
	if err := reader.Collect(context.Background(), &metricdata.ResourceMetrics{}); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
}

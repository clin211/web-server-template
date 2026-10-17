package store_test

import (
	"context"
	"testing"
	"time"

	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/metrics"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestIntegrationQueueMetricsCacheAndLabels(t *testing.T) {
	db, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	if _, err := taskbiz.New(core).Submit(ctx, job.ReportParams{Days: 1}, "metric"); err != nil {
		t.Fatal(err)
	}
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	queue, err := metrics.NewQueueMetrics(db, provider.Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	queue.RecordExecution(ctx, "succeeded", time.Second)
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := queue.Refresh(ctx); err == nil {
		t.Fatal("database failure ignored")
	}
	var resource metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &resource); err != nil {
		t.Fatal("cached scrape queried closed database", err)
	}
	found := false
	for _, scope := range resource.ScopeMetrics {
		for _, metric := range scope.Metrics {
			gauge, ok := metric.Data.(metricdata.Gauge[float64])
			if !ok {
				continue
			}
			for _, point := range gauge.DataPoints {
				for _, attr := range point.Attributes.ToSlice() {
					if attr.Key == "task_id" || attr.Key == "event_id" || attr.Key == "owner_id" || attr.Key == "worker_id" {
						t.Fatal("unbounded metric label", attr.Key)
					}
				}
				if metric.Name == "task_queue_count" {
					found = true
					if point.Value != 1 {
						t.Fatal("lost last good snapshot", point.Value)
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("queue gauge missing")
	}
}

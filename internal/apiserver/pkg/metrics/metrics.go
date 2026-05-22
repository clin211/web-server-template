package metrics

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	Meter                     metric.Meter
	RESTResourceCreateCounter metric.Int64Counter
	RESTResourceGetCounter    metric.Int64Counter
}

var M *Metrics

// Initialize 初始化 Prometheus 导出器和自定义业务指标。
func Initialize(ctx context.Context, scope string) error {
	metricPrefix := sanitizeMetricPrefix(scope)
	meter := otel.Meter(scope + ".metrics")

	createCounter, _ := meter.Int64Counter(
		fmt.Sprintf("%s_resource_create_total", metricPrefix),
		metric.WithDescription("Total number of REST resource create requests"),
	)
	getCount, _ := meter.Int64Counter(
		fmt.Sprintf("%s_resource_get_total", metricPrefix),
		metric.WithDescription("Total number of REST resource get requests"),
	)

	M = &Metrics{
		Meter:                     meter,
		RESTResourceCreateCounter: createCounter,
		RESTResourceGetCounter:    getCount,
	}

	return nil
}

func sanitizeMetricPrefix(scope string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_")
	prefix := replacer.Replace(scope)
	prefix = strings.Trim(prefix, "_")
	if prefix == "" {
		return "apiserver"
	}
	return prefix
}

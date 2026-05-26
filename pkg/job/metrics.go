package job

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics 封装任务队列和定时调度相关的 OpenTelemetry 指标。
type Metrics struct {
	enqueuedTotal       metric.Int64Counter
	processedTotal      metric.Int64Counter
	processDuration     metric.Float64Histogram
	schedulerTicksTotal metric.Int64Counter
	schedulerReconcile  metric.Int64Counter
	registeredTaskGauge metric.Int64UpDownCounter
	deadTotal           metric.Int64Counter
}

// NewMetrics 创建任务模块使用的指标集合。
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("github.com/clin211/gin-enterprise-template/pkg/job")

	enqueuedTotal, err := meter.Int64Counter("job_tasks_enqueued_total")
	if err != nil {
		return nil, fmt.Errorf("create job_tasks_enqueued_total metric: %w", err)
	}
	processedTotal, err := meter.Int64Counter("job_tasks_processed_total")
	if err != nil {
		return nil, fmt.Errorf("create job_tasks_processed_total metric: %w", err)
	}
	processDuration, err := meter.Float64Histogram("job_task_duration_seconds")
	if err != nil {
		return nil, fmt.Errorf("create job_task_duration_seconds metric: %w", err)
	}
	schedulerTicksTotal, err := meter.Int64Counter("job_scheduler_ticks_total")
	if err != nil {
		return nil, fmt.Errorf("create job_scheduler_ticks_total metric: %w", err)
	}
	schedulerReconcile, err := meter.Int64Counter("job_scheduler_reconcile_total")
	if err != nil {
		return nil, fmt.Errorf("create job_scheduler_reconcile_total metric: %w", err)
	}
	registeredTaskGauge, err := meter.Int64UpDownCounter("job_scheduler_registered_tasks")
	if err != nil {
		return nil, fmt.Errorf("create job_scheduler_registered_tasks metric: %w", err)
	}
	deadTotal, err := meter.Int64Counter("job_task_dead_total")
	if err != nil {
		return nil, fmt.Errorf("create job_task_dead_total metric: %w", err)
	}

	return &Metrics{
		enqueuedTotal:       enqueuedTotal,
		processedTotal:      processedTotal,
		processDuration:     processDuration,
		schedulerTicksTotal: schedulerTicksTotal,
		schedulerReconcile:  schedulerReconcile,
		registeredTaskGauge: registeredTaskGauge,
		deadTotal:           deadTotal,
	}, nil
}

// RecordEnqueue 记录任务入队结果。
func (m *Metrics) RecordEnqueue(ctx context.Context, taskType string, queue string, result string) {
	if m == nil {
		return
	}
	m.enqueuedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("taskType", taskType),
		attribute.String("queue", queue),
		attribute.String("result", result),
	))
}

// RecordProcessed 记录任务处理状态和耗时。
func (m *Metrics) RecordProcessed(ctx context.Context, taskType string, queue string, status string, duration time.Duration) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("taskType", taskType),
		attribute.String("queue", queue),
		attribute.String("status", status),
	)
	m.processedTotal.Add(ctx, 1, attrs)
	m.processDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordSchedulerTick 记录定时调度单次触发结果。
func (m *Metrics) RecordSchedulerTick(ctx context.Context, taskType string, result string) {
	if m == nil {
		return
	}
	m.schedulerTicksTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("taskType", taskType),
		attribute.String("result", result),
	))
}

// RecordSchedulerReconcile 记录客户端定时任务同步结果。
func (m *Metrics) RecordSchedulerReconcile(ctx context.Context, result string) {
	if m == nil {
		return
	}
	m.schedulerReconcile.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

// AddRegisteredTasks 增减指定来源的已注册定时任务数量。
func (m *Metrics) AddRegisteredTasks(ctx context.Context, source string, delta int64) {
	if m == nil {
		return
	}
	m.registeredTaskGauge.Add(ctx, delta, metric.WithAttributes(attribute.String("source", source)))
}

// RecordDead 记录进入死信状态的任务。
func (m *Metrics) RecordDead(ctx context.Context, taskType string, queue string) {
	if m == nil {
		return
	}
	m.deadTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("taskType", taskType),
		attribute.String("queue", queue),
	))
}

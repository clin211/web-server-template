package metrics

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
	"sync/atomic"
	"time"
)

// QueueMetrics caches authoritative database gauges so scrapes never run SQL.
type QueueMetrics struct {
	db        *gorm.DB
	points    atomic.Pointer[[]gaugePoint]
	gauges    map[string]metric.Float64ObservableGauge
	duration  metric.Float64Histogram
	outcomes  metric.Int64Counter
	leaseLoss metric.Int64Counter
}
type gaugePoint struct {
	name  string
	value float64
	attrs []attribute.KeyValue
}

// NewQueueMetrics creates bounded-label instruments and a cache-only callback.
func NewQueueMetrics(db *gorm.DB, meter metric.Meter) (*QueueMetrics, error) {
	m := &QueueMetrics{db: db, gauges: make(map[string]metric.Float64ObservableGauge)}
	for _, name := range []string{"task_attempt_outcomes", "dlq_replay_results", "task_state_count", "schedule_trigger_delay_seconds", "task_queue_count", "task_oldest_wait_seconds", "task_running_count", "task_lease_takeovers", "schedule_due_count", "schedule_oldest_due_seconds", "schedule_trigger_outcomes", "outbox_pending_count", "outbox_oldest_pending_seconds", "outbox_dead_count", "dlq_pending_count", "dlq_dead_count", "dlq_replay_count", "runtime_available_count", "queue_metrics_last_success_timestamp_seconds"} {
		g, err := meter.Float64ObservableGauge(name)
		if err != nil {
			return nil, err
		}
		m.gauges[name] = g
	}
	if err := m.createCounters(meter); err != nil {
		return nil, err
	}
	instruments := make([]metric.Observable, 0, len(m.gauges))
	for _, g := range m.gauges {
		instruments = append(instruments, g)
	}
	if err := m.registerCallback(meter, instruments); err != nil {
		return nil, err
	}
	return m, nil
}

// RecordExecution observes local execution outcomes; task IDs remain in logs and traces.
func (m *QueueMetrics) RecordExecution(ctx context.Context, result string, duration time.Duration) {
	m.outcomes.Add(ctx, 1, metric.WithAttributes(attribute.String("task_type", "report.generate"), attribute.String("result", result)))
	m.duration.Record(ctx, duration.Seconds(), metric.WithAttributes(attribute.String("task_type", "report.generate")))
}

// RecordLeaseLost records rejected leases without a worker or task label.
func (m *QueueMetrics) RecordLeaseLost(ctx context.Context) { m.leaseLoss.Add(ctx, 1) }

// Refresh performs bounded database aggregation and atomically replaces the last good snapshot.
func (m *QueueMetrics) Refresh(ctx context.Context) error {
	points := make([]gaugePoint, 0, 40)
	queries := []struct{ name, sql string }{
		{"task_queue_count", "SELECT count(*) FROM task WHERE status IN ('pending','retry_wait')"},
		{"task_oldest_wait_seconds", "SELECT coalesce(greatest(extract(epoch FROM clock_timestamp()-min(available_at)),0),0) FROM task WHERE status IN ('pending','retry_wait') AND available_at<=clock_timestamp()"},
		{"task_running_count", "SELECT count(*) FROM task WHERE status='running'"},
		{"task_lease_takeovers", "SELECT count(*) FROM task_attempt WHERE outcome='lease_expired'"},
		{"schedule_due_count", "SELECT count(*) FROM schedule WHERE enabled AND NOT archived AND next_run_at<=clock_timestamp()"},
		{"schedule_oldest_due_seconds", "SELECT coalesce(greatest(extract(epoch FROM clock_timestamp()-min(next_run_at)),0),0) FROM schedule WHERE enabled AND NOT archived"},
		{"outbox_pending_count", "SELECT count(*) FROM outbox_event WHERE status IN ('pending','failed','processing')"},
		{"outbox_oldest_pending_seconds", "SELECT coalesce(extract(epoch FROM clock_timestamp()-min(created_at)),0) FROM outbox_event WHERE status IN ('pending','failed','processing')"},
		{"outbox_dead_count", "SELECT count(*) FROM outbox_event WHERE status='dead'"},
		{"dlq_pending_count", "SELECT count(*) FROM dead_letter_event WHERE status IN ('pending','failed','processing')"},
		{"dlq_dead_count", "SELECT count(*) FROM dead_letter_event WHERE status='dead'"},
		{"dlq_replay_count", "SELECT count(*) FROM event_replay WHERE created_at>=clock_timestamp()-interval '1 day'"},
	}
	for _, q := range queries {
		var n float64
		if err := m.db.WithContext(ctx).Raw(q.sql).Scan(&n).Error; err != nil {
			return fmt.Errorf("refresh %s: %w", q.name, err)
		}
		points = append(points, gaugePoint{name: q.name, value: n})
	}

	more, err := m.statePoints(ctx)
	if err != nil {
		return err
	}
	points = append(points, more...)
	var latency float64
	if err := m.db.WithContext(ctx).Raw("SELECT coalesce(max(extract(epoch FROM created_at-scheduled_at)),0) FROM schedule_trigger WHERE outcome='created' AND created_at>=clock_timestamp()-interval '1 hour'").Scan(&latency).Error; err != nil {
		return err
	}
	points = append(points, gaugePoint{name: "schedule_trigger_delay_seconds", value: latency})
	more, err = m.rolePoints(ctx)
	if err != nil {
		return err
	}
	points = append(points, more...)
	more, err = m.outcomePoints(ctx)
	if err != nil {
		return err
	}
	points = append(points, more...)

	points = append(points, gaugePoint{name: "queue_metrics_last_success_timestamp_seconds", value: float64(time.Now().Unix())})
	m.points.Store(&points)
	return nil
}

func (m *QueueMetrics) createCounters(meter metric.Meter) error {
	var err error
	m.duration, err = meter.Float64Histogram("task_execution_duration_seconds", metric.WithUnit("s"))
	if err != nil {
		return err
	}
	m.outcomes, err = meter.Int64Counter("task_execution_total")
	if err != nil {
		return err
	}
	m.leaseLoss, err = meter.Int64Counter("task_lease_lost_total")
	if err != nil {
		return err
	}
	return nil
}

func (m *QueueMetrics) statePoints(ctx context.Context) ([]gaugePoint, error) {
	points := make([]gaugePoint, 0, 6)
	var states []struct {
		Status string
		Count  float64
	}
	if err := m.db.WithContext(ctx).Raw("SELECT status,count(*) AS count FROM task GROUP BY status").Scan(&states).Error; err != nil {
		return nil, err
	}
	for _, state := range []string{"pending", "running", "retry_wait", "succeeded", "failed", "canceled"} {
		n := float64(0)
		for _, r := range states {
			if r.Status == state {
				n = r.Count
			}
		}
		points = append(points, gaugePoint{"task_state_count", n, []attribute.KeyValue{attribute.String("status", state)}})
	}
	return points, nil
}

func (m *QueueMetrics) rolePoints(ctx context.Context) ([]gaugePoint, error) {
	points := make([]gaugePoint, 0, 6)
	var roles []struct {
		Role  string
		Count float64
	}
	if err := m.db.WithContext(ctx).Raw("SELECT role,count(*) AS count FROM runtime_heartbeat WHERE updated_at>clock_timestamp()-interval '10 seconds' GROUP BY role").Scan(&roles).Error; err != nil {
		return nil, err
	}
	for _, role := range []string{"worker", "scheduler"} {
		n := float64(0)
		for _, r := range roles {
			if r.Role == role {
				n = r.Count
			}
		}
		points = append(points, gaugePoint{"runtime_available_count", n, []attribute.KeyValue{attribute.String("role", role)}})
	}
	return points, nil
}

func (m *QueueMetrics) registerCallback(meter metric.Meter, instruments []metric.Observable) error {
	if _, err := meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		if p := m.points.Load(); p != nil {
			for _, v := range *p {
				o.ObserveFloat64(m.gauges[v.name], v.value, metric.WithAttributes(v.attrs...))
			}
		}
		return nil
	}, instruments...); err != nil {
		return err
	}
	return nil
}

func (m *QueueMetrics) outcomePoints(ctx context.Context) ([]gaugePoint, error) {
	queries := []struct {
		name, label, sql string
		allowed          []string
	}{
		{"schedule_trigger_outcomes", "outcome", "SELECT outcome AS name,count(*) AS count FROM schedule_trigger WHERE created_at>=clock_timestamp()-interval '1 day' GROUP BY outcome", []string{"created", "overlap_skipped", "misfire_skipped", "window_skipped", "limit_skipped"}},
		{"task_attempt_outcomes", "outcome", "SELECT coalesce(outcome,'running') AS name,count(*) AS count FROM task_attempt WHERE started_at>=clock_timestamp()-interval '1 day' GROUP BY outcome", []string{"running", "retry_wait", "succeeded", "failed", "canceled", "lease_expired"}},
		{"dlq_replay_results", "result", `SELECT name,count(*) AS count FROM (SELECT r.event_id,CASE WHEN bool_or(i.status='completed') THEN 'completed' WHEN bool_or(i.status='dead_lettered') THEN 'failed' ELSE 'pending' END AS name FROM event_replay r LEFT JOIN inbox_event i ON i.event_id=r.event_id WHERE r.created_at>=clock_timestamp()-interval '1 day' GROUP BY r.event_id) outcomes GROUP BY name`, []string{"completed", "failed", "pending"}},
	}
	points := make([]gaugePoint, 0, 14)
	for _, q := range queries {
		series, err := m.boundedCounts(ctx, q.name, q.label, q.sql, q.allowed)
		if err != nil {
			return nil, err
		}
		points = append(points, series...)
	}
	return points, nil
}
func (m *QueueMetrics) boundedCounts(ctx context.Context, name, label, sql string, allowed []string) ([]gaugePoint, error) {
	var rows []struct {
		Name  string
		Count float64
	}
	if err := m.db.WithContext(ctx).Raw(sql).Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]float64, len(rows))
	for _, row := range rows {
		counts[row.Name] = row.Count
	}
	points := make([]gaugePoint, 0, len(allowed))
	for _, value := range allowed {
		points = append(points, gaugePoint{name, counts[value], []attribute.KeyValue{attribute.String(label, value)}})
	}
	return points, nil
}

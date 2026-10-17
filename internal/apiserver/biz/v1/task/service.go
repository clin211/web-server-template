// Package task implements resource-authorized report, task and schedule use cases.
package task

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

var errInvalidReportEvent = errors.New("invalid report event")
var errReportEventDoesNotMatchCommittedResult = errors.New("report event does not match committed result")

// Service shares business behavior between API and background processes.
type Service struct {
	core  store.IStore
	repo  *store.TaskStore
	plans *store.ScheduleStore
	auth  *store.AuthorizationStore
}

// New constructs the report vertical slice.
func New(core store.IStore) *Service {
	return &Service{core, store.NewTaskStore(core), store.NewScheduleStore(core), store.NewAuthorizationStore(core)}
}

// Require validates an action and an optional target owner; denials are audited.
func (s *Service) Require(ctx context.Context, permission, owner string) error {
	actor := contextx.UserID(ctx)
	if actor == "" {
		return errno.ErrUnauthenticated
	}
	p := permission
	if owner != "" && owner != actor {
		p += ".any"
	}
	ok, err := s.auth.Allowed(ctx, actor, p)
	if err != nil {
		return errno.ErrServiceUnavailable
	}
	if !ok {
		if err := s.auth.Audit(ctx, actor, p, owner, "denied", nil); err != nil {
			return errno.ErrServiceUnavailable
		}
		return errno.ErrPermissionDenied
	}
	return nil
}

// Registry registers only complete, versioned business handlers.
func (s *Service) Registry() (*job.Registry, error) {
	r := job.NewRegistry()
	if err := job.Register(r, "report.generate", 1, job.ValidateReport, s.Generate); err != nil {
		return nil, err
	}
	r.Freeze()
	return r, nil
}

// Submit validates a bounded typed report and atomically persists its task intent.
func (s *Service) Submit(ctx context.Context, p job.ReportParams, key string) (model.TaskM, error) {
	if err := s.Require(ctx, "task.submit", ""); err != nil {
		return model.TaskM{}, err
	}
	if err := job.ValidateReport(p); err != nil {
		return model.TaskM{}, errno.ErrInvalidArgument
	}
	if key == "" || len(key) > 128 {
		return model.TaskM{}, errno.ErrInvalidArgument.WithMessage("Idempotency-Key required, at most 128 characters")
	}
	//nolint:errchkjson // Explicitly preserve the encoding error contract.
	b, err := json.Marshal(p)
	if err != nil {
		return model.TaskM{}, err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(b))
	actor := contextx.UserID(ctx)
	trace, request := contextx.TraceID(ctx), contextx.RequestID(ctx)
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	parent := carrier.Get("traceparent")
	t := model.TaskM{TaskID: "tsk_" + uuid.NewString(), TaskType: "report.generate", Params: datatypes.JSON(b), ParamsVersion: 1, Source: "api", OwnerID: actor, Status: "pending", AvailableAt: time.Now().UTC(), TimeoutSeconds: 300, MaxAttempts: 3, IdempotencyKey: &key, RequestHash: hash, TraceID: &trace, TraceParent: &parent, RequestID: &request}
	err = s.core.TX(ctx, func(tx context.Context) error {
		if err := s.repo.Create(tx, &t); err != nil {
			return err
		}
		return s.auth.Audit(tx, actor, "task.submit", t.TaskID, "success", map[string]any{"type": t.TaskType})
	})
	return t, err
}

// Get authorizes before exposing a task's parameters or history.
func (s *Service) Get(ctx context.Context, id, permission string) (model.TaskM, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return t, errno.FromStore(err)
	}
	return t, s.Require(ctx, permission, t.OwnerID)
}

// List only exposes other owners when an explicit global view grant exists.
func (s *Service) List(ctx context.Context, all bool, before int64) ([]model.TaskM, error) {
	owner := contextx.UserID(ctx)
	p := "task.view"
	if all {
		p = "task.view.any"
		owner = ""
	}
	if err := s.Require(ctx, p, ""); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, owner, before)
}

// Attempts returns the authorized task execution history.
func (s *Service) Attempts(ctx context.Context, id string) ([]model.TaskAttemptM, error) {
	if _, err := s.Get(ctx, id, "task.view"); err != nil {
		return nil, err
	}
	return s.repo.Attempts(ctx, id)
}

// Cancel records the request and its audit in the same transaction.
func (s *Service) Cancel(ctx context.Context, id string) error {
	t, err := s.Get(ctx, id, "task.cancel")
	if err != nil {
		return err
	}
	if t.Status == "succeeded" || t.Status == "failed" {
		return store.ErrIdempotencyConflict
	}
	return s.core.TX(ctx, func(tx context.Context) error {
		if err := s.repo.Cancel(tx, id); err != nil {
			return err
		}
		return s.auth.Audit(tx, contextx.UserID(ctx), "task.cancel", id, "success", nil)
	})
}

// Retry preserves the failed instance and creates an independently auditable attempt budget.
func (s *Service) Retry(ctx context.Context, id, key string) (model.TaskM, error) {
	old, err := s.Get(ctx, id, "task.retry")
	if err != nil {
		return old, err
	}
	if old.Status != "failed" && old.Status != "canceled" {
		return old, store.ErrIdempotencyConflict
	}
	if key == "" || len(key) > 128 {
		return old, errno.ErrInvalidArgument
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte("retry:"+id)))
	var t model.TaskM
	err = s.core.TX(ctx, func(tx context.Context) error {
		var err error
		t, err = s.repo.Retry(tx, old, key, hash)
		if err != nil {
			return err
		}
		return s.auth.Audit(tx, contextx.UserID(ctx), "task.retry", id, "success", map[string]any{"new_task_id": t.TaskID})
	})
	return t, err
}

// Download performs live object-level authorization for every access.
func (s *Service) Download(ctx context.Context, id string) (model.ReportResultM, error) {
	t, err := s.Get(ctx, id, "result.download")
	if err != nil {
		return model.ReportResultM{}, err
	}
	if t.Status != "succeeded" {
		return model.ReportResultM{}, store.ErrIdempotencyConflict
	}
	r, err := s.repo.Result(ctx, id)
	return r, errno.FromStore(err)
}

// Generate reads only the task owner's safe columns and bounded task statistics.
func (s *Service) Generate(ctx context.Context, p job.ReportParams) (job.Result, error) {
	if err := s.Require(ctx, "task.submit", ""); err != nil {
		if errors.Is(err, errno.ErrPermissionDenied) || errors.Is(err, errno.ErrUnauthenticated) {
			return job.Result{}, job.Permanent(err)
		}
		return job.Result{}, err
	}
	owner := contextx.UserID(ctx)
	db := s.core.DB(ctx).WithContext(ctx)
	var profile struct {
		UserID, Username, Nickname string
		CreatedAt                  time.Time
	}
	if err := db.Table("user").Select("user_id,username,nickname,created_at").Where("user_id=? AND status=0", owner).Take(&profile).Error; err != nil {
		return job.Result{}, fmt.Errorf("report profile: %w", err)
	}
	var stats []struct {
		Status string
		Count  int64
	}
	if err := db.Table("task").Select("status,count(*) AS count").Where("owner_id=? AND created_at>=clock_timestamp()-(? * interval '1 day')", owner, p.Days).Group("status").Order("status").Scan(&stats).Error; err != nil {
		return job.Result{}, fmt.Errorf("report statistics: %w", err)
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	rows := [][]string{{"field", "value"}, {"user_id", profile.UserID}, {"username", safeCSV(profile.Username)}, {"nickname", safeCSV(profile.Nickname)}, {"created_at", profile.CreatedAt.UTC().Format(time.RFC3339)}, {"period_days", strconv.Itoa(p.Days)}}
	for _, v := range stats {
		rows = append(rows, []string{"tasks_" + v.Status, strconv.FormatInt(v.Count, 10)})
	}
	if err := w.WriteAll(rows); err != nil {
		return job.Result{}, fmt.Errorf("write report: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return job.Result{}, err
	}
	return job.Result{Content: b.Bytes(), ContentType: "text/csv; charset=utf-8"}, nil
}
func safeCSV(s string) string {
	trim := strings.TrimLeft(s, " \t\r\n")
	if strings.HasPrefix(trim, "=") || strings.HasPrefix(trim, "+") || strings.HasPrefix(trim, "-") || strings.HasPrefix(trim, "@") {
		return "'" + s
	}
	return s
}

// ScheduleInput is the versioned report plan contract.
type ScheduleInput struct {
	Name              string `json:"name"`
	Days              int    `json:"days"`
	Cron              string `json:"cron"`
	Timezone          string `json:"timezone"`
	MisfirePolicy     string `json:"misfire_policy"`
	ConcurrencyPolicy string `json:"concurrency_policy"`
	Version           int64  `json:"version"`
}

// SaveSchedule creates or updates a plan; updates require the observed configuration version.
func (s *Service) SaveSchedule(ctx context.Context, id string, p ScheduleInput) (model.ScheduleM, error) {
	var m model.ScheduleM
	var err error
	if id != "" {
		m, err = s.plans.Get(ctx, id)
		if err != nil {
			return m, errno.FromStore(err)
		}
	} else {
		m = model.ScheduleM{ScheduleID: "sch_" + uuid.NewString(), OwnerID: contextx.UserID(ctx), Enabled: true, Version: 1, TaskType: "report.generate", ParamsVersion: 1}
	}
	if err := s.Require(ctx, "schedule.manage", m.OwnerID); err != nil {
		return m, err
	}
	if err := s.Require(contextx.WithUserID(ctx, m.OwnerID), "task.submit", ""); err != nil {
		return m, err
	}
	if err := configureSchedule(&m, p); err != nil {
		return m, err
	}
	err = s.persistSchedule(ctx, id, p.Version, &m)
	if err == nil && id != "" {
		m.Version++
	}
	return m, err
}

// Schedule retrieves an authorized schedule.
func (s *Service) Schedule(ctx context.Context, id string) (model.ScheduleM, error) {
	m, err := s.plans.Get(ctx, id)
	if err != nil {
		return m, errno.FromStore(err)
	}
	return m, s.Require(ctx, "schedule.manage", m.OwnerID)
}

// Schedules returns bounded owner-filtered plans.
func (s *Service) Schedules(ctx context.Context, all bool) ([]model.ScheduleM, error) {
	owner := contextx.UserID(ctx)
	permission := "schedule.manage"
	if all {
		owner = ""
		permission = "schedule.manage.any"
	}
	if err := s.Require(ctx, permission, ""); err != nil {
		return nil, err
	}
	return s.plans.List(ctx, owner)
}

// ScheduleAction pauses, resumes or archives a plan, without changing existing tasks.
func (s *Service) ScheduleAction(ctx context.Context, id, action string, version int64) error {
	m, err := s.Schedule(ctx, id)
	if err != nil {
		return err
	}
	switch action {
	case "pause":
		m.Enabled = false
	case "resume":
		m.Enabled = true
	case "archive":
		m.Enabled = false
		m.Archived = true
	default:
		return errno.ErrInvalidArgument
	}
	spec, err := job.ParseSchedule(m.Cron, m.Timezone)
	if err != nil {
		return err
	}
	m.NextRunAt = spec.Next(time.Now().UTC()).UTC()
	return s.core.TX(ctx, func(tx context.Context) error {
		if err := s.plans.Update(tx, &m, version); err != nil {
			return err
		}
		return s.auth.Audit(tx, contextx.UserID(ctx), "schedule."+action, id, "success", nil)
	})
}

// ScheduleHistory returns durable trigger outcomes.
func (s *Service) ScheduleHistory(ctx context.Context, id string) ([]map[string]any, error) {
	if _, err := s.Schedule(ctx, id); err != nil {
		return nil, err
	}
	return s.plans.History(ctx, id)
}

// EventRouter installs the concrete report availability consumer.
func (s *Service) EventRouter() (*consumer.EventRouter, error) {
	r := consumer.NewEventRouter()
	if err := r.Register("report.generated.v1", consumer.HandlerFunc(s.ReportGenerated)); err != nil {
		return nil, err
	}
	return r, nil
}

// ReportGenerated writes a durable user notification in the caller's Inbox transaction.
func (s *Service) ReportGenerated(ctx context.Context, e outbox.Envelope) error {
	var p struct {
		TaskID   string `json:"task_id"`
		ResultID string `json:"result_id"`
		OwnerID  string `json:"owner_id"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return consumer.Permanent(err)
	}
	if p.TaskID == "" || p.ResultID == "" || p.OwnerID == "" || p.TaskID != e.AggregateID {
		return consumer.Permanent(errInvalidReportEvent)
	}
	var n int64
	if err := s.core.DB(ctx).WithContext(ctx).Table("task").Where("task_id=? AND owner_id=? AND result_id=? AND status='succeeded'", p.TaskID, p.OwnerID, p.ResultID).Count(&n).Error; err != nil {
		return err
	}
	if n != 1 {
		return consumer.Permanent(errReportEventDoesNotMatchCommittedResult)
	}
	return s.core.DB(ctx).WithContext(ctx).Exec("INSERT INTO report_notification(task_id,owner_id,result_id) VALUES(?,?,?) ON CONFLICT DO NOTHING", p.TaskID, p.OwnerID, p.ResultID).Error
}

// Notifications returns the caller's durable availability feed.
func (s *Service) Notifications(ctx context.Context) ([]map[string]any, error) {
	if err := s.Require(ctx, "task.view", ""); err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0)
	err := s.core.DB(ctx).WithContext(ctx).Table("report_notification").Where("owner_id=?", contextx.UserID(ctx)).Order("created_at DESC").Limit(100).Find(&rows).Error
	return rows, err
}

// RoleMutation applies explicit RBAC administration.
func (s *Service) RoleMutation(ctx context.Context, action, id, target string, permissions []string) error {
	if err := s.Require(ctx, "rbac.manage", ""); err != nil {
		return err
	}
	if len(id) == 0 || len(id) > 64 || len(target) > 80 || len(permissions) > 32 {
		return errno.ErrInvalidArgument
	}
	err := s.auth.ManageRole(ctx, contextx.UserID(ctx), action, id, target, permissions)
	return s.auditFailure(ctx, "rbac."+action, id, err)
}

// AdminList exposes only explicitly selected safe fields of fixed management tables.
func (s *Service) AdminList(ctx context.Context, kind string) ([]map[string]any, error) {
	var table, fields, permission string
	switch kind {
	case "roles":
		table = "role"
		fields = "role_id,name,builtin"
		permission = "rbac.manage"
	case "permissions":
		table = "permission"
		fields = "code"
		permission = "rbac.manage"
	case "audits":
		table = "audit_log"
		fields = "user_id,action,resource,outcome,details,created_at,request_id"
		permission = "audit.view"
	case "dead-letters":
		table = "dead_letter_event"
		fields = "dead_letter_id,original_event_id,original_event_type,failure_kind,consume_attempts,status,created_at"
		permission = "dlq.view"
	default:
		return nil, errno.ErrInvalidArgument
	}
	if err := s.Require(ctx, permission, ""); err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0)
	err := s.core.DB(ctx).WithContext(ctx).Table(table).Select(fields).Limit(100).Find(&rows).Error
	return rows, err
}

// Replay creates an audited delivery identity without resetting original Inbox history.
func (s *Service) Replay(ctx context.Context, id, key string) (string, error) {
	if err := s.Require(ctx, "dlq.replay", ""); err != nil {
		return "", err
	}
	if key == "" || len(key) > 128 {
		return "", errno.ErrInvalidArgument
	}
	actor := contextx.UserID(ctx)
	eventID := "evt_" + uuid.NewString()
	err := s.core.TX(ctx, func(tx context.Context) error {
		resolved, err := s.prepareReplay(tx, id, eventID, actor, key)
		eventID = resolved
		return err
	})
	return eventID, s.auditFailure(ctx, "dlq.replay", id, err)
}
func existingReplay(db *gorm.DB, actor, key, id string) (string, error) {
	var old struct{ EventID, DeadLetterID string }
	r := db.Table("event_replay").Where("actor_id=? AND idempotency_key=?", actor, key).Scan(&old)
	if r.Error != nil {
		return "", r.Error
	}
	if r.RowsAffected == 0 {
		return "", nil
	}
	if old.DeadLetterID != id {
		return "", store.ErrIdempotencyConflict
	}
	return old.EventID, nil
}
func (s *Service) replayEnvelope(ctx context.Context, id string) (outbox.Envelope, error) {
	db := s.core.DB(ctx).WithContext(ctx)
	var dead model.DeadLetterEventM
	if err := db.Where("dead_letter_id=?", id).Take(&dead).Error; err != nil {
		return outbox.Envelope{}, errno.FromStore(err)
	}
	if dead.CreatedAt.Before(time.Now().Add(-90 * 24 * time.Hour)) {
		return outbox.Envelope{}, errno.ErrInvalidArgument.WithMessage("replay window expired")
	}
	e, err := outbox.ParseEnvelope(dead.Payload)
	if err != nil {
		return outbox.Envelope{}, errno.ErrInvalidArgument.WithMessage("invalid original envelope")
	}
	if e.EventType != "report.generated.v1" {
		return outbox.Envelope{}, errno.ErrInvalidArgument.WithMessage("unsupported replay type")
	}
	var p struct {
		TaskID   string `json:"task_id"`
		ResultID string `json:"result_id"`
		OwnerID  string `json:"owner_id"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return outbox.Envelope{}, err
	}
	if p.TaskID == "" || p.ResultID == "" || p.OwnerID == "" || p.TaskID != e.AggregateID {
		return outbox.Envelope{}, errno.ErrInvalidArgument
	}
	return e, nil
}

// DeadLetter returns diagnostic metadata without raw message bytes.
func (s *Service) DeadLetter(ctx context.Context, id string) (map[string]any, error) {
	if err := s.Require(ctx, "dlq.view", ""); err != nil {
		return nil, err
	}
	var row map[string]any
	r := s.core.DB(ctx).WithContext(ctx).Table("dead_letter_event").Select("dead_letter_id,consumer_name,original_event_id,original_event_type,source_topic,source_partition,source_offset,failure_kind,consume_attempts,publish_attempts,status,created_at").Where("dead_letter_id=?", id).Take(&row)
	if r.Error != nil {
		return nil, errno.FromStore(r.Error)
	}
	return row, nil
}
func configureSchedule(m *model.ScheduleM, p ScheduleInput) error {
	if p.Timezone == "" {
		p.Timezone = "UTC"
	}
	if p.MisfirePolicy == "" {
		p.MisfirePolicy = "skip"
	}
	if p.ConcurrencyPolicy == "" {
		p.ConcurrencyPolicy = "forbid"
	}
	if err := validateScheduleInput(p); err != nil {
		return err
	}
	spec, err := job.ParseSchedule(p.Cron, p.Timezone)
	if err != nil {
		return errno.ErrInvalidArgument
	}
	//nolint:errchkjson // Explicit encoding errors remain part of the business contract.
	b, err := json.Marshal(job.ReportParams{Days: p.Days})
	if err != nil {
		return err
	}
	m.Name = p.Name
	m.Params = b
	m.Cron = p.Cron
	m.Timezone = p.Timezone
	m.MisfirePolicy = p.MisfirePolicy
	m.ConcurrencyPolicy = p.ConcurrencyPolicy
	m.NextRunAt = spec.Next(time.Now().UTC()).UTC()
	if m.NextRunAt.IsZero() {
		return errno.ErrInvalidArgument
	}
	return nil
}
func validateScheduleInput(p ScheduleInput) error {
	if len(p.Name) == 0 || len(p.Name) > 80 {
		return errno.ErrInvalidArgument
	}
	if err := job.ValidateReport(job.ReportParams{Days: p.Days}); err != nil {
		return errno.ErrInvalidArgument
	}
	if p.MisfirePolicy != "skip" && p.MisfirePolicy != "catch_up" {
		return errno.ErrInvalidArgument
	}
	if p.ConcurrencyPolicy != "allow" && p.ConcurrencyPolicy != "forbid" {
		return errno.ErrInvalidArgument
	}
	return nil
}

func (s *Service) persistSchedule(ctx context.Context, id string, version int64, m *model.ScheduleM) error {
	return s.core.TX(ctx, func(tx context.Context) error {
		var err error
		if id == "" {
			err = s.plans.Create(tx, m)
		} else {
			err = s.plans.Update(tx, m, version)
		}
		if err != nil {
			return err
		}
		return s.auth.Audit(tx, contextx.UserID(ctx), "schedule.save", m.ScheduleID, "success", map[string]any{"version": m.Version})
	})
}

func (s *Service) commitReplay(ctx context.Context, e outbox.Envelope, id, eventID, actor, key string) error {
	db := s.core.DB(ctx).WithContext(ctx)
	evt, err := outbox.NewEvent(outbox.NewEventInput{EventID: eventID, EventType: e.EventType, Producer: "gin-enterprise-template-replay", AggregateType: e.AggregateType, AggregateID: e.AggregateID, TraceID: e.TraceID, OccurredAt: e.OccurredAt, Payload: e.Payload})
	if err != nil {
		return err
	}
	if err := db.Exec("INSERT INTO event_replay(replay_id,dead_letter_id,event_id,actor_id,idempotency_key) VALUES(?,?,?,?,?)", "rep_"+uuid.NewString(), id, eventID, actor, key).Error; err != nil {
		return err
	}
	if err := s.core.Outbox().Create(ctx, evt); err != nil {
		return err
	}
	return s.auth.Audit(ctx, actor, "dlq.replay", id, "success", map[string]any{"event_id": eventID})
}

func (s *Service) prepareReplay(ctx context.Context, id, eventID, actor, key string) (string, error) {
	db := s.core.DB(ctx).WithContext(ctx)
	if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "replay:"+actor+":"+key).Error; err != nil {
		return "", err
	}
	previous, err := existingReplay(db, actor, key, id)
	if err != nil {
		return "", err
	}
	if previous != "" {
		return previous, nil
	}
	e, err := s.replayEnvelope(ctx, id)
	if err != nil {
		return "", err
	}
	return eventID, s.commitReplay(ctx, e, id, eventID, actor, key)
}

func (s *Service) auditFailure(ctx context.Context, action, target string, err error) error {
	if err == nil {
		return nil
	}
	audit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	return errors.Join(err, s.auth.Audit(audit, contextx.UserID(ctx), action, target, "failed", nil))
}

// RejectManagement records rate-limit failures without the rejected request payload.
func (s *Service) RejectManagement(ctx context.Context, action, target string) error {
	return s.auth.Audit(ctx, contextx.UserID(ctx), action, target, "denied", nil)
}

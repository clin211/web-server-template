package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var errTaskLeaseLost = errors.New("task lease lost")
var errIdempotencyKeyConflict = errors.New("idempotency key conflict")
var errResultExceedsLimit = errors.New("result exceeds limit")
var errReportPermissionRevoked = errors.New("report permission revoked")

// ErrTaskLeaseLost prevents a stale execution from committing state or results.
var ErrTaskLeaseLost = errTaskLeaseLost

// ErrIdempotencyConflict indicates a key reused with different parameters.
var ErrIdempotencyConflict = errIdempotencyKeyConflict

// TaskStore persists short claims and fenced task transitions.
type TaskStore struct{ core IStore }

// NewTaskStore uses the existing transaction context protocol.
func NewTaskStore(core IStore) *TaskStore { return &TaskStore{core} }

// Create atomically inserts an intent or returns the matching original instance.
func (s *TaskStore) Create(ctx context.Context, t *model.TaskM) error {
	result := s.core.DB(ctx).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(t)
	if result.Error != nil {
		return fmt.Errorf("create task: %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return nil
	}
	if t.IdempotencyKey == nil {
		return ErrIdempotencyConflict
	}
	var old model.TaskM
	if err := s.core.DB(ctx).WithContext(ctx).Where("owner_id=? AND task_type=? AND idempotency_key=?", t.OwnerID, t.TaskType, *t.IdempotencyKey).Take(&old).Error; err != nil {
		return err
	}
	if old.RequestHash != t.RequestHash {
		return ErrIdempotencyConflict
	}
	*t = old
	return nil
}

// Get returns a task, leaving resource policy to the business consumer.
func (s *TaskStore) Get(ctx context.Context, id string) (model.TaskM, error) {
	var t model.TaskM
	err := s.core.DB(ctx).WithContext(ctx).Where("task_id=?", id).Take(&t).Error
	return t, err
}

// List applies ownership in SQL, bounds pages and uses the internal key only as an opaque cursor source.
func (s *TaskStore) List(ctx context.Context, owner string, before int64) ([]model.TaskM, error) {
	rows := make([]model.TaskM, 0)
	q := s.core.DB(ctx).WithContext(ctx)
	if owner != "" {
		q = q.Where("owner_id=?", owner)
	}
	if before > 0 {
		q = q.Where("id<?", before)
	}
	err := q.Order("id DESC").Limit(100).Find(&rows).Error
	return rows, err
}

// Attempts returns retained execution history.
func (s *TaskStore) Attempts(ctx context.Context, id string) ([]model.TaskAttemptM, error) {
	a := make([]model.TaskAttemptM, 0)
	err := s.core.DB(ctx).WithContext(ctx).Where("task_id=?", id).Order("attempt").Find(&a).Error
	return a, err
}

// Claim only claims one task for an available worker slot. No network I/O occurs here.
func (s *TaskStore) Claim(ctx context.Context, worker string, lease time.Duration) (*model.TaskM, error) {
	var claimed *model.TaskM
	err := s.core.TX(ctx, func(tx context.Context) error {
		db := s.core.DB(tx).WithContext(tx)
		// Candidates are read unlocked first so every path can lock schedule before task.
		var candidates []model.TaskM
		if err := db.Where("(status IN ('pending','retry_wait') AND available_at<=clock_timestamp()) OR (status='running' AND lease_until<=clock_timestamp())").Order("priority DESC,available_at,id").Limit(32).Find(&candidates).Error; err != nil {
			return err
		}
		first, claimErr := claimFirst(db, candidates, worker, lease)
		claimed = first
		return claimErr
	})
	if errors.Is(err, errNoRunnableTask) {
		return claimed, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim task: %w", err)
	}
	return claimed, nil
}
func claimCandidate(db *gorm.DB, candidate model.TaskM, worker string, lease time.Duration) (*model.TaskM, error) {
	if err := lockClaimSchedule(db, candidate.ScheduleID); err != nil {
		return nil, err
	}
	var t model.TaskM
	r := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("task_id=? AND ((status IN ('pending','retry_wait') AND available_at<=clock_timestamp()) OR (status='running' AND lease_until<=clock_timestamp()))", candidate.TaskID).Find(&t)
	if r.Error != nil {
		return nil, r.Error
	}
	if r.RowsAffected == 0 {
		return nil, errNoRunnableTask
	}
	exhausted, err := closeExpiredAttempt(db, &t)
	if err != nil {
		return nil, err
	}
	if exhausted {
		return nil, errNoRunnableTask
	}
	overlap, err := taskOverlap(db, t)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, errNoRunnableTask
	}
	query := `UPDATE task SET status='running',attempts=attempts+1,fencing_token=fencing_token+1,worker_id=?,lease_until=clock_timestamp()+(? * interval '1 second'),started_at=coalesce(started_at,clock_timestamp()),updated_at=clock_timestamp() WHERE task_id=? RETURNING *`
	if err := db.Raw(query, worker, lease.Seconds(), t.TaskID).Scan(&t).Error; err != nil {
		return nil, err
	}
	if err := db.Exec(`INSERT INTO task_attempt(task_id,attempt,worker_id,fencing_token,lease_until,heartbeat_at,started_at,trace_id) VALUES(?,?,?,?,?,clock_timestamp(),clock_timestamp(),?)`, t.TaskID, t.Attempts, worker, t.FencingToken, t.LeaseUntil, t.TraceID).Error; err != nil {
		return nil, err
	}
	return &t, nil
}
func closeExpiredAttempt(db *gorm.DB, t *model.TaskM) (bool, error) {
	if t.Status == "running" {
		if err := db.Exec("UPDATE task_attempt SET finished_at=clock_timestamp(),outcome='lease_expired',error_code='lease_expired' WHERE task_id=? AND attempt=? AND finished_at IS NULL", t.TaskID, t.Attempts).Error; err != nil {
			return false, err
		}
	}
	if t.CancelRequestedAt == nil && t.Attempts < t.MaxAttempts {
		return false, nil
	}
	status, code := "failed", "attempts_exhausted"
	if t.CancelRequestedAt != nil {
		status, code = "canceled", "canceled"
	}
	err := db.Model(t).Updates(map[string]any{"status": status, "finished_at": gorm.Expr("clock_timestamp()"), "updated_at": gorm.Expr("clock_timestamp()"), "lease_until": nil, "error_code": code, "error_summary": code}).Error
	return true, err
}
func taskOverlap(db *gorm.DB, t model.TaskM) (bool, error) {
	if t.ScheduleID == nil {
		return false, nil
	}
	var p model.ScheduleM
	if err := db.Where("schedule_id=?", *t.ScheduleID).Take(&p).Error; err != nil {
		return false, err
	}
	return scheduleOverlap(db, p, t.TaskID, true)
}

func (s *TaskStore) fenced(ctx context.Context, t *model.TaskM) *gorm.DB {
	return s.core.DB(ctx).WithContext(ctx).Model(&model.TaskM{}).Where("task_id=? AND status='running' AND worker_id=? AND fencing_token=? AND lease_until>clock_timestamp()", t.TaskID, t.WorkerID, t.FencingToken)
}

// Heartbeat renews an unexpired lease and observes durable cancellation.
func (s *TaskStore) Heartbeat(ctx context.Context, t *model.TaskM, lease time.Duration) (bool, error) {
	var canceled bool
	err := s.core.TX(ctx, func(tx context.Context) error {
		var row model.TaskM
		r := s.fenced(tx, t).Clauses(clause.Locking{Strength: "UPDATE"}).Find(&row)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return ErrTaskLeaseLost
		}
		r = s.fenced(tx, t).Updates(map[string]any{"lease_until": gorm.Expr("clock_timestamp()+(? * interval '1 second')", lease.Seconds()), "updated_at": gorm.Expr("clock_timestamp()")})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return ErrTaskLeaseLost
		}
		if err := s.core.DB(tx).WithContext(tx).Exec("UPDATE task_attempt SET heartbeat_at=clock_timestamp(),lease_until=clock_timestamp()+(? * interval '1 second'),trace_id=? WHERE task_id=? AND attempt=?", lease.Seconds(), t.TraceID, t.TaskID, t.Attempts).Error; err != nil {
			return err
		}
		canceled = row.CancelRequestedAt != nil
		return nil
	})
	return canceled, err
}

// Finish commits a result and its business event with the current valid lease.
func (s *TaskStore) Finish(ctx context.Context, t *model.TaskM, result job.Result, runErr error) error {
	return s.core.TX(ctx, func(tx context.Context) error {
		var row model.TaskM
		r := s.fenced(tx, t).Clauses(clause.Locking{Strength: "UPDATE"}).Find(&row)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return ErrTaskLeaseLost
		}
		db := s.core.DB(tx).WithContext(tx)
		updates, status, code, err := s.finishUpdates(tx, db, t, row, result, runErr)
		if err != nil {
			return err
		}
		r = s.fenced(tx, t).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return ErrTaskLeaseLost
		}
		return db.Exec("UPDATE task_attempt SET finished_at=clock_timestamp(),outcome=?,error_code=?,trace_id=? WHERE task_id=? AND attempt=? AND fencing_token=?", status, code, t.TraceID, t.TaskID, t.Attempts, t.FencingToken).Error
	})
}
func taskOutcome(row model.TaskM, runErr error) (string, string) {
	if row.CancelRequestedAt != nil {
		return "canceled", "canceled"
	}
	if runErr == nil {
		return "succeeded", ""
	}
	code := "execution_failed"
	switch {
	case job.IsPermanent(runErr):
		return "failed", "permanent"
	case errors.Is(runErr, context.DeadlineExceeded):
		code = "timeout"
	case errors.Is(runErr, context.Canceled):
		code = "interrupted"
	}
	if row.Attempts < row.MaxAttempts {
		return "retry_wait", code
	}
	return "failed", code
}
func (s *TaskStore) saveReport(ctx context.Context, db *gorm.DB, t *model.TaskM, result job.Result) (string, error) {
	if err := lockReportPermission(ctx, db, t.OwnerID); err != nil {
		return "", err
	}
	if len(result.Content) > 10<<20 {
		return "", job.Permanent(errResultExceedsLimit)
	}
	id := "res_" + uuid.NewString()
	if err := db.Exec("INSERT INTO report_result(result_id,task_id,owner_id,content,content_type,expires_at) VALUES(?,?,?,?,?,clock_timestamp()+interval '7 days')", id, t.TaskID, t.OwnerID, result.Content, result.ContentType).Error; err != nil {
		return "", err
	}
	payload := fmt.Appendf(nil, `{"task_id":%q,"result_id":%q,"owner_id":%q}`, t.TaskID, id, t.OwnerID)
	trace := ""
	if t.TraceID != nil {
		trace = *t.TraceID
	}
	event, err := outbox.NewEvent(outbox.NewEventInput{EventID: "evt_" + uuid.NewString(), EventType: "report.generated.v1", AggregateType: "report", AggregateID: t.TaskID, Producer: "gin-enterprise-template-worker", TraceID: trace, OccurredAt: time.Now().UTC(), Payload: json.RawMessage(payload)})
	if err != nil {
		return "", err
	}
	return id, s.core.Outbox().Create(ctx, event)
}

func lockReportPermission(ctx context.Context, db *gorm.DB, owner string) error {
	var n int
	err := db.Raw(`SELECT 1 FROM "user" u JOIN user_role ur ON ur.user_id=u.user_id JOIN role_permission rp ON rp.role_id=ur.role_id WHERE u.user_id=? AND u.status=0 AND rp.permission_code='task.submit' LIMIT 1 FOR SHARE OF u,ur,rp`, owner).Scan(&n).Error
	if err != nil {
		return err
	}
	if n != 1 {
		return job.Permanent(errReportPermissionRevoked)
	}
	return nil
}

// Cancel requests cooperative cancellation, or immediately cancels unclaimed work.
func (s *TaskStore) Cancel(ctx context.Context, id string) error {
	r := s.core.DB(ctx).WithContext(ctx).Exec(`UPDATE task SET cancel_requested_at=coalesce(cancel_requested_at,clock_timestamp()),status=CASE WHEN status='running' THEN status ELSE 'canceled' END,finished_at=CASE WHEN status='running' THEN finished_at ELSE clock_timestamp() END,updated_at=clock_timestamp() WHERE task_id=? AND status IN ('pending','retry_wait','running')`, id)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected > 0 {
		return nil
	}
	t, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if t.Status == "canceled" {
		return nil
	}
	return ErrIdempotencyConflict
}

// Result returns an unexpired shared output.
func (s *TaskStore) Result(ctx context.Context, id string) (model.ReportResultM, error) {
	var r model.ReportResultM
	err := s.core.DB(ctx).WithContext(ctx).Where("task_id=? AND expires_at>clock_timestamp()", id).Take(&r).Error
	return r, err
}

var errNoRunnableTask = errors.New("no runnable task")

func claimFirst(db *gorm.DB, candidates []model.TaskM, worker string, lease time.Duration) (*model.TaskM, error) {
	for _, candidate := range candidates {
		t, err := claimCandidate(db, candidate, worker, lease)
		if errors.Is(err, errNoRunnableTask) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return t, nil
	}
	return nil, errNoRunnableTask
}

func lockClaimSchedule(db *gorm.DB, id *string) error {
	if id != nil {
		var p model.ScheduleM
		r := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("schedule_id=?", *id).Find(&p)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return errNoRunnableTask
		}
	}
	return nil
}

func (s *TaskStore) finishUpdates(ctx context.Context, db *gorm.DB, t *model.TaskM, row model.TaskM, result job.Result, runErr error) (map[string]any, string, string, error) {
	status, code := taskOutcome(row, runErr)
	updates := map[string]any{"updated_at": gorm.Expr("clock_timestamp()"), "lease_until": nil, "trace_id": t.TraceID}
	if status == "retry_wait" {
		updates["available_at"] = gorm.Expr("clock_timestamp()+(? * interval '1 second')", job.RetryDelay(row.Attempts).Seconds())
	}
	if status == "succeeded" {
		id, finalCode, err := s.resolveReport(ctx, db, t, result)
		if err != nil {
			return nil, "", "", err
		}
		if finalCode != "" {
			status, code = "failed", finalCode
		} else {
			updates["result_id"] = id
		}
	}
	updates["status"] = status
	if status != "retry_wait" {
		updates["finished_at"] = gorm.Expr("clock_timestamp()")
	}
	if code != "" {
		updates["error_code"] = code
		updates["error_summary"] = code
	}
	return updates, status, code, nil
}

func (s *TaskStore) resolveReport(ctx context.Context, db *gorm.DB, t *model.TaskM, result job.Result) (string, string, error) {
	id, err := s.saveReport(ctx, db, t, result)
	if job.IsPermanent(err) {
		return "", "permission_revoked", nil
	}
	return id, "", err
}

package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var errCronHasNoNextOccurrence = errors.New("cron has no next occurrence")
var errCronHasNoFutureOccurrence = errors.New("cron has no future occurrence")

// ScheduleStore owns atomic schedule cursor advancement and task intent insertion.
type ScheduleStore struct{ core IStore }

// NewScheduleStore constructs a schedule repository.
func NewScheduleStore(core IStore) *ScheduleStore { return &ScheduleStore{core} }

// Create stores a validated plan.
func (s *ScheduleStore) Create(ctx context.Context, m *model.ScheduleM) error {
	return s.core.DB(ctx).WithContext(ctx).Create(m).Error
}

// Get retrieves the plan, including archived history.
func (s *ScheduleStore) Get(ctx context.Context, id string) (model.ScheduleM, error) {
	var m model.ScheduleM
	err := s.core.DB(ctx).WithContext(ctx).Where("schedule_id=?", id).Take(&m).Error
	return m, err
}

// List bounds owner-filtered plans.
func (s *ScheduleStore) List(ctx context.Context, owner string) ([]model.ScheduleM, error) {
	m := make([]model.ScheduleM, 0)
	q := s.core.DB(ctx).WithContext(ctx)
	if owner != "" {
		q = q.Where("owner_id=?", owner)
	}
	err := q.Order("id DESC").Limit(100).Find(&m).Error
	return m, err
}

// Update rejects stale versions and serializes all changes with dispatch.
func (s *ScheduleStore) Update(ctx context.Context, m *model.ScheduleM, expected int64) error {
	r := s.core.DB(ctx).WithContext(ctx).Model(&model.ScheduleM{}).Where("schedule_id=? AND version=? AND NOT archived", m.ScheduleID, expected).Updates(map[string]any{"name": m.Name, "params": m.Params, "params_version": m.ParamsVersion, "cron": m.Cron, "timezone": m.Timezone, "enabled": m.Enabled, "archived": m.Archived, "next_run_at": m.NextRunAt, "misfire_policy": m.MisfirePolicy, "concurrency_policy": m.ConcurrencyPolicy, "version": gorm.Expr("version+1"), "updated_at": gorm.Expr("clock_timestamp()")})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrIdempotencyConflict
	}
	return nil
}

// History returns trigger outcomes, including overlap and missed-time summaries.
func (s *ScheduleStore) History(ctx context.Context, id string) ([]map[string]any, error) {
	rows := make([]map[string]any, 0)
	err := s.core.DB(ctx).WithContext(ctx).Table("schedule_trigger").Select("schedule_id,scheduled_at,schedule_version,outcome,task_id,summary,created_at").Where("schedule_id=?", id).Order("id DESC").Limit(100).Find(&rows).Error
	return rows, err
}

// Tick locks bounded plans. Task creation and cursor progression share one commit.
func (s *ScheduleStore) Tick(ctx context.Context) error {
	return s.core.TX(ctx, func(tx context.Context) error {
		db := s.core.DB(tx).WithContext(tx)
		var now time.Time
		if err := db.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		var plans []model.ScheduleM
		if err := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("enabled AND NOT archived AND next_run_at<=?", now).Order("next_run_at,id").Limit(32).Find(&plans).Error; err != nil {
			return err
		}
		for _, p := range plans {
			if err := s.dispatchPlan(tx, db, p, now); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *ScheduleStore) dispatchPlan(ctx context.Context, db *gorm.DB, p model.ScheduleM, now time.Time) error {
	spec, err := job.ParseSchedule(p.Cron, p.Timezone)
	if err != nil {
		return err
	}
	if err := lockReportPermission(ctx, db, p.OwnerID); err != nil {
		if !job.IsPermanent(err) {
			return err
		}
		if err := db.Model(&p).Updates(map[string]any{"enabled": false, "version": gorm.Expr("version+1"), "updated_at": now}).Error; err != nil {
			return err
		}
		return NewAuthorizationStore(s.core).Audit(ctx, "system", "schedule.permission_revoked", p.ScheduleID, "denied", nil)
	}
	cursor, err := scheduleCursor(db, p, now, spec)
	if err != nil {
		return err
	}
	if cursor.IsZero() {
		return errCronHasNoFutureOccurrence
	}
	return db.Model(&p).Updates(map[string]any{"next_run_at": cursor.UTC(), "updated_at": now}).Error
}
func scheduleCursor(db *gorm.DB, p model.ScheduleM, now time.Time, spec cron.Schedule) (time.Time, error) {
	cursor := p.NextRunAt
	if p.MisfirePolicy == "skip" && now.Sub(cursor) > time.Minute {
		if err := recordTrigger(db, p, cursor, "misfire_skipped", "", "cursor advanced to future"); err != nil {
			return time.Time{}, err
		}
		return spec.Next(now), nil
	}
	if cursor.Before(now.Add(-24 * time.Hour)) {
		if err := recordTrigger(db, p, cursor, "window_skipped", "", "older than 24h"); err != nil {
			return time.Time{}, err
		}
		cursor = spec.Next(now.Add(-24 * time.Hour))
	}
	return catchUp(db, p, now, spec, cursor)
}
func fireSlot(db *gorm.DB, p model.ScheduleM, at, now time.Time) error {
	overlap, err := scheduleOverlap(db, p, "", false)
	if err != nil {
		return err
	}
	if overlap {
		return recordTrigger(db, p, at, "overlap_skipped", "", "")
	}
	id := "tsk_" + uuid.NewString()
	scheduled := at.UTC()
	version := p.Version
	t := model.TaskM{TaskID: id, TaskType: p.TaskType, Params: p.Params, ParamsVersion: p.ParamsVersion, Source: "schedule", ScheduleID: &p.ScheduleID, ScheduleVersion: &version, ScheduledAt: &scheduled, OwnerID: p.OwnerID, Status: "pending", AvailableAt: now, TimeoutSeconds: 300, MaxAttempts: 3, RequestHash: fmt.Sprintf("%x", sha256.Sum256(p.Params))}
	r := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&t)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		if err := db.Where("schedule_id=? AND scheduled_at=? AND source='schedule'", p.ScheduleID, scheduled).Take(&t).Error; err != nil {
			return err
		}
		id = t.TaskID
	}
	return recordTrigger(db, p, at, "created", id, "")
}
func scheduleOverlap(db *gorm.DB, p model.ScheduleM, except string, runningOnly bool) (bool, error) {
	if p.ConcurrencyPolicy != "forbid" {
		return false, nil
	}
	q := db.Model(&model.TaskM{}).Where("schedule_id=? AND task_id<>?", p.ScheduleID, except)
	if runningOnly {
		q = q.Where("status='running' AND lease_until>clock_timestamp()")
	} else {
		q = q.Where("status IN ('pending','running','retry_wait')")
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func recordTrigger(db *gorm.DB, p model.ScheduleM, at time.Time, outcome, id, summary string) error {
	return db.Exec(`INSERT INTO schedule_trigger(schedule_id,scheduled_at,schedule_version,outcome,task_id,summary) VALUES(?,?,?,?,?,?) ON CONFLICT DO NOTHING`, p.ScheduleID, at.UTC(), p.Version, outcome, id, summary).Error
}

// Retry creates a new instance while coordinating forbid-overlap with the plan lock.
func (s *TaskStore) Retry(ctx context.Context, old model.TaskM, key, hash string) (model.TaskM, error) {
	t := model.TaskM{TaskID: "tsk_" + uuid.NewString(), TaskType: old.TaskType, Params: old.Params, ParamsVersion: old.ParamsVersion, Source: "retry", ScheduleID: old.ScheduleID, ScheduleVersion: old.ScheduleVersion, RetryOf: &old.TaskID, OwnerID: old.OwnerID, Status: "pending", AvailableAt: time.Now().UTC(), TimeoutSeconds: old.TimeoutSeconds, MaxAttempts: old.MaxAttempts, IdempotencyKey: &key, RequestHash: hash}
	err := s.core.TX(ctx, func(tx context.Context) error {
		db := s.core.DB(tx).WithContext(tx)
		if err := db.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "retry:"+old.OwnerID+":"+old.TaskType+":"+key).Error; err != nil {
			return err
		}
		found, err := lookupRetry(db, &t, old.OwnerID, old.TaskType, key, hash)
		if err != nil {
			return err
		}
		if found {
			return nil
		}
		if err := lockRetrySchedule(db, old.ScheduleID); err != nil {
			return err
		}
		return s.Create(tx, &t)
	})
	return t, err
}

func lockRetrySchedule(db *gorm.DB, id *string) error {
	if id == nil {
		return nil
	}
	var p model.ScheduleM
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("schedule_id=?", *id).Take(&p).Error; err != nil {
		return err
	}
	overlap, err := scheduleOverlap(db, p, "", false)
	if err != nil {
		return err
	}
	if overlap {
		return ErrIdempotencyConflict
	}
	return nil
}

func catchUp(db *gorm.DB, p model.ScheduleM, now time.Time, spec cron.Schedule, cursor time.Time) (time.Time, error) {
	for count := 0; !cursor.After(now) && count < 100; count++ {
		if err := fireSlot(db, p, cursor, now); err != nil {
			return time.Time{}, err
		}
		cursor = spec.Next(cursor)
		if cursor.IsZero() {
			return time.Time{}, errCronHasNoNextOccurrence
		}
	}
	if !cursor.After(now) {
		if err := recordTrigger(db, p, cursor, "limit_skipped", "", "catch-up limit 100"); err != nil {
			return time.Time{}, err
		}
		cursor = spec.Next(now)
	}
	return cursor, nil
}

func lookupRetry(db *gorm.DB, t *model.TaskM, owner, typ, key, hash string) (bool, error) {
	var existing model.TaskM
	r := db.Where("owner_id=? AND task_type=? AND idempotency_key=?", owner, typ, key).Find(&existing)
	if r.Error != nil {
		return false, r.Error
	}
	if r.RowsAffected == 0 {
		return false, nil
	}
	if existing.RequestHash != hash {
		return false, ErrIdempotencyConflict
	}
	*t = existing
	return true, nil
}

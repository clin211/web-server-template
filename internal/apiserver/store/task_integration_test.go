package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/dbschema"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) (*gorm.DB, store.IStore) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration disabled in short mode")
	}
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN required")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	name := "tasks_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := raw.Close(); err != nil {
			t.Error(err)
		}
		if err := admin.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error; err != nil {
			t.Error(err)
		}
		a, e := admin.DB()
		if e != nil {
			t.Error(e)
		} else if err := a.Close(); err != nil {
			t.Error(err)
		}
	})
	goose := os.Getenv("GOOSE_BIN")
	if goose == "" {
		goose = "../../../_output/tools/goose"
	}
	goose, err = filepath.Abs(goose)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		cmd := exec.CommandContext(ctx, goose, "-env=none", "-dir", "../../../migrations", "up")
		cmd.Env = append(os.Environ(), "GOOSE_DRIVER=postgres", "GOOSE_DBSTRING="+u.String(), "GOOSE_TABLE=goose_db_version")
		output, runErr := cmd.CombinedOutput()
		cancel()
		if runErr != nil {
			t.Fatalf("official Goose up failed (run make tools.goose first): %v\n%s", runErr, output)
		}
	}
	if err := dbschema.Check(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"owner", "other"} {
		if err := db.Exec(`INSERT INTO "user"(user_id,username,password,nickname) VALUES(?,?,?,?)`, id, id, "test-hash", id).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("INSERT INTO user_role VALUES(?,'role_user')", id).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, store.NewStore(db)
}
func TestIntegrationTaskRecoveryAndAuthorization(t *testing.T) {
	db, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	s := taskbiz.New(core)
	repo := store.NewTaskStore(core)
	t.Run("transaction rollback removes task and event", func(t *testing.T) {
		err := core.TX(ctx, func(tx context.Context) error {
			if _, err := s.Submit(tx, job.ReportParams{Days: 7}, "rollback"); err != nil {
				return err
			}
			e, err := outbox.NewEvent(outbox.NewEventInput{EventID: "evt_rollback", EventType: "report.generated.v1", AggregateType: "report", AggregateID: "rollback", Producer: "test", OccurredAt: time.Now(), Payload: map[string]any{"task_id": "rollback"}})
			if err != nil {
				return err
			}
			if err := core.Outbox().Create(tx, e); err != nil {
				return err
			}
			return errors.New("rollback")
		})
		if err == nil {
			t.Fatal("expected rollback")
		}
		var n int64
		if err := db.Table("task").Where("idempotency_key='rollback'").Count(&n).Error; err != nil || n != 0 {
			t.Fatalf("task count %d %v", n, err)
		}
		if err := db.Table("outbox_event").Count(&n).Error; err != nil || n != 0 {
			t.Fatalf("event count %d %v", n, err)
		}
	})
	task, err := s.Submit(ctx, job.ReportParams{Days: 7}, "first")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Submit(ctx, job.ReportParams{Days: 7}, "first")
	if err != nil || again.TaskID != task.TaskID {
		t.Fatal("idempotency", err)
	}
	if _, err := s.Submit(ctx, job.ReportParams{Days: 8}, "first"); !errors.Is(err, store.ErrIdempotencyConflict) {
		t.Fatal("conflict", err)
	}
	other := contextx.WithUserID(ctx, "other")
	for _, permission := range []string{"task.view", "task.cancel", "task.retry", "result.download"} {
		if _, err := s.Get(other, task.TaskID, permission); !errors.Is(err, errno.ErrPermissionDenied) {
			t.Fatalf("foreign %s: %v", permission, err)
		}
	}
	var wg sync.WaitGroup
	claims := make(chan *model.TaskM, 2)
	for _, worker := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			x, err := repo.Claim(ctx, id, time.Minute)
			if err != nil {
				t.Error(err)
			}
			if x != nil {
				claims <- x
			}
		}(worker)
	}
	wg.Wait()
	close(claims)
	var first *model.TaskM
	n := 0
	for c := range claims {
		first = c
		n++
	}
	if n != 1 {
		t.Fatalf("claims=%d", n)
	}
	if err := db.Exec("UPDATE task SET lease_until=clock_timestamp()-interval '1 second' WHERE task_id=?", first.TaskID).Error; err != nil {
		t.Fatal(err)
	}
	second, err := repo.Claim(ctx, "worker-c", time.Minute)
	if err != nil || second == nil || second.Attempts != 2 {
		t.Fatalf("takeover: %+v %v", second, err)
	}
	if err := repo.Finish(ctx, first, job.Result{Content: []byte("stale")}, nil); !errors.Is(err, store.ErrTaskLeaseLost) {
		t.Fatal("stale accepted", err)
	}
	if err := repo.Finish(ctx, second, job.Result{}, errors.New("temporary")); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE task SET available_at=clock_timestamp()-interval '1 second' WHERE task_id=?", second.TaskID).Error; err != nil {
		t.Fatal(err)
	}
	third, err := store.NewTaskStore(core).Claim(ctx, "worker-restarted", time.Minute)
	if err != nil || third == nil || third.Attempts != 3 {
		t.Fatal("persisted attempts", err)
	}
	if err := repo.Finish(ctx, third, job.Result{}, job.Permanent(errors.New("secret must not appear"))); err != nil {
		t.Fatal(err)
	}
	failed, err := repo.Get(ctx, task.TaskID)
	if err != nil || failed.Status != "failed" || failed.ErrorSummary == nil || strings.Contains(*failed.ErrorSummary, "secret") {
		t.Fatal("permanent", err)
	}
	retried, err := s.Retry(ctx, failed.TaskID, "retry-key")
	if err != nil || retried.TaskID == failed.TaskID {
		t.Fatal("retry", err)
	}
	active, err := repo.Claim(ctx, "worker-new", time.Minute)
	if err != nil || active == nil {
		t.Fatal(err)
	}
	result, err := s.Generate(ctx, job.ReportParams{Days: 7})
	if err != nil {
		t.Fatal(err)
	}
	for _, sensitive := range []string{"password", "token", "test-hash"} {
		if strings.Contains(string(result.Content), sensitive) {
			t.Fatal("sensitive result")
		}
	}
	if err := repo.Finish(ctx, active, result, nil); err != nil {
		t.Fatal(err)
	}
	download, err := s.Download(ctx, active.TaskID)
	if err != nil || len(download.Content) == 0 {
		t.Fatal("download", err)
	}
	if _, err := s.Download(other, active.TaskID); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatal("foreign download", err)
	}
	events, err := core.Outbox().Claim(ctx, 10, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatalf("events %+v %v", events, err)
	}
	envelope, err := outbox.ParseEnvelope(events[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	router, err := s.EventRouter()
	if err != nil {
		t.Fatal(err)
	}
	processor, err := consumer.NewInboxProcessor(core, core.Inbox(), router, consumer.InboxOptions{ConsumerName: "test", LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	msg := kafka.Message{Topic: "test", Partition: 0, Offset: 1, Value: events[0].Payload, Headers: []kafka.Header{{Key: "event_id", Value: []byte(envelope.EventID)}, {Key: "event_type", Value: []byte(envelope.EventType)}}}
	if err := processor.Process(ctx, msg); err != nil {
		t.Fatal(err)
	}
	msg.Offset = 2
	if err := processor.Process(ctx, msg); err != nil {
		t.Fatal(err)
	}
	var notice int64
	if err := db.Table("report_notification").Count(&notice).Error; err != nil || notice != 1 {
		t.Fatal("duplicate effect", err)
	}
	// A terminal original Inbox must not swallow a controlled replay.
	bootstrapAdmin(t, db, "owner")
	deadEvent, err := outbox.NewEvent(outbox.NewEventInput{EventID: "evt_dead_original", EventType: envelope.EventType, AggregateType: envelope.AggregateType, AggregateID: envelope.AggregateID, Producer: "test", OccurredAt: time.Now(), Payload: json.RawMessage(envelope.Payload)})
	if err != nil {
		t.Fatal(err)
	}
	badRouter := consumer.NewEventRouter()
	if err := badRouter.Register(envelope.EventType, consumer.HandlerFunc(func(context.Context, outbox.Envelope) error {
		return consumer.Permanent(errors.New("test business failure"))
	})); err != nil {
		t.Fatal(err)
	}
	badProcessor, err := consumer.NewInboxProcessor(core, core.Inbox(), badRouter, consumer.InboxOptions{ConsumerName: "test", LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	badMsg := kafka.Message{Topic: "test", Partition: 0, Offset: 3, Value: deadEvent.Payload, Headers: []kafka.Header{{Key: "event_id", Value: []byte(deadEvent.EventID)}, {Key: "event_type", Value: []byte(deadEvent.EventType)}}}
	fail := badProcessor.Process(ctx, badMsg)
	if fail == nil {
		t.Fatal("expected permanent error")
	}
	publisher, err := consumer.NewPersistentDeadLetterPublisher(core, core.DeadLetter(), core.Inbox(), "test", "test.dlq")
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(ctx, consumer.DeadLetter{EventID: deadEvent.EventID, EventType: deadEvent.EventType, Topic: "test", Partition: 0, Offset: 3, Value: badMsg.Value, Headers: badMsg.Headers, Attempts: consumer.FailureAttempt(fail), FailureKind: "permanent", Error: "test failure", FailedAt: time.Now(), FinalizeInbox: true}); err != nil {
		t.Fatal(err)
	}
	var dead model.DeadLetterEventM
	if err := db.Where("original_event_id=?", deadEvent.EventID).Take(&dead).Error; err != nil {
		t.Fatal(err)
	}
	replayID, err := s.Replay(ctx, dead.DeadLetterID, "replay-key")
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := s.Replay(ctx, dead.DeadLetterID, "replay-key")
	if err != nil || repeat != replayID {
		t.Fatal("replay idempotency", err)
	}
	replayEvents, err := core.Outbox().Claim(ctx, 10, time.Minute)
	if err != nil || len(replayEvents) != 1 {
		t.Fatal("replay intent", err)
	}
	replay := replayEvents[0]
	badMsg.Value = replay.Payload
	badMsg.Offset = 4
	badMsg.Headers = []kafka.Header{{Key: "event_id", Value: []byte(replay.EventID)}, {Key: "event_type", Value: []byte(replay.EventType)}}
	if err := processor.Process(ctx, badMsg); err != nil {
		t.Fatal("replay swallowed", err)
	}
	var completed int64
	if err := db.Table("inbox_event").Where("event_id=? AND status='completed'", replayID).Count(&completed).Error; err != nil || completed != 1 {
		t.Fatal("replay not processed", err)
	}
	if err := db.Table("report_notification").Count(&notice).Error; err != nil || notice != 1 {
		t.Fatal("replay duplicated side effect", err)
	}
	var audited int64
	if err := db.Table("audit_log").Where("action='dlq.replay'").Count(&audited).Error; err != nil || audited != 1 {
		t.Fatal("replay audit", err)
	}
	if err := db.Exec("DELETE FROM user_role WHERE user_id='other'").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(other, job.ReportParams{Days: 1}, "revoked"); !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatal("permission change stale", err)
	}

	canceled, err := s.Submit(ctx, job.ReportParams{Days: 1}, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(ctx, canceled.TaskID); err != nil {
		t.Fatal(err)
	}
	canceled, err = repo.Get(ctx, canceled.TaskID)
	if err != nil || canceled.Status != "canceled" {
		t.Fatal("cancel", err)
	}
}
func TestIntegrationSchedulerCoordinationAndRecovery(t *testing.T) {
	db, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	s := taskbiz.New(core)
	plans := store.NewScheduleStore(core)
	p, err := s.SaveSchedule(ctx, "", taskbiz.ScheduleInput{Name: "periodic", Days: 7, Cron: "* * * * *", Timezone: "UTC", MisfirePolicy: "catch_up", ConcurrencyPolicy: "allow"})
	if err != nil {
		t.Fatal(err)
	}
	var now time.Time
	if err := db.Raw("SELECT date_trunc('minute',clock_timestamp())-interval '4 minutes'").Scan(&now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schedule SET next_run_at=? WHERE schedule_id=?", now, p.ScheduleID).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := plans.Tick(ctx); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var n int64
	if err := db.Table("task").Where("schedule_id=?", p.ScheduleID).Count(&n).Error; err != nil || n != 5 {
		t.Fatalf("catchup %d %v", n, err)
	}
	var duplicate int64
	if err := db.Raw("SELECT count(*) FROM (SELECT scheduled_at FROM task WHERE schedule_id=? GROUP BY scheduled_at HAVING count(*)>1) x", p.ScheduleID).Scan(&duplicate).Error; err != nil || duplicate != 0 {
		t.Fatal("duplicate", err)
	}
	p, err = plans.Get(ctx, p.ScheduleID)
	if err != nil {
		t.Fatal(err)
	}
	if !p.NextRunAt.After(time.Now()) {
		t.Fatal("cursor not advanced")
	}
	if err := s.ScheduleAction(ctx, p.ScheduleID, "pause", p.Version); err != nil {
		t.Fatal(err)
	}
	if err := s.ScheduleAction(ctx, p.ScheduleID, "resume", p.Version); !errors.Is(err, store.ErrIdempotencyConflict) {
		t.Fatal("stale update accepted", err)
	}
	forbid, err := s.SaveSchedule(ctx, "", taskbiz.ScheduleInput{Name: "forbid", Days: 1, Cron: "* * * * *", MisfirePolicy: "catch_up"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schedule SET next_run_at=? WHERE schedule_id=?", now, forbid.ScheduleID).Error; err != nil {
		t.Fatal(err)
	}
	if err := plans.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("task").Where("schedule_id=?", forbid.ScheduleID).Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("overlap tasks %d %v", n, err)
	}
	skip, err := s.SaveSchedule(ctx, "", taskbiz.ScheduleInput{Name: "skip", Days: 1, Cron: "* * * * *"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schedule SET next_run_at=? WHERE schedule_id=?", now, skip.ScheduleID).Error; err != nil {
		t.Fatal(err)
	}
	if err := plans.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("task").Where("schedule_id=?", skip.ScheduleID).Count(&n).Error; err != nil || n != 0 {
		t.Fatal("skip created", err)
	}
	data, err := json.Marshal(job.ReportParams{Days: 1})
	if err != nil {
		t.Fatal(err)
	}
	if string(datatypes.JSON(data)) == "" {
		t.Fatal("params")
	}
}

func TestIntegrationGooseHistory(t *testing.T) {
	for _, tt := range []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{name: "official history without custom ledger"},
		{name: "missing migration", sql: "DELETE FROM goose_db_version WHERE version_id=1", wantErr: true},
		{name: "future migration", sql: "INSERT INTO goose_db_version(version_id,is_applied) VALUES(3,true)", wantErr: true},
		{name: "rolled back migration", sql: "INSERT INTO goose_db_version(version_id,is_applied) VALUES(2,false)", wantErr: true},
		{name: "reapplied migration", sql: "INSERT INTO goose_db_version(version_id,is_applied) VALUES(2,false),(2,true)"},
		{name: "uninitialized database", sql: "DROP TABLE goose_db_version", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := testDatabase(t)
			var customLedger bool
			if err := db.Raw("SELECT to_regclass('public.schema_migration') IS NOT NULL").Scan(&customLedger).Error; err != nil {
				t.Fatal(err)
			}
			if customLedger {
				t.Fatal("official Goose must not create a custom migration ledger")
			}
			if tt.sql != "" {
				if err := db.Exec(tt.sql).Error; err != nil {
					t.Fatal(err)
				}
			}
			raw, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			if err := dbschema.Check(context.Background(), raw); (err != nil) != tt.wantErr {
				t.Fatalf("schema check error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestIntegrationLastAdministratorProtection(t *testing.T) {
	db, _ := testDatabase(t)
	bootstrapAdmin(t, db, "owner")
	if err := db.Exec(`DELETE FROM "user" WHERE user_id='owner'`).Error; err == nil {
		t.Fatal("last administrator deletion accepted")
	}
	if err := db.Exec(`UPDATE "user" SET status=1 WHERE user_id='owner'`).Error; err == nil {
		t.Fatal("last administrator disable accepted")
	}
}

func TestIntegrationRunningCancellationAndPermissionRevocation(t *testing.T) {
	db, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	service := taskbiz.New(core)
	repo := store.NewTaskStore(core)
	first, err := service.Submit(ctx, job.ReportParams{Days: 1}, "running-cancel")
	if err != nil {
		t.Fatal(err)
	}
	active, err := repo.Claim(ctx, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(ctx, first.TaskID); err != nil {
		t.Fatal(err)
	}
	canceled, err := repo.Heartbeat(ctx, active, time.Minute)
	if err != nil || !canceled {
		t.Fatal("durable cancellation", err)
	}
	if err := repo.Finish(ctx, active, job.Result{Content: []byte("result"), ContentType: "text/csv"}, nil); err != nil {
		t.Fatal(err)
	}
	first, err = repo.Get(ctx, first.TaskID)
	if err != nil || first.Status != "canceled" {
		t.Fatal("cancel lost to success", err)
	}
	second, err := service.Submit(ctx, job.ReportParams{Days: 1}, "revocation")
	if err != nil {
		t.Fatal(err)
	}
	active, err = repo.Claim(ctx, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DELETE FROM user_role WHERE user_id='owner'").Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, active, job.Result{Content: []byte("result"), ContentType: "text/csv"}, nil); err != nil {
		t.Fatal(err)
	}
	second, err = repo.Get(ctx, second.TaskID)
	if err != nil || second.Status != "failed" || second.ErrorCode == nil || *second.ErrorCode != "permission_revoked" {
		t.Fatal("revoked execution committed", err)
	}
	for _, table := range []string{"report_result", "outbox_event"} {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil || n != 0 {
			t.Fatalf("%s side effects %d %v", table, n, err)
		}
	}
}
func TestIntegrationSchedulerCatchupBounds(t *testing.T) {
	db, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	service := taskbiz.New(core)
	p, err := service.SaveSchedule(ctx, "", taskbiz.ScheduleInput{Name: "bounded", Days: 1, Cron: "* * * * *", Timezone: "UTC", MisfirePolicy: "catch_up", ConcurrencyPolicy: "allow"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE schedule SET next_run_at=clock_timestamp()-interval '3 days' WHERE schedule_id=?", p.ScheduleID).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.NewScheduleStore(core).Tick(ctx); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Table("task").Count(&n).Error; err != nil || n != 100 {
		t.Fatal("catchup count", n, err)
	}
	if err := db.Table("task").Where("scheduled_at<clock_timestamp()-interval '24 hours'").Count(&n).Error; err != nil || n != 0 {
		t.Fatal("catchup window", n, err)
	}
	for _, outcome := range []string{"window_skipped", "limit_skipped"} {
		if err := db.Table("schedule_trigger").Where("outcome=?", outcome).Count(&n).Error; err != nil || n != 1 {
			t.Fatal("skip summary", outcome, err)
		}
	}
}

func TestIntegrationEmptyHistoryAndTerminalCancel(t *testing.T) {
	_, core := testDatabase(t)
	ctx := contextx.WithUserID(context.Background(), "owner")
	service := taskbiz.New(core)
	p, err := service.SaveSchedule(ctx, "", taskbiz.ScheduleInput{Name: "empty", Days: 1, Cron: "* * * * *", Timezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := service.ScheduleHistory(ctx, p.ScheduleID)
	if err != nil || history == nil {
		t.Fatal("history must be JSON array", err)
	}
	_, err = service.Submit(ctx, job.ReportParams{Days: 1}, "terminal")
	if err != nil {
		t.Fatal(err)
	}
	repo := store.NewTaskStore(core)
	active, err := repo.Claim(ctx, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, active, job.Result{Content: []byte("safe"), ContentType: "text/csv"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Cancel(ctx, active.TaskID); !errors.Is(err, store.ErrIdempotencyConflict) {
		t.Fatal("completed task cancellation accepted", err)
	}
}

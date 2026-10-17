package store_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	taskbiz "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/task"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/job"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type failFirstPublisher struct {
	next   outbox.Publisher
	failed bool
}

func (p *failFirstPublisher) Publish(ctx context.Context, event outbox.Event) error {
	if !p.failed {
		p.failed = true
		return errors.New("injected broker unavailable")
	}
	return p.next.Publish(ctx, event)
}

type rejectCommitReader struct {
	reader  *kafka.Reader
	fetches int
}

func (r *rejectCommitReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.fetches++
	return r.reader.FetchMessage(ctx)
}
func (r *rejectCommitReader) CommitMessages(context.Context, ...kafka.Message) error {
	return errors.New("injected commit unavailable")
}

func TestIntegrationKafkaTaskEventRecovery(t *testing.T) {
	broker := os.Getenv("TEST_KAFKA_BROKERS")
	if broker == "" {
		t.Skip("TEST_KAFKA_BROKERS required")
	}
	db, core := testDatabase(t)
	ctx, cancel := context.WithTimeout(contextx.WithUserID(context.Background(), "owner"), 45*time.Second)
	defer cancel()
	topic := "tasks-recovery-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	conn, err := kafka.DialContext(ctx, "tcp", strings.Split(broker, ",")[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.CreateTopics(kafka.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.DeleteTopics(topic); err != nil {
			t.Error(err)
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	service := taskbiz.New(core)
	task, err := service.Submit(ctx, job.ReportParams{Days: 1}, "kafka")
	if err != nil {
		t.Fatal(err)
	}
	repo := store.NewTaskStore(core)
	active, err := repo.Claim(ctx, "kafka-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, active, job.Result{Content: []byte("user_id\nowner\n"), ContentType: "text/csv"}, nil); err != nil {
		t.Fatal(err)
	}
	writer := &kafka.Writer{Addr: kafka.TCP(broker), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	publisher, err := outbox.NewKafkaPublisher(writer)
	if err != nil {
		t.Fatal(err)
	}
	relay, err := outbox.NewRelay(core.Outbox(), &failFirstPublisher{next: publisher}, outbox.RelayOptions{BatchSize: 10, PollInterval: time.Millisecond, LeaseDuration: time.Minute, RetryDelay: time.Millisecond, MaxRetryDelay: time.Second, MaxAttempts: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := relay.RunOnce(ctx); err == nil {
		t.Fatal("missing publication failure")
	}
	if err := db.Exec("UPDATE outbox_event SET available_at=clock_timestamp()-interval '1 second' WHERE status='failed'").Error; err != nil {
		t.Fatal(err)
	}
	if err := relay.RunOnce(ctx); err != nil {
		t.Fatal("publication recovery", err)
	}
	var eventID string
	if err := db.Table("outbox_event").Select("event_id").Where("status='published'").Scan(&eventID).Error; err != nil || eventID == "" {
		t.Fatal("missing published intent", err)
	}
	failed := false
	handler := consumer.HandlerFunc(func(ctx context.Context, e outbox.Envelope) error {
		if !failed {
			failed = true
			return errors.New("injected business unavailable")
		}
		return service.ReportGenerated(ctx, e)
	})
	processor, err := consumer.NewInboxProcessor(core, core.Inbox(), handler, consumer.InboxOptions{ConsumerName: "kafka-recovery", LeaseDuration: time.Minute, RetryDelay: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	dlq, err := consumer.NewPersistentDeadLetterPublisher(core, core.DeadLetter(), core.Inbox(), "kafka-recovery", topic+".dlq")
	if err != nil {
		t.Fatal(err)
	}
	group := "recovery-" + uuid.NewString()
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(broker, ","), Topic: topic, GroupID: group, MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0})
	reject := &rejectCommitReader{reader: reader}
	worker, err := consumer.New(reject, processor, dlq, consumer.Options{MaxAttempts: 3, RetryDelay: time.Millisecond, MaxRetryDelay: time.Second, ProcessTimeout: 5 * time.Second, Wait: func(ctx context.Context, d time.Duration) error {
		if !failed {
			return errors.New("unexpected wait")
		}
		var n int64
		if err := db.Table("report_notification").Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return errors.New("stop on unconfirmed offset")
		}
		return db.Exec("UPDATE inbox_event SET available_at=clock_timestamp()-interval '1 second' WHERE event_id=?", eventID).Error
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Run(ctx); err == nil {
		t.Fatal("commit failure ignored")
	}
	if reject.fetches != 1 {
		t.Fatalf("fetched beyond failed commit: %d", reject.fetches)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := kafka.NewReader(kafka.ReaderConfig{Brokers: strings.Split(broker, ","), Topic: topic, GroupID: group, MinBytes: 1, MaxBytes: 1 << 20, CommitInterval: 0})
	t.Cleanup(func() {
		if err := restarted.Close(); err != nil {
			t.Error(err)
		}
	})
	msg, err := restarted.FetchMessage(ctx)
	if err != nil {
		t.Fatal("uncommitted replay", err)
	}
	env, err := outbox.ParseEnvelope(msg.Value)
	if err != nil || env.EventID != eventID {
		t.Fatal("wrong uncommitted delivery", err)
	}
	if err := processor.Process(ctx, msg); err != nil {
		t.Fatal("duplicate processing", err)
	}
	if err := restarted.CommitMessages(ctx, msg); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Table("report_notification").Where("task_id=?", task.TaskID).Count(&n).Error; err != nil || n != 1 {
		t.Fatal("duplicate side effect", err)
	}
}

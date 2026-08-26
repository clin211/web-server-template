package consumer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
)

type transactionKey struct{}

type fakeTransactioner struct {
	err error
}

func (t *fakeTransactioner) TX(ctx context.Context, fn func(context.Context) error) error {
	if t.err != nil {
		return t.err
	}
	return fn(context.WithValue(ctx, transactionKey{}, true))
}

type fakeInboxRepository struct {
	claim         Claim
	claimErr      error
	completed     []int
	completedInTX bool
	failed        []int
	failureReason string
}

func (r *fakeInboxRepository) Claim(context.Context, MessageIdentity, time.Duration) (Claim, error) {
	return r.claim, r.claimErr
}

func (r *fakeInboxRepository) MarkCompleted(ctx context.Context, _ string, _ string, attempt int, _ time.Time) error {
	r.completed = append(r.completed, attempt)
	r.completedInTX, _ = ctx.Value(transactionKey{}).(bool)
	return nil
}

func (r *fakeInboxRepository) MarkFailed(_ context.Context, _ string, _ string, attempt int, reason string, _ time.Time) error {
	r.failed = append(r.failed, attempt)
	r.failureReason = reason
	return nil
}

func TestInboxProcessorCompletesBusinessHandlerInTransaction(t *testing.T) {
	t.Parallel()

	repository := &fakeInboxRepository{claim: Claim{Disposition: ClaimAcquired, Attempt: 2}}
	handled := 0
	processor, err := NewInboxProcessor(&fakeTransactioner{}, repository, HandlerFunc(func(_ context.Context, envelope outbox.Envelope) error {
		handled++
		if envelope.EventID != "event-1" {
			t.Fatalf("event ID = %q", envelope.EventID)
		}
		return nil
	}), InboxOptions{ConsumerName: "users", LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatalf("NewInboxProcessor() error = %v", err)
	}

	if err := processor.Process(context.Background(), eventMessage(t)); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if handled != 1 {
		t.Fatalf("handled = %d, want 1", handled)
	}
	if len(repository.completed) != 1 || repository.completed[0] != 2 || !repository.completedInTX {
		t.Fatalf("completed = %v, in transaction = %v", repository.completed, repository.completedInTX)
	}
}

func TestInboxProcessorSkipsCompletedDuplicate(t *testing.T) {
	t.Parallel()

	repository := &fakeInboxRepository{claim: Claim{Disposition: ClaimCompleted, Attempt: 3}}
	handled := 0
	processor, err := NewInboxProcessor(&fakeTransactioner{}, repository, HandlerFunc(func(context.Context, outbox.Envelope) error {
		handled++
		return nil
	}), InboxOptions{ConsumerName: "users", LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatalf("NewInboxProcessor() error = %v", err)
	}

	if err := processor.Process(context.Background(), eventMessage(t)); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if handled != 0 || len(repository.completed) != 0 {
		t.Fatalf("handled = %d, completed = %v", handled, repository.completed)
	}
}

func TestInboxProcessorRejectsHeaderEnvelopeIdentityConflict(t *testing.T) {
	t.Parallel()

	message := eventMessage(t)
	message.Headers[0].Value = []byte("different-event")
	processor, err := NewInboxProcessor(&fakeTransactioner{}, &fakeInboxRepository{}, HandlerFunc(func(context.Context, outbox.Envelope) error {
		return nil
	}), InboxOptions{ConsumerName: "users", LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatalf("NewInboxProcessor() error = %v", err)
	}

	err = processor.Process(context.Background(), message)
	if !IsPermanent(err) || !errors.Is(err, ErrEventIdentityConflict) {
		t.Fatalf("Process() error = %v, want permanent identity conflict", err)
	}
}

func TestInboxProcessorMarksFailedAttempt(t *testing.T) {
	t.Parallel()

	repository := &fakeInboxRepository{claim: Claim{Disposition: ClaimAcquired, Attempt: 4}}
	handlerErr := errors.New("dependency unavailable")
	processor, err := NewInboxProcessor(&fakeTransactioner{}, repository, HandlerFunc(func(context.Context, outbox.Envelope) error {
		return handlerErr
	}), InboxOptions{
		ConsumerName:  "users",
		LeaseDuration: time.Minute,
		RetryDelay:    2 * time.Second,
		Now:           func() time.Time { return time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewInboxProcessor() error = %v", err)
	}

	err = processor.Process(context.Background(), eventMessage(t))
	if !errors.Is(err, handlerErr) || FailureAttempt(err) != 4 {
		t.Fatalf("Process() error = %v, attempt = %d", err, FailureAttempt(err))
	}
	if len(repository.failed) != 1 || repository.failed[0] != 4 || !strings.Contains(repository.failureReason, handlerErr.Error()) {
		t.Fatalf("failed = %v, reason = %q", repository.failed, repository.failureReason)
	}
}

func eventMessage(t *testing.T) kafka.Message {
	t.Helper()
	event, err := outbox.NewEvent(outbox.NewEventInput{
		EventID: "event-1", EventType: "example.created.v1", Producer: "apiserver",
		AggregateType: "user", AggregateID: "user-1", OccurredAt: time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC),
		Payload: map[string]string{"user_id": "user-1"},
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	return kafka.Message{
		Topic: "events", Partition: 1, Offset: 7, Value: event.Payload,
		Headers: []kafka.Header{{Key: "event_id", Value: []byte(event.EventID)}, {Key: "event_type", Value: []byte(event.EventType)}},
	}
}

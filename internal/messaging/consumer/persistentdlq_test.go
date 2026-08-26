package consumer

import (
	"context"
	"testing"
	"time"
)

type fakeDeadLetterRepository struct {
	record      StoredDeadLetter
	createdInTX bool
}

func (r *fakeDeadLetterRepository) Create(ctx context.Context, record StoredDeadLetter) error {
	r.record = record
	r.createdInTX, _ = ctx.Value(transactionKey{}).(bool)
	return nil
}

type fakeInboxFinalizer struct {
	consumerName string
	eventID      string
	attempt      int
	markedInTX   bool
}

func (f *fakeInboxFinalizer) MarkDeadLettered(ctx context.Context, consumerName, eventID string, attempt int, _ time.Time) error {
	f.consumerName = consumerName
	f.eventID = eventID
	f.attempt = attempt
	f.markedInTX, _ = ctx.Value(transactionKey{}).(bool)
	return nil
}

func TestPersistentDeadLetterPublisherStoresAndFinalizesInboxAtomically(t *testing.T) {
	t.Parallel()

	repository := &fakeDeadLetterRepository{}
	inbox := &fakeInboxFinalizer{}
	publisher, err := NewPersistentDeadLetterPublisher(
		&fakeTransactioner{},
		repository,
		inbox,
		"users",
		"events.dlq",
	)
	if err != nil {
		t.Fatalf("NewPersistentDeadLetterPublisher() error = %v", err)
	}
	record := DeadLetter{
		EventID: "event-1", EventType: "example.created.v1", Topic: "events", Partition: 1, Offset: 9,
		Attempts: 4, Error: "handler failed", FailedAt: time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC),
		FinalizeInbox: true,
	}

	if err := publisher.Publish(context.Background(), record); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if !repository.createdInTX || !inbox.markedInTX {
		t.Fatalf("created in TX = %v, marked in TX = %v", repository.createdInTX, inbox.markedInTX)
	}
	if repository.record.ConsumerName != "users" || repository.record.DLQTopic != "events.dlq" || repository.record.DeadLetterID == "" {
		t.Fatalf("stored record = %+v", repository.record)
	}
	if inbox.consumerName != "users" || inbox.eventID != "event-1" || inbox.attempt != 4 {
		t.Fatalf("inbox finalization = %q/%q/%d", inbox.consumerName, inbox.eventID, inbox.attempt)
	}
}

func TestPersistentDeadLetterPublisherDoesNotFinalizeInboxForMalformedEvent(t *testing.T) {
	t.Parallel()

	repository := &fakeDeadLetterRepository{}
	inbox := &fakeInboxFinalizer{}
	publisher, err := NewPersistentDeadLetterPublisher(&fakeTransactioner{}, repository, inbox, "users", "events.dlq")
	if err != nil {
		t.Fatalf("NewPersistentDeadLetterPublisher() error = %v", err)
	}

	if err := publisher.Publish(context.Background(), DeadLetter{Topic: "events", Partition: 0, Offset: 2, Error: "malformed"}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if inbox.eventID != "" {
		t.Fatalf("finalized inbox event = %q, want none", inbox.eventID)
	}
}

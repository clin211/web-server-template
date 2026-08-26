package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeReader struct {
	mu          sync.Mutex
	messages    []kafka.Message
	fetches     int
	committed   []kafka.Message
	commitErr   error
	operations  []string
	fetchNotify chan struct{}
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	r.fetches++
	r.operations = append(r.operations, "fetch")
	if len(r.messages) > 0 {
		message := r.messages[0]
		r.messages = r.messages[1:]
		r.mu.Unlock()
		return message, nil
	}
	notify := r.fetchNotify
	r.mu.Unlock()
	if notify != nil {
		select {
		case notify <- struct{}{}:
		default:
		}
	}
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *fakeReader) CommitMessages(_ context.Context, messages ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operations = append(r.operations, "commit")
	if r.commitErr != nil {
		return r.commitErr
	}
	r.committed = append(r.committed, messages...)
	return nil
}

func (r *fakeReader) snapshot() (fetches int, committed []kafka.Message, operations []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fetches, append([]kafka.Message(nil), r.committed...), append([]string(nil), r.operations...)
}

type processorFunc func(context.Context, kafka.Message) error

func (f processorFunc) Process(ctx context.Context, message kafka.Message) error {
	return f(ctx, message)
}

type fakeDLQ struct {
	mu      sync.Mutex
	records []DeadLetter
	err     error
}

func (d *fakeDLQ) Publish(_ context.Context, record DeadLetter) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil {
		return d.err
	}
	d.records = append(d.records, record)
	return nil
}

func TestConsumerProcessesBeforeCommitting(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{messages: []kafka.Message{{Topic: "events", Partition: 2, Offset: 7}}}
	consumer, err := New(reader, processorFunc(func(_ context.Context, _ kafka.Message) error {
		reader.mu.Lock()
		reader.operations = append(reader.operations, "process")
		reader.mu.Unlock()
		return nil
	}), &fakeDLQ{}, Options{MaxAttempts: 3, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	waitForCommitted(t, reader, 1)
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v", err)
	}

	_, committed, operations := reader.snapshot()
	if len(committed) != 1 || committed[0].Offset != 7 {
		t.Fatalf("committed = %v, want offset 7", committed)
	}
	wantOperations := []string{"fetch", "process", "commit", "fetch"}
	if !equalStrings(operations, wantOperations) {
		t.Fatalf("operations = %v, want %v", operations, wantOperations)
	}
}

func TestConsumerRetriesThenPublishesDLQBeforeCommit(t *testing.T) {
	t.Parallel()

	message := kafka.Message{
		Topic: "events", Partition: 1, Offset: 9, Key: []byte("user-1"), Value: []byte(`{"event_id":"event-1"}`),
		Headers: []kafka.Header{{Key: "event_id", Value: []byte("event-1")}, {Key: "event_type", Value: []byte("example.created.v1")}},
	}
	reader := &fakeReader{messages: []kafka.Message{message}}
	dlq := &fakeDLQ{}
	processErr := errors.New("poison message")
	attempts := 0
	consumer, err := New(reader, processorFunc(func(context.Context, kafka.Message) error {
		attempts++
		return processErr
	}), dlq, Options{
		MaxAttempts:    3,
		RetryDelay:     time.Nanosecond,
		MaxRetryDelay:  time.Nanosecond,
		ProcessTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	waitForCommitted(t, reader, 1)
	cancel()
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v", err)
	}

	if attempts != 3 {
		t.Fatalf("process attempts = %d, want 3", attempts)
	}
	if len(dlq.records) != 1 {
		t.Fatalf("DLQ records = %d, want 1", len(dlq.records))
	}
	record := dlq.records[0]
	if record.EventID != "event-1" || record.EventType != "example.created.v1" || record.Attempts != 3 || record.Error != processErr.Error() {
		t.Fatalf("DLQ record = %+v", record)
	}
}

func TestConsumerDoesNotFetchNextMessageWhenCommitFails(t *testing.T) {
	t.Parallel()

	commitErr := errors.New("commit unavailable")
	reader := &fakeReader{
		messages:  []kafka.Message{{Topic: "events", Offset: 1}, {Topic: "events", Offset: 2}},
		commitErr: commitErr,
	}
	consumer, err := New(reader, processorFunc(func(context.Context, kafka.Message) error { return nil }), &fakeDLQ{}, Options{
		MaxAttempts:    1,
		RetryDelay:     time.Hour,
		MaxRetryDelay:  time.Hour,
		ProcessTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	waitForOperation(t, reader, "commit")
	cancel()
	if err := <-done; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want deadline exceeded", err)
	}

	fetches, committed, _ := reader.snapshot()
	if fetches != 1 {
		t.Fatalf("fetches = %d, want 1", fetches)
	}
	if len(committed) != 0 {
		t.Fatalf("committed = %v, want none", committed)
	}
}

func TestConsumerDrainsFetchedMessageAfterRootCancellation(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{messages: []kafka.Message{{Topic: "events", Offset: 3}}}
	started := make(chan struct{})
	release := make(chan struct{})
	consumer, err := New(reader, processorFunc(func(ctx context.Context, _ kafka.Message) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}), &fakeDLQ{}, Options{MaxAttempts: 1, ProcessTimeout: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	<-started
	cancel()
	close(release)
	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v", err)
	}

	_, committed, _ := reader.snapshot()
	if len(committed) != 1 || committed[0].Offset != 3 {
		t.Fatalf("committed = %v, want offset 3", committed)
	}
}

func waitForCommitted(t *testing.T, reader *fakeReader, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, committed, _ := reader.snapshot()
		if len(committed) >= count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d committed messages", count)
}

func waitForOperation(t *testing.T, reader *fakeReader, operation string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, _, operations := reader.snapshot()
		for _, current := range operations {
			if current == operation {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for operation %q", operation)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

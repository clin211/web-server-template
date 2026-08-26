package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	events        []Event
	claimErr      error
	publishedIDs  []string
	publishedRuns []int
	failedIDs     []string
	failedRuns    []int
	nextAvailable time.Time
	exhaustedIDs  []string
}

func (r *fakeRepository) Claim(context.Context, int, time.Duration) ([]Event, error) {
	return r.events, r.claimErr
}

func (r *fakeRepository) MarkPublished(_ context.Context, eventID string, attempt int, _ time.Time) error {
	r.publishedIDs = append(r.publishedIDs, eventID)
	r.publishedRuns = append(r.publishedRuns, attempt)
	return nil
}

func (r *fakeRepository) MarkFailed(_ context.Context, eventID string, attempt int, _ string, next time.Time) error {
	r.failedIDs = append(r.failedIDs, eventID)
	r.failedRuns = append(r.failedRuns, attempt)
	r.nextAvailable = next
	return nil
}

func (r *fakeRepository) MarkExhausted(_ context.Context, eventID string, _ int, _ string, _ time.Time) error {
	r.exhaustedIDs = append(r.exhaustedIDs, eventID)
	return nil
}

type fakePublisher struct {
	err       error
	published []string
}

func (p *fakePublisher) Publish(_ context.Context, event Event) error {
	p.published = append(p.published, event.EventID)
	return p.err
}

func TestRelayRunOnceMarksPublishedEvents(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{events: []Event{{EventID: "event-1", Attempts: 3}, {EventID: "event-2", Attempts: 7}}}
	publisher := &fakePublisher{}
	relay, err := NewRelay(repository, publisher, RelayOptions{BatchSize: 10, LeaseDuration: time.Minute, RetryDelay: time.Second})
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}

	if err := relay.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(repository.publishedIDs) != 2 {
		t.Fatalf("published count = %d, want 2", len(repository.publishedIDs))
	}
	if len(repository.failedIDs) != 0 {
		t.Fatalf("failed count = %d, want 0", len(repository.failedIDs))
	}
	if got := repository.publishedRuns; len(got) != 2 || got[0] != 3 || got[1] != 7 {
		t.Fatalf("published attempts = %v, want [3 7]", got)
	}
}

func TestRelayRunOnceSchedulesFailedEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 21, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{events: []Event{{EventID: "event-1", Attempts: 2}}}
	publisher := &fakePublisher{err: errors.New("kafka unavailable")}
	relay, err := NewRelay(repository, publisher, RelayOptions{
		BatchSize:     10,
		LeaseDuration: time.Minute,
		RetryDelay:    time.Second,
		Now:           func() time.Time { return now },
		Jitter:        func(delay time.Duration) time.Duration { return delay },
	})
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}

	if err := relay.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() error = nil, want publish error")
	}
	if len(repository.failedIDs) != 1 || repository.failedIDs[0] != "event-1" {
		t.Fatalf("failed IDs = %v, want [event-1]", repository.failedIDs)
	}
	if got := repository.failedRuns; len(got) != 1 || got[0] != 2 {
		t.Fatalf("failed attempts = %v, want [2]", got)
	}
	wantNext := now.Add(2 * time.Second)
	if !repository.nextAvailable.Equal(wantNext) {
		t.Fatalf("next available = %s, want %s", repository.nextAvailable, wantNext)
	}
}

func TestRelayAppliesRetryJitter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 21, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{events: []Event{{EventID: "event-1", Attempts: 1}}}
	relay, err := NewRelay(repository, &fakePublisher{err: errors.New("kafka unavailable")}, RelayOptions{
		BatchSize:     1,
		LeaseDuration: time.Minute,
		RetryDelay:    2 * time.Second,
		Now:           func() time.Time { return now },
		Jitter: func(delay time.Duration) time.Duration {
			if delay != 2*time.Second {
				t.Fatalf("jitter input = %s, want 2s", delay)
			}
			return 1500 * time.Millisecond
		},
	})
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}

	if err := relay.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() error = nil, want publish error")
	}
	if want := now.Add(1500 * time.Millisecond); !repository.nextAvailable.Equal(want) {
		t.Fatalf("next available = %s, want %s", repository.nextAvailable, want)
	}
}

func TestRelayMovesExhaustedEventToTerminalState(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{events: []Event{{EventID: "event-1", Attempts: 3}}}
	relay, err := NewRelay(repository, &fakePublisher{err: errors.New("kafka unavailable")}, RelayOptions{
		BatchSize: 1, LeaseDuration: time.Minute, RetryDelay: time.Second, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}
	if err := relay.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() error = nil, want exhausted publish error")
	}
	if len(repository.exhaustedIDs) != 1 || len(repository.failedIDs) != 0 {
		t.Fatalf("exhausted IDs = %v, failed IDs = %v", repository.exhaustedIDs, repository.failedIDs)
	}
}

func TestRelayRunStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	relay, err := NewRelay(&fakeRepository{}, &fakePublisher{}, RelayOptions{
		BatchSize:     1,
		LeaseDuration: time.Minute,
		RetryDelay:    time.Second,
		PollInterval:  time.Hour,
	})
	if err != nil {
		t.Fatalf("NewRelay() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

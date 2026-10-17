package apiserver

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

type blockingRunner struct {
	started chan struct{}
	once    sync.Once
}

func (r *blockingRunner) Run(ctx context.Context) error {
	r.once.Do(func() { close(r.started) })
	<-ctx.Done()
	return nil
}

type closeRecorder struct {
	count atomic.Int32
	err   error
}

type recordingKafkaWriter struct {
	closeRecorder
}

func (*recordingKafkaWriter) WriteMessages(context.Context, ...kafka.Message) error { return nil }

type failingKafkaFactory struct {
	messageWriter kafkaMessageWriter
	readerErr     error
}

func (f failingKafkaFactory) writer() (kafkaMessageWriter, error) {
	return f.messageWriter, nil
}

func (f failingKafkaFactory) writerForTopic(string) (kafkaMessageWriter, error) {
	return nil, f.readerErr
}

func (f failingKafkaFactory) reader() (kafkaMessageReader, error) {
	return nil, f.readerErr
}

func TestNewAsyncRuntimeRejectsConsumerWithoutRegisteredHandlers(t *testing.T) {
	t.Parallel()

	consumerOptions := genericoptions.NewConsumerOptions()
	consumerOptions.Enabled = true
	runtime, err := NewAsyncRuntime(&Config{
		KafkaOptions:    genericoptions.NewKafkaOptions(),
		ConsumerOptions: consumerOptions,
	}, nil)
	if runtime != nil {
		t.Fatal("NewAsyncRuntime() runtime is non-nil, want fail-closed startup")
	}
	if !errors.Is(err, errConsumerHandlersRequired) {
		t.Fatalf("NewAsyncRuntime() error = %v, want %v", err, errConsumerHandlersRequired)
	}
}

func TestNewAsyncRuntimeClosesCreatedResourcesOnConstructionFailure(t *testing.T) {
	t.Parallel()

	outboxOptions := genericoptions.NewOutboxOptions()
	outboxOptions.Enabled = true
	consumerOptions := genericoptions.NewConsumerOptions()
	consumerOptions.Enabled = true
	readerErr := errors.New("reader construction failed")
	closeErr := errors.New("writer close failed")
	writer := &recordingKafkaWriter{closeRecorder: closeRecorder{err: closeErr}}
	router := consumer.NewEventRouter()
	if err := router.Register("example.created.v1", consumer.HandlerFunc(func(context.Context, outbox.Envelope) error {
		return nil
	})); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	runtime, err := newAsyncRuntimeWithKafkaFactory(&Config{
		KafkaOptions:    genericoptions.NewKafkaOptions(),
		OutboxOptions:   outboxOptions,
		ConsumerOptions: consumerOptions,
	}, store.NewStore(nil), router, failingKafkaFactory{messageWriter: writer, readerErr: readerErr})
	if runtime != nil {
		t.Fatal("newAsyncRuntimeWithKafkaFactory() runtime is non-nil, want construction failure")
	}
	if !errors.Is(err, readerErr) {
		t.Fatalf("newAsyncRuntimeWithKafkaFactory() error = %v, want reader error", err)
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("newAsyncRuntimeWithKafkaFactory() error = %v, want close error", err)
	}
	if got := writer.count.Load(); got != 1 {
		t.Fatalf("writer close count = %d, want 1", got)
	}
}

func (c *closeRecorder) Close() error {
	c.count.Add(1)
	return c.err
}

func TestAsyncRuntimeStopsRunnerBeforeClosingPublisher(t *testing.T) {
	t.Parallel()

	runner := &blockingRunner{started: make(chan struct{})}
	closer := &closeRecorder{}
	runtime := newAsyncRuntime(runner, closer)
	ctx, cancel := context.WithCancel(context.Background())
	runtime.Start(ctx)

	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if got := closer.count.Load(); got != 1 {
		t.Fatalf("publisher close count = %d, want 1", got)
	}
}

func TestAsyncRuntimeShutdownIsIdempotent(t *testing.T) {
	t.Parallel()

	runner := &blockingRunner{started: make(chan struct{})}
	closeErr := errors.New("close failed")
	closer := &closeRecorder{err: closeErr}
	runtime := newAsyncRuntime(runner, closer)
	runtime.Start(context.Background())
	<-runner.started

	for range 2 {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := runtime.Shutdown(shutdownCtx)
		cancel()
		if !errors.Is(err, closeErr) {
			t.Fatalf("Shutdown() error = %v, want %v", err, closeErr)
		}
	}
	if got := closer.count.Load(); got != 1 {
		t.Fatalf("publisher close count = %d, want 1", got)
	}
}

type delayedRunner struct {
	started chan struct{}
	release chan struct{}
}

func (r *delayedRunner) Run(ctx context.Context) error {
	close(r.started)
	<-ctx.Done()
	<-r.release
	return nil
}

func TestAsyncRuntimeShutdownContinuesCleanupAfterCallerTimeout(t *testing.T) {
	t.Parallel()

	runner := &delayedRunner{started: make(chan struct{}), release: make(chan struct{})}
	closer := &closeRecorder{}
	runtime := newAsyncRuntime(runner, closer)
	runtime.Start(context.Background())
	<-runner.started

	shutdownCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Shutdown(shutdownCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown() error = %v, want context canceled", err)
	}
	if got := closer.count.Load(); got != 0 {
		t.Fatalf("publisher close count before runner exits = %d, want 0", got)
	}

	close(runner.release)
	shutdownCtx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if got := closer.count.Load(); got != 1 {
		t.Fatalf("publisher close count = %d, want 1", got)
	}
}

type failingRunner struct {
	err error
}

func (r failingRunner) Run(context.Context) error { return r.err }

func TestAsyncRuntimeExposesRunnerFailure(t *testing.T) {
	t.Parallel()

	runnerErr := errors.New("consumer stopped")
	runtime := newAsyncRuntime(failingRunner{err: runnerErr}, nil)
	runtime.Start(context.Background())
	select {
	case <-runtime.Done():
		if !errors.Is(runtime.Err(), runnerErr) {
			t.Fatalf("Err() = %v, want %v", runtime.Err(), runnerErr)
		}
		if err := runtime.Check(context.Background()); !errors.Is(err, runnerErr) {
			t.Fatalf("Check() error = %v, want runner failure", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runtime did not expose runner failure")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); !errors.Is(err, runnerErr) {
		t.Fatalf("Shutdown() error = %v, want %v", err, runnerErr)
	}
}

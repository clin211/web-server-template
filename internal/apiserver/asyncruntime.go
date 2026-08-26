package apiserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/metrics"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

type asyncRunner interface {
	Run(ctx context.Context) error
}

type kafkaMessageWriter interface {
	outbox.MessageWriter
	io.Closer
}

type kafkaMessageReader interface {
	consumer.MessageReader
	io.Closer
}

type asyncKafkaFactory interface {
	Writer() (kafkaMessageWriter, error)
	WriterForTopic(topic string) (kafkaMessageWriter, error)
	Reader() (kafkaMessageReader, error)
}

type kafkaOptionsFactory struct {
	options *genericoptions.KafkaOptions
}

func (f kafkaOptionsFactory) Writer() (kafkaMessageWriter, error) {
	return f.options.Writer()
}

func (f kafkaOptionsFactory) WriterForTopic(topic string) (kafkaMessageWriter, error) {
	return f.options.WriterForTopic(topic)
}

func (f kafkaOptionsFactory) Reader() (kafkaMessageReader, error) {
	return f.options.Reader()
}

var (
	errKafkaOptionsRequired     = errors.New("kafka options are required when async messaging is enabled")
	errConsumerHandlersRequired = errors.New("consumer cannot start before business event handlers are registered")
)

// AsyncRuntime 管理 Outbox Relay 和消息发布连接的生命周期。
type AsyncRuntime struct {
	runner asyncRunner
	closer io.Closer

	mu       sync.Mutex
	started  bool
	cancel   context.CancelFunc
	done     chan error
	finished chan struct{}
	runErr   error

	shutdownStarted bool
	shutdownDone    chan struct{}
	shutdownErr     error
}

func newAsyncRuntime(runner asyncRunner, closer io.Closer) *AsyncRuntime {
	return &AsyncRuntime{
		runner:       runner,
		closer:       closer,
		shutdownDone: make(chan struct{}),
	}
}

// NewAsyncRuntime 根据配置创建可选的 Outbox Relay 运行时。
func NewAsyncRuntime(cfg *Config, datastore store.IStore) (*AsyncRuntime, error) {
	return NewAsyncRuntimeWithRouter(cfg, datastore, nil)
}

// NewAsyncRuntimeWithRouter 创建异步运行时。Consumer 启用时必须传入至少注册了
// 一个业务 handler 的 router；生产应用应由业务垂直切片显式构造并注入该 router。
// 保留 NewAsyncRuntime 作为无 handler 的安全入口，防止配置误开 Consumer 后静默吞消息。
func NewAsyncRuntimeWithRouter(cfg *Config, datastore store.IStore, router *consumer.EventRouter) (*AsyncRuntime, error) {
	var kafkaFactory asyncKafkaFactory
	if cfg != nil && cfg.KafkaOptions != nil {
		kafkaFactory = kafkaOptionsFactory{options: cfg.KafkaOptions}
	}
	return newAsyncRuntimeWithKafkaFactory(cfg, datastore, router, kafkaFactory)
}

func newAsyncRuntimeWithKafkaFactory(
	cfg *Config,
	datastore store.IStore,
	router *consumer.EventRouter,
	kafkaFactory asyncKafkaFactory,
) (_ *AsyncRuntime, returnErr error) {
	outboxEnabled := cfg != nil && cfg.OutboxOptions != nil && cfg.OutboxOptions.Enabled
	consumerEnabled := cfg != nil && cfg.ConsumerOptions != nil && cfg.ConsumerOptions.Enabled
	if cfg == nil || (!outboxEnabled && !consumerEnabled) {
		return newAsyncRuntime(nil, nil), nil
	}
	if cfg.KafkaOptions == nil || kafkaFactory == nil {
		return nil, errKafkaOptionsRequired
	}
	// 当前应用尚未实现任何入站业务事件 handler。Consumer 必须 fail-closed，不能
	// 创建空 Router 后把所有合法业务事件当成 unsupported 并写入 DLQ。待首个消费
	// 垂直切片提供显式 handler 注册后，再移除此保护并在启动前校验 Router 非空。
	if consumerEnabled && !router.HasHandlers() {
		return nil, errConsumerHandlersRequired
	}

	var runners []asyncRunner
	var closers []io.Closer
	defer func() {
		if returnErr != nil {
			returnErr = errors.Join(returnErr, closeAll{closers: closers}.Close())
		}
	}()
	var observer *metrics.AsyncMetrics
	newObserver := func() (*metrics.AsyncMetrics, error) {
		if observer != nil {
			return observer, nil
		}
		var err error
		observer, err = metrics.NewAsyncMetrics(otel.Meter("async-runtime.metrics"))
		if err != nil {
			return nil, err
		}
		return observer, nil
	}
	if outboxEnabled {
		writer, err := kafkaFactory.Writer()
		if err != nil {
			return nil, fmt.Errorf("create Kafka outbox writer: %w", err)
		}
		closers = append(closers, writer)
		publisher, err := outbox.NewKafkaPublisher(writer)
		if err != nil {
			return nil, fmt.Errorf("create Kafka publisher: %w", err)
		}
		observer, metricsErr := newObserver()
		if metricsErr != nil {
			return nil, fmt.Errorf("create async metrics: %w", metricsErr)
		}
		relay, err := outbox.NewRelay(datastore.Outbox(), publisher, outbox.RelayOptions{
			BatchSize: cfg.OutboxOptions.BatchSize, PollInterval: cfg.OutboxOptions.PollInterval,
			LeaseDuration: cfg.OutboxOptions.LeaseDuration, RetryDelay: cfg.OutboxOptions.RetryDelay,
			MaxRetryDelay: cfg.OutboxOptions.MaxRetryDelay, MaxAttempts: cfg.OutboxOptions.MaxAttempts, Observer: observer,
		})
		if err != nil {
			return nil, fmt.Errorf("create outbox relay: %w", err)
		}
		runners = append(runners, relay)
	}
	if consumerEnabled {
		reader, err := kafkaFactory.Reader()
		if err != nil {
			return nil, fmt.Errorf("create Kafka consumer reader: %w", err)
		}
		closers = append(closers, reader)
		dlqWriter, err := kafkaFactory.WriterForTopic(cfg.ConsumerOptions.DLQTopic)
		if err != nil {
			return nil, fmt.Errorf("create Kafka DLQ writer: %w", err)
		}
		closers = append(closers, dlqWriter)
		storedDLQ, err := consumer.NewPersistentDeadLetterPublisher(datastore, datastore.DeadLetter(), datastore.Inbox(), cfg.KafkaOptions.ReaderOptions.GroupID, cfg.ConsumerOptions.DLQTopic)
		if err != nil {
			return nil, fmt.Errorf("create persistent DLQ publisher: %w", err)
		}
		metricsObserver, err := newObserver()
		if err != nil {
			return nil, fmt.Errorf("create consumer metrics: %w", err)
		}
		processor, err := consumer.NewInboxProcessor(datastore, datastore.Inbox(), router, consumer.InboxOptions{
			ConsumerName:  cfg.KafkaOptions.ReaderOptions.GroupID,
			LeaseDuration: cfg.ConsumerOptions.InboxLeaseDuration,
			RetryDelay:    cfg.ConsumerOptions.RetryDelay,
		})
		if err != nil {
			return nil, fmt.Errorf("create Inbox processor: %w", err)
		}
		worker, err := consumer.New(reader, processor, storedDLQ, consumer.Options{
			MaxAttempts:    cfg.ConsumerOptions.MaxAttempts,
			RetryDelay:     cfg.ConsumerOptions.RetryDelay,
			MaxRetryDelay:  cfg.ConsumerOptions.MaxRetryDelay,
			ProcessTimeout: cfg.ConsumerOptions.ProcessTimeout,
			Observer:       metricsObserver,
		})
		if err != nil {
			return nil, fmt.Errorf("create Kafka consumer: %w", err)
		}
		dlqPublisher, err := outbox.NewKafkaPublisher(dlqWriter)
		if err != nil {
			return nil, fmt.Errorf("create Kafka DLQ publisher: %w", err)
		}
		dlqRelay, err := outbox.NewRelay(datastore.DeadLetter(), dlqPublisher, outbox.RelayOptions{
			BatchSize: 100, PollInterval: time.Second,
			LeaseDuration: cfg.ConsumerOptions.InboxLeaseDuration,
			RetryDelay:    cfg.ConsumerOptions.RetryDelay,
			MaxRetryDelay: cfg.ConsumerOptions.MaxRetryDelay,
			MaxAttempts:   cfg.ConsumerOptions.MaxAttempts,
			Observer:      metricsObserver,
		})
		if err != nil {
			return nil, fmt.Errorf("create DLQ relay: %w", err)
		}
		runners = append(runners, worker, dlqRelay)
	}
	return newAsyncRuntime(multiRunner{runners: runners}, closeAll{closers: closers}), nil
}

type multiRunner struct{ runners []asyncRunner }

func (r multiRunner) Run(ctx context.Context) error {
	if len(r.runners) == 0 {
		<-ctx.Done()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	results := make(chan error, len(r.runners))
	for _, runner := range r.runners {
		wg.Add(1)
		go func(runner asyncRunner) { defer wg.Done(); results <- runner.Run(runCtx) }(runner)
	}
	var errs []error
	// 任一 worker 非正常退出都取消同组 worker，使 HTTP Server 能感知异步链路故障，
	// 避免 API 继续对外健康但 Outbox/Consumer 已停止工作的假健康状态。
	for range r.runners {
		err := <-results
		if err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
			cancel()
		}
	}
	wg.Wait()
	return errors.Join(errs...)
}

type closeAll struct{ closers []io.Closer }

func (c closeAll) Close() error {
	var errs []error
	for _, closer := range c.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Start 启动 Relay。重复调用不会创建额外 goroutine。
func (r *AsyncRuntime) Start(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runner == nil || r.started || r.shutdownStarted {
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	// done 携带 runner 结果供 Shutdown 汇总；finished 只广播完成状态，供 Server 的
	// select 感知后台 worker 是否提前退出，避免多个接收方竞争同一个错误值。
	r.done = make(chan error, 1)
	r.finished = make(chan struct{})
	r.started = true
	go func() {
		err := r.runner.Run(runCtx)
		r.mu.Lock()
		r.runErr = err
		r.mu.Unlock()
		r.done <- err
		close(r.done)
		close(r.finished)
	}()
}

// Done 返回异步运行时完成信号，可用于让服务感知后台 worker 的提前退出。
//
// 未启动或没有 runner 时返回 nil，刻意利用 nil channel 禁用 Server 的对应 select
// 分支；返回已关闭 channel 会让仅启用 HTTP 的服务立即退出。
func (r *AsyncRuntime) Done() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished == nil {
		return nil
	}
	return r.finished
}

// Err 返回异步运行时完成后的错误。
func (r *AsyncRuntime) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runErr
}

// Check 验证已配置的异步 worker 仍在运行。Kafka 连接状态由组合 readiness
// checker 单独探测；这里负责发现 worker 提前退出或关闭中的假健康。
func (r *AsyncRuntime) Check(context.Context) error {
	if r == nil {
		return errors.New("async runtime is not configured")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runner == nil {
		return nil
	}
	if r.shutdownStarted {
		return errors.New("async runtime is shutting down")
	}
	if !r.started {
		return errors.New("async runtime has not started")
	}
	if r.finished != nil {
		select {
		case <-r.finished:
			if r.runErr != nil {
				return fmt.Errorf("async runtime stopped: %w", r.runErr)
			}
			return errors.New("async runtime stopped unexpectedly")
		default:
		}
	}
	return nil
}

// Shutdown 先停止并等待 Relay，再关闭消息发布连接。
func (r *AsyncRuntime) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	if !r.shutdownStarted {
		r.shutdownStarted = true
		// 清理在后台持续执行：调用方 context 超时只停止本次等待，不能中断真实资源
		// 回收；后续 Shutdown 会继续等待同一次清理并得到相同结果。
		go r.finishShutdown(r.started, r.cancel, r.done, r.closer)
	}
	shutdownDone := r.shutdownDone
	r.mu.Unlock()

	select {
	case <-shutdownDone:
		return r.shutdownErr
	case <-ctx.Done():
		return fmt.Errorf("wait for async runtime: %w", ctx.Err())
	}
}

func (r *AsyncRuntime) finishShutdown(started bool, cancel context.CancelFunc, done <-chan error, closer io.Closer) {
	var runErr error
	if started {
		// 先取消 intake 并等待 worker 完成在途处理，再关闭 reader/writer。提前关闭连接会
		// 打断 DLQ 落库后的发布或业务成功后的 offset 提交，扩大重复投递窗口。
		cancel()
		runErr = <-done
	}

	var closeErr error
	if closer != nil {
		if err := closer.Close(); err != nil {
			closeErr = fmt.Errorf("close async publisher: %w", err)
		}
	}
	r.shutdownErr = errors.Join(runErr, closeErr)
	close(r.shutdownDone)
}

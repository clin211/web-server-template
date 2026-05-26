package job

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	genericoptions "github.com/clin211/gin-enterprise-template/pkg/options"
)

// Producer 定义异步任务入队的生产者接口。
type Producer interface {
	// Enqueue 将任务加入异步队列。
	Enqueue(context.Context, EnqueueRequest) (*EnqueueResult, error)
	// Close 释放生产者持有的资源。
	Close() error
}

// EnqueueRequest 表示一次异步任务入队请求。
type EnqueueRequest struct {
	// TaskType 是任务注册表中的任务类型。
	TaskType string
	// Payload 是任务处理器接收的 JSON 负载。
	Payload json.RawMessage
	// Queue 是任务投递到的队列名称。
	Queue string
	// TaskID 是外部指定的幂等任务 ID。
	TaskID string
	// ExecutionID 是本次调度执行记录 ID。
	ExecutionID string
	// ScheduledTaskID 是触发本次入队的定时任务 ID。
	ScheduledTaskID string
	// Metadata 是随任务传递的额外元数据。
	Metadata map[string]string
	// ProcessIn 表示任务延迟多久后开始处理。
	ProcessIn time.Duration
	// ProcessAt 表示任务在指定时间点开始处理。
	ProcessAt *time.Time
	// UniqueTTL 表示 Asynq 唯一任务锁的有效期。
	UniqueTTL time.Duration
	// Retention 表示任务元数据保留时长。
	Retention time.Duration
	// Timeout 表示单次任务处理超时时间。
	Timeout time.Duration
	// MaxRetry 表示覆盖任务定义的最大重试次数。
	MaxRetry *int
}

// EnqueueResult 表示任务成功入队后的返回信息。
type EnqueueResult struct {
	// TaskID 是 Asynq 返回的任务 ID。
	TaskID string
	// Queue 是任务实际进入的队列。
	Queue string
	// State 是任务入队后的 Asynq 状态。
	State string
}

// AsynqProducer 使用 Asynq 客户端实现 Producer。
type AsynqProducer struct {
	client    *asynq.Client
	registry  *Registry
	metrics   *Metrics
	retention time.Duration
}

// NewAsynqProducer 创建基于 Asynq 的任务生产者。
func NewAsynqProducer(rdb *redis.Client, registry *Registry, metrics *Metrics, opts *genericoptions.JobOptions) *AsynqProducer {
	var retention time.Duration
	if opts != nil {
		retention = opts.Async.DeadLetter.Retention
	}
	return &AsynqProducer{
		client:    asynq.NewClientFromRedisClient(rdb),
		registry:  registry,
		metrics:   metrics,
		retention: retention,
	}
}

// Enqueue 校验任务定义并将任务投递到 Asynq 队列。
func (p *AsynqProducer) Enqueue(ctx context.Context, req EnqueueRequest) (*EnqueueResult, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("job producer is not initialized")
	}
	if p.registry == nil {
		return nil, fmt.Errorf("job registry is not initialized")
	}

	def, queue, err := p.registry.ValidateEnqueue(ctx, req.TaskType, req.Payload, req.Queue)
	if err != nil {
		p.metrics.RecordEnqueue(ctx, req.TaskType, req.Queue, "validation_failed")
		return nil, err
	}

	envelopePayload, err := marshalEnvelope(ctx, req)
	if err != nil {
		p.metrics.RecordEnqueue(ctx, req.TaskType, queue, "marshal_failed")
		return nil, fmt.Errorf("marshal job payload: %w", err)
	}

	opts := []asynq.Option{asynq.Queue(queue)}
	maxRetry := def.RetryPolicy.MaxRetry
	if req.MaxRetry != nil {
		maxRetry = *req.MaxRetry
	}
	if maxRetry >= 0 {
		opts = append(opts, asynq.MaxRetry(maxRetry))
	}
	if req.TaskID != "" {
		opts = append(opts, asynq.TaskID(req.TaskID))
	}
	if req.ProcessIn > 0 {
		opts = append(opts, asynq.ProcessIn(req.ProcessIn))
	}
	if req.ProcessAt != nil {
		opts = append(opts, asynq.ProcessAt(*req.ProcessAt))
	}
	if req.UniqueTTL > 0 {
		opts = append(opts, asynq.Unique(req.UniqueTTL))
	}
	if req.Retention > 0 {
		opts = append(opts, asynq.Retention(req.Retention))
	} else if p.retention > 0 {
		opts = append(opts, asynq.Retention(p.retention))
	}
	if req.Timeout > 0 {
		opts = append(opts, asynq.Timeout(req.Timeout))
	} else if def.Timeout > 0 {
		opts = append(opts, asynq.Timeout(def.Timeout))
	}

	info, err := p.client.EnqueueContext(ctx, asynq.NewTask(req.TaskType, envelopePayload), opts...)
	if err != nil {
		p.metrics.RecordEnqueue(ctx, req.TaskType, queue, "failed")
		return nil, fmt.Errorf("enqueue job task %q: %w", req.TaskType, err)
	}

	p.metrics.RecordEnqueue(ctx, req.TaskType, queue, "succeeded")
	return &EnqueueResult{TaskID: info.ID, Queue: info.Queue, State: info.State.String()}, nil
}

// Close 实现 Producer 接口，当前复用外部 Redis 客户端无需额外关闭。
func (*AsynqProducer) Close() error {
	return nil
}

func marshalEnvelope(ctx context.Context, req EnqueueRequest) ([]byte, error) {
	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}

	metadata := make(map[string]string, len(req.Metadata)+4)
	maps.Copy(metadata, req.Metadata)
	if req.ExecutionID != "" {
		metadata[MetadataExecutionID] = req.ExecutionID
	}
	if req.ScheduledTaskID != "" {
		metadata[MetadataScheduledTaskID] = req.ScheduledTaskID
	}

	envelope := taskEnvelope{
		Payload:  payload,
		Trace:    InjectTraceContext(ctx),
		Metadata: metadata,
	}
	return json.Marshal(envelope)
}

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"
)

const (
	// VisibilityInternal 表示仅系统内部可调度的任务。
	VisibilityInternal Visibility = "internal"
	// VisibilityPublic 表示允许客户端通过 API 触发的任务。
	VisibilityPublic Visibility = "public"

	// DefaultQueue 是未显式指定队列时使用的默认队列。
	DefaultQueue = "default"
)

// Visibility 表示任务对外暴露的可见性。
type Visibility string

// HandlerFunc 定义异步任务的处理函数签名。
type HandlerFunc func(context.Context, *Task) error

// PayloadValidator 定义任务负载的校验函数签名。
type PayloadValidator func(context.Context, json.RawMessage) error

// Task 表示工作器执行时接收到的任务实例。
type Task struct {
	// Type 是任务注册表中的任务类型。
	Type string
	// Payload 是任务处理器接收的 JSON 负载。
	Payload json.RawMessage
	// ID 是队列系统分配或外部指定的任务 ID。
	ID string
	// Queue 是任务所在的队列名称。
	Queue string
	// ExecutionID 是本次调度执行记录 ID。
	ExecutionID string
	// ScheduledTaskID 是触发本次执行的定时任务 ID。
	ScheduledTaskID string
	// Metadata 是随任务传递的额外元数据。
	Metadata map[string]string
}

// RetryPolicy 描述任务失败后的重试策略。
type RetryPolicy struct {
	// MaxRetry 是任务允许的最大重试次数。
	MaxRetry int
}

// TaskDef 描述一种可注册的异步任务。
type TaskDef struct {
	// Type 是任务类型的全局唯一标识。
	Type string
	// Handler 是任务执行入口。
	Handler HandlerFunc
	// PayloadValidator 在任务入队前校验 JSON 负载。
	PayloadValidator PayloadValidator
	// DefaultQueue 是任务未指定队列时使用的默认队列。
	DefaultQueue string
	// AllowedQueues 是任务允许投递的队列集合。
	AllowedQueues []string
	// Permission 是客户端触发任务时需要的权限标识。
	Permission string
	// Visibility 决定任务是否允许由客户端触发。
	Visibility Visibility
	// MaxPayloadBytes 限制任务负载的最大字节数。
	MaxPayloadBytes int
	// Timeout 是单次任务处理超时时间。
	Timeout time.Duration
	// RetryPolicy 是任务失败后的重试策略。
	RetryPolicy RetryPolicy
}

// Registry 保存任务定义，并使用读写锁保证并发访问安全。
type Registry struct {
	mu    sync.RWMutex
	tasks map[string]TaskDef
}

// NewRegistry 创建空的任务注册表。
func NewRegistry() *Registry {
	return &Registry{tasks: map[string]TaskDef{}}
}

// Register 注册一个任务定义。
func (r *Registry) Register(def TaskDef) error {
	if def.Type == "" {
		return fmt.Errorf("job task type must not be empty")
	}
	if def.Handler == nil {
		return fmt.Errorf("job task %q handler must not be nil", def.Type)
	}
	if def.DefaultQueue == "" {
		def.DefaultQueue = DefaultQueue
	}
	if len(def.AllowedQueues) == 0 {
		def.AllowedQueues = []string{def.DefaultQueue}
	}
	if def.Visibility == "" {
		def.Visibility = VisibilityInternal
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[def.Type]; ok {
		return fmt.Errorf("job task %q already registered", def.Type)
	}
	r.tasks[def.Type] = def
	return nil
}

// MustRegister 注册任务定义，失败时触发 panic。
func (r *Registry) MustRegister(def TaskDef) {
	if err := r.Register(def); err != nil {
		panic(err)
	}
}

// Get 根据任务类型获取任务定义。
func (r *Registry) Get(taskType string) (TaskDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.tasks[taskType]
	return def, ok
}

// List 返回当前注册的全部任务定义副本。
func (r *Registry) List() []TaskDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]TaskDef, 0, len(r.tasks))
	for _, def := range r.tasks {
		defs = append(defs, def)
	}
	return defs
}

// ValidateEnqueue 校验任务类型、负载和队列，并返回最终队列名称。
func (r *Registry) ValidateEnqueue(ctx context.Context, taskType string, payload json.RawMessage, queue string) (TaskDef, string, error) {
	def, ok := r.Get(taskType)
	if !ok {
		return TaskDef{}, "", fmt.Errorf("job task type %q is not registered", taskType)
	}

	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if def.MaxPayloadBytes > 0 && len(payload) > def.MaxPayloadBytes {
		return TaskDef{}, "", fmt.Errorf("job task %q payload exceeds %d bytes", taskType, def.MaxPayloadBytes)
	}
	if !json.Valid(payload) {
		return TaskDef{}, "", fmt.Errorf("job task %q payload must be valid JSON", taskType)
	}

	if queue == "" {
		queue = def.DefaultQueue
	}
	if !slices.Contains(def.AllowedQueues, queue) {
		return TaskDef{}, "", fmt.Errorf("job task %q queue %q is not allowed", taskType, queue)
	}

	if def.PayloadValidator != nil {
		if err := def.PayloadValidator(ctx, payload); err != nil {
			return TaskDef{}, "", fmt.Errorf("job task %q payload validation failed: %w", taskType, err)
		}
	}

	return def, queue, nil
}

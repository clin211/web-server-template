package options

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*JobOptions)(nil)

// JobOptions 包含异步任务、定时调度和客户端任务的配置项。
type JobOptions struct {
	// Async 包含异步队列相关配置。
	Async AsyncJobOptions `json:"async" mapstructure:"async"`
	// Scheduler 包含 Cron 调度器相关配置。
	Scheduler SchedulerJobOptions `json:"scheduler" mapstructure:"scheduler"`
	// ClientTask 包含客户端自管理定时任务相关配置。
	ClientTask ClientTaskJobOptions `json:"client-task" mapstructure:"client-task"`
	// SystemTasks 包含随服务启动注册的系统定时任务配置。
	SystemTasks []SystemTaskJobOptions `json:"system-tasks" mapstructure:"system-tasks"`
}

// AsyncJobOptions 包含异步任务队列配置。
type AsyncJobOptions struct {
	// Enabled 表示是否启用异步任务队列。
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	// Worker 包含工作器并发和队列权重配置。
	Worker WorkerJobOptions `json:"worker" mapstructure:"worker"`
	// Retry 包含任务重试和超时配置。
	Retry RetryJobOptions `json:"retry" mapstructure:"retry"`
	// DeadLetter 包含死信和任务元数据保留配置。
	DeadLetter DeadLetterOptions `json:"dead-letter" mapstructure:"dead-letter"`
}

// WorkerJobOptions 包含异步任务工作器配置。
type WorkerJobOptions struct {
	// Concurrency 是工作器最大并发数。
	Concurrency int `json:"concurrency" mapstructure:"concurrency"`
	// StrictPriority 表示是否启用严格优先级队列处理。
	StrictPriority bool `json:"strict-priority" mapstructure:"strict-priority"`
	// Queues 表示队列名称到处理权重的映射。
	Queues map[string]int `json:"queues" mapstructure:"queues"`
}

// RetryJobOptions 包含任务重试与处理时限配置。
type RetryJobOptions struct {
	// MaxRetry 是任务最大重试次数。
	MaxRetry int `json:"max-retry" mapstructure:"max-retry"`
	// Timeout 是单次任务处理超时时间。
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
	// Deadline 是任务从入队起允许处理的总时限。
	Deadline time.Duration `json:"deadline" mapstructure:"deadline"`
}

// DeadLetterOptions 包含死信和任务元数据保留配置。
type DeadLetterOptions struct {
	// Retention 是任务完成或进入死信后的元数据保留时长。
	Retention time.Duration `json:"retention" mapstructure:"retention"`
}

// SchedulerJobOptions 包含定时调度器配置。
type SchedulerJobOptions struct {
	// Enabled 表示是否启用定时调度器。
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	// Timezone 是调度器默认使用的时区。
	Timezone string `json:"timezone" mapstructure:"timezone"`
	// LockTTL 是调度分布式锁的有效期。
	LockTTL time.Duration `json:"lock-ttl" mapstructure:"lock-ttl"`
	// ReconcileInterval 是客户端定时任务同步间隔。
	ReconcileInterval time.Duration `json:"reconcile-interval" mapstructure:"reconcile-interval"`
	// MinInterval 是客户端定时任务允许的最小调度间隔。
	MinInterval time.Duration `json:"min-interval" mapstructure:"min-interval"`
}

// ClientTaskJobOptions 包含客户端自管理定时任务配置。
type ClientTaskJobOptions struct {
	// Enabled 表示是否允许客户端管理定时任务。
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	// MaxTasksPerUser 是单个用户允许创建的最大定时任务数。
	MaxTasksPerUser int `json:"max-tasks-per-user" mapstructure:"max-tasks-per-user"`
	// MaxPayloadBytes 是客户端任务负载的最大字节数。
	MaxPayloadBytes int `json:"max-payload-bytes" mapstructure:"max-payload-bytes"`
	// AllowedQueues 是客户端任务允许投递的队列集合。
	AllowedQueues []string `json:"allowed-queues" mapstructure:"allowed-queues"`
}

// SystemTaskJobOptions 包含系统定时任务配置。
type SystemTaskJobOptions struct {
	// Name 是系统定时任务名称。
	Name string `json:"name" mapstructure:"name"`
	// CronExpr 是五段式 Cron 表达式。
	CronExpr string `json:"cron-expr" mapstructure:"cron-expr"`
	// TaskType 是要投递的异步任务类型。
	TaskType string `json:"task-type" mapstructure:"task-type"`
	// Queue 是任务投递目标队列。
	Queue string `json:"queue" mapstructure:"queue"`
	// Payload 是任务入队时携带的 JSON 负载。
	Payload map[string]any `json:"payload" mapstructure:"payload"`
}

// NewJobOptions 创建带默认值的任务配置。
func NewJobOptions() *JobOptions {
	return &JobOptions{
		Async: AsyncJobOptions{
			Enabled: false,
			Worker: WorkerJobOptions{
				Concurrency:    10,
				StrictPriority: false,
				Queues: map[string]int{
					"critical": 6,
					"default":  3,
					"low":      1,
				},
			},
			Retry: RetryJobOptions{
				MaxRetry: 3,
				Timeout:  30 * time.Second,
				Deadline: 5 * time.Minute,
			},
			DeadLetter: DeadLetterOptions{Retention: 168 * time.Hour},
		},
		Scheduler: SchedulerJobOptions{
			Enabled:           false,
			Timezone:          "Asia/Shanghai",
			LockTTL:           2 * time.Minute,
			ReconcileInterval: time.Minute,
			MinInterval:       time.Minute,
		},
		ClientTask: ClientTaskJobOptions{
			Enabled:         false,
			MaxTasksPerUser: 100,
			MaxPayloadBytes: 8192,
			AllowedQueues:   []string{"default", "low"},
		},
	}
}

// Validate 校验任务相关配置项。
func (o *JobOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}
	if o.Async.Enabled {
		if o.Async.Worker.Concurrency <= 0 {
			errs = append(errs, fmt.Errorf("job.async.worker.concurrency must be greater than 0"))
		}
		if len(o.Async.Worker.Queues) == 0 {
			errs = append(errs, fmt.Errorf("job.async.worker.queues must not be empty"))
		}
		for queue, weight := range o.Async.Worker.Queues {
			if queue == "" {
				errs = append(errs, fmt.Errorf("job.async.worker.queues contains empty queue name"))
			}
			if weight <= 0 {
				errs = append(errs, fmt.Errorf("job.async.worker.queues.%s must be greater than 0", queue))
			}
		}
		if o.Async.Retry.MaxRetry < 0 {
			errs = append(errs, fmt.Errorf("job.async.retry.max-retry must be greater than or equal to 0"))
		}
		if o.Async.Retry.Timeout <= 0 {
			errs = append(errs, fmt.Errorf("job.async.retry.timeout must be greater than 0"))
		}
		if o.Async.DeadLetter.Retention < 0 {
			errs = append(errs, fmt.Errorf("job.async.dead-letter.retention must be greater than or equal to 0"))
		}
	}

	if o.Scheduler.Enabled {
		if o.Scheduler.Timezone == "" {
			errs = append(errs, fmt.Errorf("job.scheduler.timezone must not be empty"))
		}
		if o.Scheduler.LockTTL <= 0 {
			errs = append(errs, fmt.Errorf("job.scheduler.lock-ttl must be greater than 0"))
		}
		if o.Scheduler.ReconcileInterval <= 0 {
			errs = append(errs, fmt.Errorf("job.scheduler.reconcile-interval must be greater than 0"))
		}
		if o.Scheduler.MinInterval <= 0 {
			errs = append(errs, fmt.Errorf("job.scheduler.min-interval must be greater than 0"))
		}
	}

	if o.ClientTask.Enabled {
		if o.ClientTask.MaxTasksPerUser <= 0 {
			errs = append(errs, fmt.Errorf("job.client-task.max-tasks-per-user must be greater than 0"))
		}
		if o.ClientTask.MaxPayloadBytes <= 0 {
			errs = append(errs, fmt.Errorf("job.client-task.max-payload-bytes must be greater than 0"))
		}
		if len(o.ClientTask.AllowedQueues) == 0 {
			errs = append(errs, fmt.Errorf("job.client-task.allowed-queues must not be empty"))
		}
	}

	return errs
}

// AddFlags 将任务相关命令行标志添加到指定 FlagSet。
func (o *JobOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.Async.Enabled, fullPrefix+".async.enabled", o.Async.Enabled, "Enable asynchronous job queue.")
	fs.IntVar(&o.Async.Worker.Concurrency, fullPrefix+".async.worker.concurrency", o.Async.Worker.Concurrency, "Maximum number of concurrent job workers.")
	fs.BoolVar(&o.Async.Worker.StrictPriority, fullPrefix+".async.worker.strict-priority", o.Async.Worker.StrictPriority, "Enable strict priority queue processing.")
	fs.StringToIntVar(&o.Async.Worker.Queues, fullPrefix+".async.worker.queues", o.Async.Worker.Queues, "Asynq queue weights.")
	fs.IntVar(&o.Async.Retry.MaxRetry, fullPrefix+".async.retry.max-retry", o.Async.Retry.MaxRetry, "Maximum retry count for async jobs.")
	fs.DurationVar(&o.Async.Retry.Timeout, fullPrefix+".async.retry.timeout", o.Async.Retry.Timeout, "Per-attempt job timeout.")
	fs.DurationVar(&o.Async.Retry.Deadline, fullPrefix+".async.retry.deadline", o.Async.Retry.Deadline, "Job processing deadline relative to enqueue time.")
	fs.DurationVar(&o.Async.DeadLetter.Retention, fullPrefix+".async.dead-letter.retention", o.Async.DeadLetter.Retention, "Retention for completed or dead-letter task metadata.")
	fs.BoolVar(&o.Scheduler.Enabled, fullPrefix+".scheduler.enabled", o.Scheduler.Enabled, "Enable cron scheduler.")
	fs.StringVar(&o.Scheduler.Timezone, fullPrefix+".scheduler.timezone", o.Scheduler.Timezone, "Timezone used by scheduler.")
	fs.DurationVar(&o.Scheduler.LockTTL, fullPrefix+".scheduler.lock-ttl", o.Scheduler.LockTTL, "Redis scheduler lock TTL.")
	fs.DurationVar(&o.Scheduler.ReconcileInterval, fullPrefix+".scheduler.reconcile-interval", o.Scheduler.ReconcileInterval, "Scheduler reconcile interval.")
	fs.DurationVar(&o.Scheduler.MinInterval, fullPrefix+".scheduler.min-interval", o.Scheduler.MinInterval, "Minimum client task schedule interval.")
	fs.BoolVar(&o.ClientTask.Enabled, fullPrefix+".client-task.enabled", o.ClientTask.Enabled, "Enable client managed scheduled tasks.")
	fs.IntVar(&o.ClientTask.MaxTasksPerUser, fullPrefix+".client-task.max-tasks-per-user", o.ClientTask.MaxTasksPerUser, "Maximum scheduled tasks per user.")
	fs.IntVar(&o.ClientTask.MaxPayloadBytes, fullPrefix+".client-task.max-payload-bytes", o.ClientTask.MaxPayloadBytes, "Maximum client task payload size in bytes.")
	fs.StringSliceVar(&o.ClientTask.AllowedQueues, fullPrefix+".client-task.allowed-queues", o.ClientTask.AllowedQueues, "Queues allowed for client managed scheduled tasks.")
}

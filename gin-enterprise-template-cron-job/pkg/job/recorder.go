package job

import (
	"context"
	"time"
)

// ExecutionRecorder 定义任务执行状态的记录接口。
type ExecutionRecorder interface {
	// MarkRunning 标记任务开始执行。
	MarkRunning(context.Context, *Task) error
	// MarkRetrying 标记任务将进入重试。
	MarkRetrying(context.Context, *Task, error) error
	// MarkSucceeded 标记任务执行成功。
	MarkSucceeded(context.Context, *Task, time.Duration) error
	// MarkFailed 标记任务执行失败。
	MarkFailed(context.Context, *Task, error, time.Duration) error
	// MarkDead 标记任务进入死信状态。
	MarkDead(context.Context, *Task, error, time.Duration) error
}

// NoopExecutionRecorder 是不持久化执行状态的空实现。
type NoopExecutionRecorder struct{}

// MarkRunning 实现 ExecutionRecorder 接口。
func (NoopExecutionRecorder) MarkRunning(context.Context, *Task) error {
	return nil
}

// MarkRetrying 实现 ExecutionRecorder 接口。
func (NoopExecutionRecorder) MarkRetrying(context.Context, *Task, error) error {
	return nil
}

// MarkSucceeded 实现 ExecutionRecorder 接口。
func (NoopExecutionRecorder) MarkSucceeded(context.Context, *Task, time.Duration) error {
	return nil
}

// MarkFailed 实现 ExecutionRecorder 接口。
func (NoopExecutionRecorder) MarkFailed(context.Context, *Task, error, time.Duration) error {
	return nil
}

// MarkDead 实现 ExecutionRecorder 接口。
func (NoopExecutionRecorder) MarkDead(context.Context, *Task, error, time.Duration) error {
	return nil
}

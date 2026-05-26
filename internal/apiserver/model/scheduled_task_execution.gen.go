package model

import (
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/store/registry"
)

const TableNameScheduledTaskExecutionM = "scheduled_task_execution"

type ScheduledTaskExecutionM struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	ExecutionID     string     `gorm:"column:execution_id;not null;default:gen_random_uuid();uniqueIndex:idx_execution_id" json:"executionID"`
	ScheduledTaskID string     `gorm:"column:scheduled_task_id;not null;size:128;uniqueIndex:idx_task_scheduled_at;index:idx_execution_task_id" json:"scheduledTaskID"`
	UserID          string     `gorm:"column:user_id;not null;size:64;index:idx_execution_user_id" json:"userID"`
	TriggerType     string     `gorm:"column:trigger_type;not null;size:32" json:"triggerType"`
	ScheduledAt     time.Time  `gorm:"column:scheduled_at;not null;uniqueIndex:idx_task_scheduled_at" json:"scheduledAt"`
	EnqueuedAt      *time.Time `gorm:"column:enqueued_at" json:"enqueuedAt"`
	AsynqTaskID     *string    `gorm:"column:asynq_task_id;size:128" json:"asynqTaskID"`
	DispatchStatus  string     `gorm:"column:dispatch_status;not null;size:32;index:idx_dispatch_status" json:"dispatchStatus"`
	ProcessStatus   string     `gorm:"column:process_status;not null;size:32;index:idx_process_status" json:"processStatus"`
	Attempt         int32      `gorm:"column:attempt;not null;default:0" json:"attempt"`
	ErrorMsg        *string    `gorm:"column:error_msg;size:1024" json:"errorMsg"`
	StartedAt       *time.Time `gorm:"column:started_at" json:"startedAt"`
	FinishedAt      *time.Time `gorm:"column:finished_at" json:"finishedAt"`
	DurationMs      int64      `gorm:"column:duration_ms;not null;default:0" json:"durationMs"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null;default:current_timestamp" json:"updatedAt"`
}

func (*ScheduledTaskExecutionM) TableName() string {
	return TableNameScheduledTaskExecutionM
}

func init() {
	registry.Register(&ScheduledTaskExecutionM{})
}

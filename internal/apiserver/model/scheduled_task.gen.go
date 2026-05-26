package model

import (
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/store/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const TableNameScheduledTaskM = "scheduled_task"

type ScheduledTaskM struct {
	ID              int64          `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	ScheduledTaskID string         `gorm:"column:scheduled_task_id;not null;default:gen_random_uuid();uniqueIndex:idx_scheduled_task_id" json:"scheduledTaskID"`
	Name            string         `gorm:"column:name;not null;size:128" json:"name"`
	TaskType        string         `gorm:"column:task_type;not null;size:128;index:idx_scheduled_task_type" json:"taskType"`
	Payload         datatypes.JSON `gorm:"column:payload;type:jsonb;not null;default:'{}'" json:"payload"`
	CronExpr        string         `gorm:"column:cron_expr;not null;size:64" json:"cronExpr"`
	Queue           string         `gorm:"column:queue;not null;size:64" json:"queue"`
	Enabled         bool           `gorm:"column:enabled;not null;default:true;index:idx_scheduled_task_enabled_next_run" json:"enabled"`
	Timezone        string         `gorm:"column:timezone;not null;size:64" json:"timezone"`
	UserID          string         `gorm:"column:user_id;not null;size:64;index:idx_scheduled_task_user_id" json:"userID"`
	NextRunTime     *time.Time     `gorm:"column:next_run_time;index:idx_scheduled_task_enabled_next_run" json:"nextRunTime"`
	LastScheduledAt *time.Time     `gorm:"column:last_scheduled_at" json:"lastScheduledAt"`
	LastExecutionID *string        `gorm:"column:last_execution_id;size:128" json:"lastExecutionID"`
	LastError       *string        `gorm:"column:last_error;size:512" json:"lastError"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;default:current_timestamp" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deletedAt"`
}

func (*ScheduledTaskM) TableName() string {
	return TableNameScheduledTaskM
}

func init() {
	registry.Register(&ScheduledTaskM{})
}

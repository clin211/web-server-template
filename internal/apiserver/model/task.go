package model

import (
	"gorm.io/datatypes"
	"time"
)

// TaskM is an immutable task intent with fenced mutable execution state.
type TaskM struct {
	ID                int64          `json:"-"`
	TaskID            string         `json:"task_id"`
	TaskType          string         `json:"task_type"`
	Params            datatypes.JSON `json:"params"`
	ParamsVersion     int            `json:"params_version"`
	Source            string         `json:"source"`
	ScheduleID        *string        `json:"schedule_id,omitempty"`
	ScheduleVersion   *int64         `json:"schedule_version,omitempty"`
	ScheduledAt       *time.Time     `json:"scheduled_at,omitempty"`
	RetryOf           *string        `json:"retry_of,omitempty"`
	OwnerID           string         `json:"owner_id"`
	Status            string         `json:"status"`
	Priority          int            `json:"priority"`
	AvailableAt       time.Time      `json:"available_at"`
	TimeoutSeconds    int            `json:"timeout_seconds"`
	MaxAttempts       int            `json:"max_attempts"`
	Attempts          int            `json:"attempts"`
	FencingToken      int64          `json:"-"`
	WorkerID          *string        `json:"worker_id,omitempty"`
	LeaseUntil        *time.Time     `json:"lease_until,omitempty"`
	CancelRequestedAt *time.Time     `json:"cancel_requested_at,omitempty"`
	IdempotencyKey    *string        `json:"-"`
	RequestHash       string         `json:"-"`
	ResultID          *string        `json:"result_id,omitempty"`
	ErrorCode         *string        `json:"error_code,omitempty"`
	ErrorSummary      *string        `json:"error_summary,omitempty"`
	TraceParent       *string        `json:"-" gorm:"column:traceparent"`
	TraceID           *string        `json:"trace_id,omitempty"`
	RequestID         *string        `json:"request_id,omitempty"`
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	FinishedAt        *time.Time     `json:"finished_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// TableName returns the migration-managed table name.
func (*TaskM) TableName() string { return "task" }

// ScheduleM stores the durable schedule cursor.
type ScheduleM struct {
	ID                int64          `json:"-"`
	ScheduleID        string         `json:"schedule_id"`
	Name              string         `json:"name"`
	TaskType          string         `json:"task_type"`
	Params            datatypes.JSON `json:"params"`
	ParamsVersion     int            `json:"params_version"`
	Cron              string         `json:"cron"`
	Timezone          string         `json:"timezone"`
	Enabled           bool           `json:"enabled"`
	Archived          bool           `json:"archived"`
	NextRunAt         time.Time      `json:"next_run_at"`
	MisfirePolicy     string         `json:"misfire_policy"`
	ConcurrencyPolicy string         `json:"concurrency_policy"`
	OwnerID           string         `json:"owner_id"`
	Version           int64          `json:"version"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// TableName returns the migration-managed table name.
func (*ScheduleM) TableName() string { return "schedule" }

// TaskAttemptM retains every worker claim and its outcome.
type TaskAttemptM struct {
	ID           int64      `json:"-"`
	TaskID       string     `json:"task_id"`
	Attempt      int        `json:"attempt"`
	WorkerID     string     `json:"worker_id"`
	FencingToken int64      `json:"-"`
	LeaseUntil   time.Time  `json:"lease_until"`
	HeartbeatAt  time.Time  `json:"heartbeat_at"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Outcome      *string    `json:"outcome,omitempty"`
	ErrorCode    *string    `json:"error_code,omitempty"`
	TraceID      *string    `json:"trace_id,omitempty"`
}

// TableName returns the migration-managed table name.
func (*TaskAttemptM) TableName() string { return "task_attempt" }

// ReportResultM is a bounded shared result, never included in task list responses.
type ReportResultM struct {
	ID                        int64
	ResultID, TaskID, OwnerID string
	Content                   []byte
	ContentType               string
	ExpiresAt, CreatedAt      time.Time
}

// TableName returns the migration-managed table name.
func (*ReportResultM) TableName() string { return "report_result" }

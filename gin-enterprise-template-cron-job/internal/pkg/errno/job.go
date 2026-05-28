package errno

import "github.com/clin211/gin-enterprise-template/pkg/errorsx"

var (
	// ErrScheduledTaskNotFound means the requested scheduled task does not exist.
	ErrScheduledTaskNotFound = errorsx.NewCompat(404, "ScheduledTask.NotFound", "Scheduled task not found.")
	// ErrScheduledTaskAlreadyExists means a scheduled task with the same identity already exists.
	ErrScheduledTaskAlreadyExists = errorsx.NewCompat(409, "ScheduledTask.AlreadyExists", "Scheduled task already exists.")
	// ErrScheduledTaskInvalidCronExpr means the cron expression is invalid.
	ErrScheduledTaskInvalidCronExpr = errorsx.NewCompat(400, "ScheduledTask.InvalidCronExpr", "Cron expression is invalid.")
	// ErrScheduledTaskInvalidPayload means the task payload is invalid.
	ErrScheduledTaskInvalidPayload = errorsx.NewCompat(400, "ScheduledTask.InvalidPayload", "Payload is invalid.")
	// ErrScheduledTaskTaskTypeNotSupported means the task type is not public or not registered.
	ErrScheduledTaskTaskTypeNotSupported = errorsx.NewCompat(400, "ScheduledTask.TaskTypeNotSupported", "Task type is not supported.")
	// ErrScheduledTaskQueueNotAllowed means the requested queue is not allowed.
	ErrScheduledTaskQueueNotAllowed = errorsx.NewCompat(400, "ScheduledTask.QueueNotAllowed", "Queue is not allowed.")
	// ErrScheduledTaskQuotaExceeded means the user has reached the scheduled task limit.
	ErrScheduledTaskQuotaExceeded = errorsx.NewCompat(429, "ScheduledTask.QuotaExceeded", "Scheduled task quota exceeded.")
	// ErrScheduledTaskPermissionDenied means the user cannot access the scheduled task.
	ErrScheduledTaskPermissionDenied = errorsx.NewCompat(403, "ScheduledTask.PermissionDenied", "Permission denied for scheduled task.")
	// ErrScheduledTaskEnqueueFailed means the task execution could not be enqueued.
	ErrScheduledTaskEnqueueFailed = errorsx.NewCompat(500, "ScheduledTask.EnqueueFailed", "Failed to enqueue scheduled task.")
)

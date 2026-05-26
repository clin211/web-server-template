package validation

import (
	"context"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

// ValidateCreateScheduledTaskRequest validates the create request.
func (v *Validator) ValidateCreateScheduledTaskRequest(ctx context.Context, rq *v1.CreateScheduledTaskRequest) error {
	if rq.GetName() == "" || len(rq.GetName()) > 128 {
		return errno.ErrInvalidArgument.WithMessage("name must be between 1 and 128 characters")
	}
	if rq.GetTaskType() == "" {
		return errno.ErrScheduledTaskTaskTypeNotSupported
	}
	if rq.GetCronExpr() == "" {
		return errno.ErrScheduledTaskInvalidCronExpr
	}
	if err := genericjob.ValidateCronExpr(rq.GetCronExpr()); err != nil {
		return errno.ErrScheduledTaskInvalidCronExpr.WithMessage("%s", err.Error())
	}
	if rq.GetTimezone() == "" {
		return errno.ErrInvalidArgument.WithMessage("timezone cannot be empty")
	}
	if _, err := time.LoadLocation(rq.GetTimezone()); err != nil {
		return errno.ErrInvalidArgument.WithMessage("timezone is invalid")
	}
	return nil
}

// ValidateUpdateScheduledTaskRequest validates the update request.
func (v *Validator) ValidateUpdateScheduledTaskRequest(ctx context.Context, rq *v1.UpdateScheduledTaskRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	if rq.Name != nil && (rq.GetName() == "" || len(rq.GetName()) > 128) {
		return errno.ErrInvalidArgument.WithMessage("name must be between 1 and 128 characters")
	}
	if rq.CronExpr != nil {
		if err := genericjob.ValidateCronExpr(rq.GetCronExpr()); err != nil {
			return errno.ErrScheduledTaskInvalidCronExpr.WithMessage("%s", err.Error())
		}
	}
	if rq.Timezone != nil {
		if _, err := time.LoadLocation(rq.GetTimezone()); err != nil {
			return errno.ErrInvalidArgument.WithMessage("timezone is invalid")
		}
	}
	return nil
}

// ValidateDeleteScheduledTaskRequest validates the delete request.
func (v *Validator) ValidateDeleteScheduledTaskRequest(ctx context.Context, rq *v1.DeleteScheduledTaskRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	return nil
}

// ValidateGetScheduledTaskRequest validates the get request.
func (v *Validator) ValidateGetScheduledTaskRequest(ctx context.Context, rq *v1.GetScheduledTaskRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	return nil
}

// ValidateListScheduledTasksRequest validates the list request.
func (v *Validator) ValidateListScheduledTasksRequest(ctx context.Context, rq *v1.ListScheduledTasksRequest) error {
	return nil
}

// ValidateToggleScheduledTaskRequest validates the toggle request.
func (v *Validator) ValidateToggleScheduledTaskRequest(ctx context.Context, rq *v1.ToggleScheduledTaskRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	return nil
}

// ValidateTriggerScheduledTaskRequest validates the trigger request.
func (v *Validator) ValidateTriggerScheduledTaskRequest(ctx context.Context, rq *v1.TriggerScheduledTaskRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	return nil
}

// ValidateListScheduledTaskExecutionsRequest validates the list executions request.
func (v *Validator) ValidateListScheduledTaskExecutionsRequest(ctx context.Context, rq *v1.ListScheduledTaskExecutionsRequest) error {
	if rq.GetScheduledTaskID() == "" {
		return errno.ErrInvalidArgument.WithMessage("scheduledTaskID cannot be empty")
	}
	return nil
}

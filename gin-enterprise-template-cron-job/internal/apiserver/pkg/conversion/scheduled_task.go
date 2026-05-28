package conversion

import (
	"encoding/json"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ScheduledTaskModelToScheduledTaskV1 converts a database model to a protobuf message.
func ScheduledTaskModelToScheduledTaskV1(task *model.ScheduledTaskM) *v1.ScheduledTask {
	if task == nil {
		return nil
	}
	return &v1.ScheduledTask{
		ScheduledTaskID: task.ScheduledTaskID,
		Name:            task.Name,
		TaskType:        task.TaskType,
		Payload:         jsonToStruct(task.Payload),
		CronExpr:        task.CronExpr,
		Queue:           task.Queue,
		Enabled:         task.Enabled,
		Timezone:        task.Timezone,
		UserID:          task.UserID,
		NextRunTime:     unixPtr(task.NextRunTime),
		LastScheduledAt: unixPtr(task.LastScheduledAt),
		LastExecutionID: stringPtrValue(task.LastExecutionID),
		LastError:       stringPtrValue(task.LastError),
		CreatedAt:       task.CreatedAt.Unix(),
		UpdatedAt:       task.UpdatedAt.Unix(),
	}
}

// ScheduledTaskModelListToScheduledTaskV1List converts a list of models to protobuf messages.
func ScheduledTaskModelListToScheduledTaskV1List(tasks []*model.ScheduledTaskM) []*v1.ScheduledTask {
	result := make([]*v1.ScheduledTask, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, ScheduledTaskModelToScheduledTaskV1(task))
	}
	return result
}

// ScheduledTaskExecutionModelToExecutionV1 converts an execution model to a protobuf message.
func ScheduledTaskExecutionModelToExecutionV1(execution *model.ScheduledTaskExecutionM) *v1.ScheduledTaskExecution {
	if execution == nil {
		return nil
	}
	return &v1.ScheduledTaskExecution{
		ExecutionID:     execution.ExecutionID,
		ScheduledTaskID: execution.ScheduledTaskID,
		UserID:          execution.UserID,
		TriggerType:     execution.TriggerType,
		ScheduledAt:     execution.ScheduledAt.Unix(),
		EnqueuedAt:      unixPtr(execution.EnqueuedAt),
		AsynqTaskID:     stringPtrValue(execution.AsynqTaskID),
		DispatchStatus:  execution.DispatchStatus,
		ProcessStatus:   execution.ProcessStatus,
		Attempt:         execution.Attempt,
		ErrorMsg:        stringPtrValue(execution.ErrorMsg),
		StartedAt:       unixPtr(execution.StartedAt),
		FinishedAt:      unixPtr(execution.FinishedAt),
		DurationMs:      execution.DurationMs,
		CreatedAt:       execution.CreatedAt.Unix(),
		UpdatedAt:       execution.UpdatedAt.Unix(),
	}
}

// ScheduledTaskExecutionModelListToExecutionV1List converts a list of execution models to protobuf messages.
func ScheduledTaskExecutionModelListToExecutionV1List(executions []*model.ScheduledTaskExecutionM) []*v1.ScheduledTaskExecution {
	result := make([]*v1.ScheduledTaskExecution, 0, len(executions))
	for _, execution := range executions {
		result = append(result, ScheduledTaskExecutionModelToExecutionV1(execution))
	}
	return result
}

// StructToJSON converts a protobuf Struct to JSON bytes.
func StructToJSON(payload *structpb.Struct) []byte {
	if payload == nil {
		return []byte("{}")
	}
	data, err := payload.MarshalJSON()
	if err != nil {
		return []byte("{}")
	}
	return data
}

// jsonToStruct converts JSON bytes to a protobuf Struct.
func jsonToStruct(data []byte) *structpb.Struct {
	if len(data) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	value, err := structpb.NewStruct(payload)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return value
}

// unixPtr converts a time pointer to Unix timestamp.
func unixPtr(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

// stringPtrValue safely dereferences a string pointer.
func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

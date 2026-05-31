package scheduled_task

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

type stubClientTaskScheduler struct {
	called bool
	task   genericjob.SystemTask
}

func (s *stubClientTaskScheduler) RegisterClientTask(_ context.Context, task genericjob.SystemTask) error {
	s.called = true
	s.task = task
	return nil
}

func (*stubClientTaskScheduler) UnregisterTask(context.Context, string) {}

func TestRegisterSchedulerTask(t *testing.T) {
	updatedAt := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	baseTask := &model.ScheduledTaskM{
		ScheduledTaskID: "task-1",
		TaskType:        "demo.task",
		Payload:         `{"count":1,"name":"demo"}`,
		CronExpr:        "*/5 * * * *",
		Queue:           "critical",
		Enabled:         true,
		Timezone:        "Asia/Shanghai",
		UserID:          "user-1",
		UpdatedAt:       updatedAt,
	}

	tests := []struct {
		name       string
		scheduler  ClientTaskScheduler
		task       *model.ScheduledTaskM
		wantCalled bool
		wantTask   genericjob.SystemTask
	}{
		{
			name:       "skip when scheduler is nil",
			scheduler:  nil,
			task:       baseTask,
			wantCalled: false,
		},
		{
			name:       "skip when task disabled",
			scheduler:  &stubClientTaskScheduler{},
			task:       &model.ScheduledTaskM{Enabled: false},
			wantCalled: false,
		},
		{
			name:       "propagate scheduler fields",
			scheduler:  &stubClientTaskScheduler{},
			task:       baseTask,
			wantCalled: true,
			wantTask: genericjob.SystemTask{
				Name:      "task-1",
				CronExpr:  "*/5 * * * *",
				TaskType:  "demo.task",
				Queue:     "critical",
				Payload:   map[string]any{"count": float64(1), "name": "demo"},
				Enabled:   true,
				Timezone:  "Asia/Shanghai",
				UserID:    "user-1",
				UpdatedAt: updatedAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub, _ := tt.scheduler.(*stubClientTaskScheduler)
			if err := registerSchedulerTask(context.Background(), tt.scheduler, tt.task); err != nil {
				t.Fatalf("registerSchedulerTask() error = %v", err)
			}
			if stub == nil {
				return
			}
			if stub.called != tt.wantCalled {
				t.Fatalf("RegisterClientTask called = %v, want %v", stub.called, tt.wantCalled)
			}
			if !tt.wantCalled {
				return
			}
			if !reflect.DeepEqual(stub.task.Payload, tt.wantTask.Payload) {
				t.Fatalf("payload = %#v, want %#v", stub.task.Payload, tt.wantTask.Payload)
			}
			stub.task.Payload = nil
			tt.wantTask.Payload = nil
			if !reflect.DeepEqual(stub.task, tt.wantTask) {
				t.Fatalf("task = %#v, want %#v", stub.task, tt.wantTask)
			}
		})
	}
}

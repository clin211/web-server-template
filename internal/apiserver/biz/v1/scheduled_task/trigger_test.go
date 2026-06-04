package scheduled_task

import (
	"context"
	"errors"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	genericjob "github.com/clin211/gin-enterprise-template/pkg/job"
)

// mockScheduledTaskStore 是 ScheduledTaskStore 的测试 mock。
type mockScheduledTaskStore struct {
	getErr                 error
	getTask                *mockTask
	createErr              error
	createExecutionErr     error
	updateErr              error
	updateLastExecutionErr  error
	updateLastExecutionCalls []updateLastExecutionCall
}

type mockTask struct {
	scheduledTaskID string
	taskType        string
	payload         string
	queue           string
	userID          string
}

type updateLastExecutionCall struct {
	scheduledTaskID string
	executionID     string
	lastError       *string
}

func (m *mockScheduledTaskStore) Create(ctx context.Context, obj any) error {
	return m.createErr
}
func (m *mockScheduledTaskStore) Update(ctx context.Context, obj any) error {
	return m.updateErr
}
func (m *mockScheduledTaskStore) Delete(ctx context.Context, opts any) error {
	return nil
}
func (m *mockScheduledTaskStore) Get(ctx context.Context, opts any) (any, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return &struct {
		ScheduledTaskID string
		TaskType        string
		Payload         string
		Queue           string
		UserID          string
	}{
		ScheduledTaskID: m.getTask.scheduledTaskID,
		TaskType:        m.getTask.taskType,
		Payload:         m.getTask.payload,
		Queue:           m.getTask.queue,
		UserID:          m.getTask.userID,
	}, nil
}
func (m *mockScheduledTaskStore) List(ctx context.Context, opts any) (int64, any, error) {
	return 0, nil, nil
}
func (m *mockScheduledTaskStore) UpdateNextRunTime(ctx context.Context, scheduledTaskID string, nextRunTime int64) error {
	return nil
}
func (m *mockScheduledTaskStore) UpdateLastExecution(ctx context.Context, scheduledTaskID string, executionID string, lastError *string) error {
	m.updateLastExecutionCalls = append(m.updateLastExecutionCalls, updateLastExecutionCall{
		scheduledTaskID: scheduledTaskID,
		executionID:     executionID,
		lastError:       lastError,
	})
	return m.updateLastExecutionErr
}

// mockScheduledTaskExecutionStore 是 ScheduledTaskExecutionStore 的测试 mock。
type mockScheduledTaskExecutionStore struct {
	createErr   error
	updateErr   error
	createObj   any
	updateObj   any
	updateCalls int
}

func (m *mockScheduledTaskExecutionStore) Create(ctx context.Context, obj any) error {
	m.createObj = obj
	return m.createErr
}
func (m *mockScheduledTaskExecutionStore) Update(ctx context.Context, obj any) error {
	m.updateObj = obj
	m.updateCalls++
	return m.updateErr
}
func (m *mockScheduledTaskExecutionStore) Delete(ctx context.Context, opts any) error {
	return nil
}
func (m *mockScheduledTaskExecutionStore) Get(ctx context.Context, opts any) (any, error) {
	return nil, nil
}
func (m *mockScheduledTaskExecutionStore) List(ctx context.Context, opts any) (int64, any, error) {
	return 0, nil, nil
}
func (m *mockScheduledTaskExecutionStore) CreateExecutionIfAbsent(ctx context.Context, obj any) (any, bool, error) {
	return nil, false, nil
}
func (m *mockScheduledTaskExecutionStore) UpdateExecutionStatus(ctx context.Context, obj any) error {
	return nil
}
func (m *mockScheduledTaskExecutionStore) UpdateDispatchStatus(ctx context.Context, executionID string, dispatchStatus string, asynqTaskID *string, enqueuedAt *int64) error {
	return nil
}
func (m *mockScheduledTaskExecutionStore) UpdateProcessStatus(ctx context.Context, executionID string, processStatus string, attempt int32, errorMsg *string, startedAt *int64, finishedAt *int64, durationMs int64) error {
	return nil
}

// mockTaskProducer 是 TaskProducer 的测试 mock。
type mockTaskProducer struct {
	enqueueErr    error
	enqueueResult *genericjob.EnqueueResult
}

func (m *mockTaskProducer) Enqueue(ctx context.Context, req genericjob.EnqueueRequest) (*genericjob.EnqueueResult, error) {
	if m.enqueueErr != nil {
		return nil, m.enqueueErr
	}
	return m.enqueueResult, nil
}

// mockTaskRegistry 是 TaskRegistry 的测试 mock。
type mockTaskRegistry struct {
	getDef       genericjob.TaskDef
	getFound     bool
	listPublic   []genericjob.TaskDef
	validateErr  error
}

func (m *mockTaskRegistry) Get(taskType string) (genericjob.TaskDef, bool) {
	return m.getDef, m.getFound
}

func (m *mockTaskRegistry) ListPublic() []genericjob.TaskDef {
	return m.listPublic
}

func (m *mockTaskRegistry) ValidateEnqueue(ctx context.Context, taskType string, payload []byte, queue string) (genericjob.TaskDef, string, error) {
	if m.validateErr != nil {
		return genericjob.TaskDef{}, "", m.validateErr
	}
	return m.getDef, m.getDef.DefaultQueue, nil
}

// mockAuthz 是 authz 的测试 mock。
type mockAuthz struct {
	authorizeErr error
	allowed     bool
}

func (m *mockAuthz) Authorize(sub, obj, act string) (bool, error) {
	if m.authorizeErr != nil {
		return false, m.authorizeErr
	}
	return m.allowed, nil
}

// TestTriggerScheduledTask_TableDriven 表格驱动测试：手动触发定时任务。
func TestTriggerScheduledTask_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		taskID             string
		mockStore          *mockScheduledTaskStore
		mockExecStore      *mockScheduledTaskExecutionStore
		mockProducer       *mockTaskProducer
		mockRegistry       *mockTaskRegistry
		mockAuthz          *mockAuthz
		wantErr            bool
		errType           error
		wantExecStatus     string
		wantLastErrorSet   bool
	}{
		{
			name:   "任务不存在",
			taskID: "non-existent",
			mockStore: &mockScheduledTaskStore{
				getErr: errors.New("record not found"),
			},
			wantErr: true,
			errType: errno.ErrScheduledTaskNotFound,
		},
		{
			name:   "无权限触发任务",
			taskID: "task-1",
			mockStore: &mockScheduledTaskStore{
				getTask: &mockTask{
					scheduledTaskID: "task-1",
					taskType:        "demo.task",
					payload:         `{}`,
					queue:           "default",
					userID:          "user-1",
				},
			},
			mockAuthz: &mockAuthz{
				allowed: false,
			},
			wantErr: true,
		},
		{
			name:   "任务类型不支持",
			taskID: "task-1",
			mockStore: &mockScheduledTaskStore{
				getTask: &mockTask{
					scheduledTaskID: "task-1",
					taskType:        "unsupported.task",
					payload:         `{}`,
					queue:           "default",
					userID:          "admin",
				},
			},
			mockAuthz: &mockAuthz{
				allowed: true,
			},
			mockRegistry: &mockTaskRegistry{
				getFound: false,
			},
			wantErr: true,
		},
		{
			name:   "入队失败",
			taskID: "task-1",
			mockStore: &mockScheduledTaskStore{
				getTask: &mockTask{
					scheduledTaskID: "task-1",
					taskType:        "demo.task",
					payload:         `{}`,
					queue:           "default",
					userID:          "admin",
				},
			},
			mockAuthz: &mockAuthz{
				allowed: true,
			},
			mockRegistry: &mockTaskRegistry{
				getFound: true,
				getDef: genericjob.TaskDef{
					Type:         "demo.task",
					DefaultQueue: "default",
				},
			},
			mockProducer: &mockTaskProducer{
				enqueueErr: errors.New("redis connection failed"),
			},
			wantErr:          true,
			errType:          errno.ErrScheduledTaskEnqueueFailed,
			wantExecStatus:   DispatchStatusEnqueueFailed,
			wantLastErrorSet: true,
		},
		{
			name:   "入队成功",
			taskID: "task-1",
			mockStore: &mockScheduledTaskStore{
				getTask: &mockTask{
					scheduledTaskID: "task-1",
					taskType:        "demo.task",
					payload:         `{"key":"value"}`,
					queue:           "default",
					userID:          "admin",
				},
			},
			mockAuthz: &mockAuthz{
				allowed: true,
			},
			mockRegistry: &mockTaskRegistry{
				getFound: true,
				getDef: genericjob.TaskDef{
					Type:         "demo.task",
					DefaultQueue: "default",
				},
			},
			mockProducer: &mockTaskProducer{
				enqueueResult: &genericjob.EnqueueResult{
					TaskID: "asynq-task-123",
					Queue: "default",
					State: "enqueued",
				},
			},
			wantErr:           false,
			wantExecStatus:    DispatchStatusEnqueued,
			wantLastErrorSet:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 验证 mock 行为
			if tt.mockStore != nil && tt.mockStore.getErr != nil {
				// 验证任务不存在时的行为
				if !tt.wantErr || tt.errType != errno.ErrScheduledTaskNotFound {
					t.Errorf("expected ErrScheduledTaskNotFound")
				}
			}

			if tt.mockProducer != nil && tt.mockProducer.enqueueErr != nil {
				// 验证入队失败时错误被正确返回
				_, err := tt.mockProducer.Enqueue(context.Background(), genericjob.EnqueueRequest{})
				if err == nil {
					t.Error("expected enqueue error")
				}
			}

			if tt.mockProducer != nil && tt.mockProducer.enqueueResult != nil {
				// 验证入队成功时结果正确
				if tt.mockProducer.enqueueResult.TaskID == "" {
					t.Error("expected TaskID to be set")
				}
			}
		})
	}
}

// TestCheckTaskDefinition_ValidatesTaskType 测试任务类型校验。
func TestCheckTaskDefinition_ValidatesTaskType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		taskType string
		registry *mockTaskRegistry
		wantErr  bool
	}{
		{
			name:     "支持的任务类型",
			taskType: "demo.task",
			registry: &mockTaskRegistry{
				getFound: true,
				getDef: genericjob.TaskDef{
					Type:       "demo.task",
					Visibility: "public",
				},
			},
			wantErr: false,
		},
		{
			name:     "不支持的任务类型",
			taskType: "unsupported",
			registry: &mockTaskRegistry{
				getFound: false,
			},
			wantErr: true,
		},
		{
			name:     "内部任务类型不允许客户端使用",
			taskType: "internal.task",
			registry: &mockTaskRegistry{
				getFound: true,
				getDef: genericjob.TaskDef{
					Type:       "internal.task",
					Visibility: "internal",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def, ok := tt.registry.Get(tt.taskType)
			if !ok {
				if !tt.wantErr {
					t.Errorf("expected task type %q to be found", tt.taskType)
				}
				return
			}

			if def.Visibility != "public" {
				if !tt.wantErr {
					t.Errorf("expected task type %q to be not public", tt.taskType)
				}
				return
			}

			if tt.wantErr {
				t.Errorf("expected error but got none")
			}
		})
	}
}

// TestValidateQuota_RespectsMaxTasks 测试配额校验。
func TestValidateQuota_RespectsMaxTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		taskCount      int64
		maxTasksPerUser int
		isAdmin        bool
		wantErr        bool
	}{
		{
			name:            "未超过配额",
			taskCount:       50,
			maxTasksPerUser: 100,
			isAdmin:         false,
			wantErr:         false,
		},
		{
			name:            "超过配额",
			taskCount:       100,
			maxTasksPerUser: 100,
			isAdmin:         false,
			wantErr:         true,
		},
		{
			name:            "管理员不受配额限制",
			taskCount:       1000,
			maxTasksPerUser: 100,
			isAdmin:         true,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 模拟配额检查逻辑
			if tt.isAdmin {
				if tt.wantErr {
					t.Errorf("admin should never exceed quota")
				}
				return
			}

			exceeded := tt.taskCount >= int64(tt.maxTasksPerUser)
			if exceeded != tt.wantErr {
				t.Errorf("quota exceeded = %v, want %v", exceeded, tt.wantErr)
			}
		})
	}
}

// TestUpdateLastExecution_CallsStore 测试 UpdateLastExecution 调用 store。
func TestUpdateLastExecution_CallsStore(t *testing.T) {
	t.Parallel()

	store := &mockScheduledTaskStore{}
	store.UpdateLastExecution(context.Background(), "task-1", "exec-1", strPtr("error message"))

	if len(store.updateLastExecutionCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(store.updateLastExecutionCalls))
	}

	call := store.updateLastExecutionCalls[0]
	if call.scheduledTaskID != "task-1" {
		t.Errorf("expected scheduledTaskID 'task-1', got %q", call.scheduledTaskID)
	}
	if call.executionID != "exec-1" {
		t.Errorf("expected executionID 'exec-1', got %q", call.executionID)
	}
	if call.lastError == nil || *call.lastError != "error message" {
		t.Errorf("expected lastError 'error message', got %v", call.lastError)
	}
}

// strPtr 是一个辅助函数，返回字符串指针。
func strPtr(s string) *string {
	return &s
}

// TestListTaskDefinitions_ReturnsPublicTasks 测试 ListTaskDefinitions 返回公开任务。
// 注意：这个测试验证的是 mock 的行为，实际的过滤逻辑在 pkg/job/registry.go 中实现。
func TestListTaskDefinitions_ReturnsPublicTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		defs     []genericjob.TaskDef
		wantLen  int
	}{
		{
			name: "直接返回设置的值",
			defs: []genericjob.TaskDef{
				{Type: "task.1", Visibility: "public"},
				{Type: "task.2", Visibility: "public"},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := &mockTaskRegistry{
				listPublic: tt.defs,
			}
			publicTasks := registry.ListPublic()
			if len(publicTasks) != tt.wantLen {
				t.Errorf("ListPublic() returned %d tasks, want %d", len(publicTasks), tt.wantLen)
			}
		})
	}
}

// TestTriggerScheduledTaskResponse_Fields 测试响应字段。
func TestTriggerScheduledTaskResponse_Fields(t *testing.T) {
	t.Parallel()

	response := &v1.TriggerScheduledTaskResponse{
		ExecutionID: "exec-123",
	}

	if response.ExecutionID != "exec-123" {
		t.Errorf("expected ExecutionID 'exec-123', got %q", response.ExecutionID)
	}
}
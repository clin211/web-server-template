// 定时任务 API 类型定义

declare namespace Api {
  namespace ScheduledTask {
    // 定时任务
    interface ScheduledTask {
      scheduledTaskID: string;
      name: string;
      taskType: string;
      payload: Record<string, unknown>;
      cronExpr: string;
      queue: string;
      enabled: boolean;
      timezone: string;
      userID: string;
      nextRunTime: number;
      lastScheduledAt: number;
      lastExecutionID: string;
      lastError: string;
      createdAt: number;
      updatedAt: number;
    }

    // 执行记录
    interface ScheduledTaskExecution {
      executionID: string;
      scheduledTaskID: string;
      userID: string;
      triggerType: string;
      scheduledAt: number;
      enqueuedAt: number;
      asynqTaskID: string;
      dispatchStatus: string;
      processStatus: string;
      attempt: number;
      errorMsg: string;
      startedAt: number;
      finishedAt: number;
      durationMs: number;
      createdAt: number;
      updatedAt: number;
    }

    // 创建请求
    interface CreateScheduledTaskRequest {
      name: string;
      taskType: string;
      payload?: Record<string, unknown>;
      cronExpr: string;
      queue?: string;
      enabled?: boolean;
      timezone?: string;
    }

    // 更新请求
    interface UpdateScheduledTaskRequest {
      name?: string;
      payload?: Record<string, unknown>;
      cronExpr?: string;
      queue?: string;
      enabled?: boolean;
      timezone?: string;
    }

    // 列表请求
    interface ListScheduledTasksRequest {
      pageToken?: string;
      pageSize?: number;
      enabled?: boolean;
      taskType?: string;
    }

    // 列表响应
    interface ListScheduledTasksResponse {
      totalCount: number;
      scheduledTasks: ScheduledTask[];
      pageToken: string;
    }

    // 执行记录列表请求
    interface ListExecutionsRequest {
      pageToken?: string;
      pageSize?: number;
      dispatchStatus?: string;
      processStatus?: string;
    }

    // 执行记录列表响应
    interface ListExecutionsResponse {
      totalCount: number;
      executions: ScheduledTaskExecution[];
      pageToken: string;
    }

    // 切换状态请求
    interface ToggleRequest {
      enabled: boolean;
    }

    // 表格行类型
    interface ScheduledTaskTableRow extends ScheduledTask {
      // 用于表格行标识
    }

    // 搜索表单模型
    interface SearchModel {
      enabled: string | null;
      taskType: string | null;
    }

    // 任务类型定义
    interface TaskDefinition {
      type: string;
      description?: string;
      allowedQueues: string[];
      maxPayloadBytes: number;
      timeoutSeconds: number;
      maxRetry: number;
    }

    // 任务类型列表响应
    interface ListTaskDefinitionsResponse {
      definitions: TaskDefinition[];
    }
  }
}
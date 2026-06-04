import { request } from '../request';

export function fetchCreateScheduledTask(data: Api.ScheduledTask.CreateScheduledTaskRequest) {
  return request<{ scheduledTaskID: string }>({ url: '/v1/scheduled-tasks', method: 'post', data });
}

export function fetchUpdateScheduledTask(scheduledTaskID: string, data: Api.ScheduledTask.UpdateScheduledTaskRequest) {
  return request({ url: `/v1/scheduled-tasks/${scheduledTaskID}`, method: 'put', data });
}

export function fetchDeleteScheduledTask(scheduledTaskID: string) {
  return request({ url: `/v1/scheduled-tasks/${scheduledTaskID}`, method: 'delete' });
}

export function fetchGetScheduledTask(scheduledTaskID: string) {
  return request<{ scheduledTask: Api.ScheduledTask.ScheduledTask }>({ url: `/v1/scheduled-tasks/${scheduledTaskID}` });
}

export function fetchListScheduledTasks(params?: Api.ScheduledTask.ListScheduledTasksRequest) {
  return request<Api.ScheduledTask.ListScheduledTasksResponse>({ url: '/v1/scheduled-tasks', params });
}

export function fetchToggleScheduledTask(scheduledTaskID: string, enabled: boolean) {
  return request<{ scheduledTask: Api.ScheduledTask.ScheduledTask }>({
    url: `/v1/scheduled-tasks/${scheduledTaskID}/toggle`,
    method: 'put',
    data: { enabled }
  });
}

export function fetchTriggerScheduledTask(scheduledTaskID: string) {
  return request<{ executionID: string }>({ url: `/v1/scheduled-tasks/${scheduledTaskID}/trigger`, method: 'post' });
}

export function fetchListScheduledTaskExecutions(
  scheduledTaskID: string,
  params?: Api.ScheduledTask.ListExecutionsRequest
) {
  return request<Api.ScheduledTask.ListExecutionsResponse>({
    url: `/v1/scheduled-tasks/${scheduledTaskID}/executions`,
    params
  });
}

export function fetchListTaskDefinitions() {
  return request<Api.ScheduledTask.ListTaskDefinitionsResponse>({ url: '/v1/task-definitions' });
}
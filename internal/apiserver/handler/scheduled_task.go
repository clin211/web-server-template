package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/clin211/gin-enterprise-template/pkg/core"
)

func init() {
	Register(func(v1 *gin.RouterGroup, handler *Handler) {
		rg := v1.Group("/scheduled-tasks")
		rg.Use(handler.mws...)
		rg.POST("", handler.CreateScheduledTask)
		rg.PUT(":scheduledTaskID", handler.UpdateScheduledTask)
		rg.DELETE(":scheduledTaskID", handler.DeleteScheduledTask)
		rg.GET(":scheduledTaskID", handler.GetScheduledTask)
		rg.GET("", handler.ListScheduledTask)
		rg.PUT(":scheduledTaskID/toggle", handler.ToggleScheduledTask)
		rg.POST(":scheduledTaskID/trigger", handler.TriggerScheduledTask)
		rg.GET(":scheduledTaskID/executions", handler.ListScheduledTaskExecutions)

		v1.GET("/task-definitions", handler.ListTaskDefinitions)
	})
}

// CreateScheduledTask handles the create scheduled task API.
func (h *Handler) CreateScheduledTask(c *gin.Context) {
	core.HandleProtoJSONRequest(c, h.biz.ScheduledTaskV1().Create, h.val.ValidateCreateScheduledTaskRequest)
}

// UpdateScheduledTask handles the update scheduled task API.
func (h *Handler) UpdateScheduledTask(c *gin.Context) {
	core.HandleUriProtoJSONRequest(c, h.biz.ScheduledTaskV1().Update, h.val.ValidateUpdateScheduledTaskRequest)
}

// DeleteScheduledTask handles the delete scheduled task API.
func (h *Handler) DeleteScheduledTask(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.ScheduledTaskV1().Delete, h.val.ValidateDeleteScheduledTaskRequest)
}

// GetScheduledTask handles the get scheduled task API.
func (h *Handler) GetScheduledTask(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.ScheduledTaskV1().Get, h.val.ValidateGetScheduledTaskRequest)
}

// ListScheduledTask 查询定时任务列表.
func (h *Handler) ListScheduledTask(c *gin.Context) {
	core.HandleQueryRequest(c, h.biz.ScheduledTaskV1().List, h.val.ValidateListScheduledTasksRequest)
}

// ToggleScheduledTask handles the toggle scheduled task API.
func (h *Handler) ToggleScheduledTask(c *gin.Context) {
	core.HandleUriJSONRequest(c, h.biz.ScheduledTaskV1().Toggle, h.val.ValidateToggleScheduledTaskRequest)
}

// TriggerScheduledTask handles the manual trigger API.
func (h *Handler) TriggerScheduledTask(c *gin.Context) {
	core.HandleUriRequest(c, h.biz.ScheduledTaskV1().Trigger, h.val.ValidateTriggerScheduledTaskRequest)
}

// ListScheduledTaskExecutions handles the list executions API.
func (h *Handler) ListScheduledTaskExecutions(c *gin.Context) {
	core.HandleUriQueryRequest(c, h.biz.ScheduledTaskV1().ListExecutions, h.val.ValidateListScheduledTaskExecutionsRequest)
}

// ListTaskDefinitions 获取公开的任务类型列表.
func (h *Handler) ListTaskDefinitions(c *gin.Context) {
	core.HandleRequest(c, c.ShouldBindQuery, h.biz.ScheduledTaskV1().ListTaskDefinitions)
}

package scheduled_task

import (
	"context"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// ListTaskDefinitions 返回所有公开的任务类型定义。
func (b *scheduledTaskBiz) ListTaskDefinitions(ctx context.Context, rq *v1.ListTaskDefinitionsRequest) (*v1.ListTaskDefinitionsResponse, error) {
	taskDefs := b.registry.ListPublic()

	definitions := make([]*v1.TaskDefinition, 0, len(taskDefs))
	for _, def := range taskDefs {
		timeoutSeconds := int32(0)
		if def.Timeout > 0 {
			timeoutSeconds = int32(def.Timeout.Seconds())
		}
		definitions = append(definitions, &v1.TaskDefinition{
			Type:            def.Type,
			Description:     def.Description,
			AllowedQueues:   def.AllowedQueues,
			MaxPayloadBytes: int32(def.MaxPayloadBytes),
			TimeoutSeconds:  timeoutSeconds,
			MaxRetry:        int32(def.RetryPolicy.MaxRetry),
		})
	}

	return &v1.ListTaskDefinitionsResponse{Definitions: definitions}, nil
}
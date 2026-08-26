package known

const (
	// OutboxStatusPending 表示等待发布的事件。
	OutboxStatusPending = "pending"
	// OutboxStatusProcessing 表示已被 Relay 租约领取的事件。
	OutboxStatusProcessing = "processing"
	// OutboxStatusPublished 表示已经成功发布的事件。
	OutboxStatusPublished = "published"
	// OutboxStatusFailed 表示最近一次发布失败且等待重试的事件。
	OutboxStatusFailed = "failed"
	// OutboxStatusDead 表示已达到发布重试上限，需要人工重放或修复后恢复。
	OutboxStatusDead = "dead"
)

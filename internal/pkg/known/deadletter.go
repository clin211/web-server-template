package known

const (
	// DeadLetterStatusPending 表示等待发布的死信。
	DeadLetterStatusPending = "pending"
	// DeadLetterStatusProcessing 表示已被 Relay 租约领取的死信。
	DeadLetterStatusProcessing = "processing"
	// DeadLetterStatusPublished 表示死信已经发布到 Kafka DLQ。
	DeadLetterStatusPublished = "published"
	// DeadLetterStatusFailed 表示最近一次发布失败且等待重试。
	DeadLetterStatusFailed = "failed"
	// DeadLetterStatusDead 表示 DLQ 发布达到上限，需要人工重放。
	DeadLetterStatusDead = "dead"
)

package known

const (
	// InboxStatusProcessing 表示事件已被 worker 领取。
	InboxStatusProcessing = "processing"
	// InboxStatusFailed 表示最近一次处理失败且等待重试。
	InboxStatusFailed = "failed"
	// InboxStatusCompleted 表示业务处理已经成功完成。
	InboxStatusCompleted = "completed"
	// InboxStatusDeadLettered 表示事件已经可靠写入 DLQ。
	InboxStatusDeadLettered = "dead_lettered"
)

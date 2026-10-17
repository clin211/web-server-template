package store

import "context"

// Cleanup removes only bounded terminal history after the replay safety window.
// Active work and unsent events are never removed. Repeated calls are safe.
func Cleanup(ctx context.Context, core IStore) error {
	queries := []string{
		`DELETE FROM report_result WHERE id IN(SELECT id FROM report_result WHERE expires_at<clock_timestamp() LIMIT 500)`,
		`DELETE FROM event_replay WHERE replay_id IN(SELECT replay_id FROM event_replay WHERE created_at<clock_timestamp()-interval '90 days' LIMIT 500)`,
		`DELETE FROM dead_letter_event WHERE id IN(SELECT d.id FROM dead_letter_event d WHERE status IN ('published','dead') AND created_at<clock_timestamp()-interval '90 days' AND NOT EXISTS(SELECT 1 FROM event_replay r WHERE r.dead_letter_id=d.dead_letter_id) LIMIT 500)`,
		`DELETE FROM inbox_event WHERE id IN(SELECT id FROM inbox_event WHERE status IN ('completed','dead_lettered') AND updated_at<clock_timestamp()-interval '90 days' LIMIT 500)`,
		`DELETE FROM outbox_event WHERE id IN(SELECT id FROM outbox_event WHERE status='published' AND updated_at<clock_timestamp()-interval '90 days' LIMIT 500)`,
		`DELETE FROM report_notification WHERE task_id IN(SELECT task_id FROM report_notification WHERE created_at<clock_timestamp()-interval '90 days' LIMIT 500)`,
		`DELETE FROM task WHERE id IN(SELECT t.id FROM task t WHERE status IN ('succeeded','failed','canceled') AND updated_at<clock_timestamp()-interval '90 days' AND NOT EXISTS(SELECT 1 FROM task child WHERE child.retry_of=t.task_id) AND NOT EXISTS(SELECT 1 FROM report_result r WHERE r.task_id=t.task_id) LIMIT 500)`,
		`DELETE FROM schedule_trigger WHERE id IN(SELECT id FROM schedule_trigger WHERE created_at<clock_timestamp()-interval '90 days' LIMIT 500)`,
		`DELETE FROM runtime_heartbeat WHERE updated_at<clock_timestamp()-interval '1 day'`,
		`DELETE FROM audit_log WHERE id IN(SELECT id FROM audit_log WHERE created_at<clock_timestamp()-interval '365 days' LIMIT 500)`,
	}
	for _, q := range queries {
		if err := core.DB(ctx).WithContext(ctx).Exec(q).Error; err != nil {
			return err
		}
	}
	return nil
}

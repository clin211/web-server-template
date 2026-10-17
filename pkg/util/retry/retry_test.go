package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func TestPoll(t *testing.T) {
	conditionErr := errors.New("condition failure")
	for _, test := range []struct {
		name         string
		poll         func(time.Duration, time.Duration, wait.ConditionFunc) error
		timeout      time.Duration
		conditionErr error
	}{
		{name: "delayed", poll: Poll, timeout: time.Second},
		{name: "immediate", poll: PollImmediate, timeout: time.Second},
		{name: "unbounded delayed", poll: Poll},
		{name: "unbounded immediate", poll: PollImmediate},
		{name: "condition error", poll: PollImmediate, timeout: time.Second, conditionErr: conditionErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			err := test.poll(time.Millisecond, test.timeout, func() (bool, error) {
				calls++
				return calls >= 2, test.conditionErr
			})
			if !errors.Is(err, test.conditionErr) {
				t.Fatalf("error = %v, want %v", err, test.conditionErr)
			}
			if calls == 0 {
				t.Fatal("condition was not evaluated")
			}
		})
	}
}

func TestPollTimeout(t *testing.T) {
	for _, poll := range []func(time.Duration, time.Duration, wait.ConditionFunc) error{Poll, PollImmediate} {
		err := poll(time.Millisecond, 3*time.Millisecond, func() (bool, error) { return false, nil })
		if !errors.Is(err, context.DeadlineExceeded) || !wait.Interrupted(err) {
			t.Fatalf("timeout error = %v, want interrupted context deadline", err)
		}
	}
}

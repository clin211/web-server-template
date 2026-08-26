package options

import (
	"errors"
	"testing"
	"time"
)

func TestConsumerOptionsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ConsumerOptions)
		wantErr error
	}{
		{name: "defaults are valid"},
		{name: "max attempts", mutate: func(o *ConsumerOptions) { o.MaxAttempts = 0 }, wantErr: errConsumerMaxAttemptsNotPositive},
		{name: "retry delay", mutate: func(o *ConsumerOptions) { o.RetryDelay = 0 }, wantErr: errConsumerRetryDelayNotPositive},
		{name: "max retry delay", mutate: func(o *ConsumerOptions) { o.MaxRetryDelay = time.Millisecond }, wantErr: errConsumerMaxRetryDelayTooShort},
		{name: "process timeout", mutate: func(o *ConsumerOptions) { o.ProcessTimeout = 0 }, wantErr: errConsumerProcessTimeoutNotPositive},
		{name: "inbox lease", mutate: func(o *ConsumerOptions) { o.InboxLeaseDuration = 0 }, wantErr: errConsumerInboxLeaseNotPositive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			options := NewConsumerOptions()
			if tt.mutate != nil {
				tt.mutate(options)
			}
			errs := options.Validate()
			if tt.wantErr == nil && len(errs) != 0 {
				t.Fatalf("Validate() errors = %v, want none", errs)
			}
			if tt.wantErr != nil && !containsError(errs, tt.wantErr) {
				t.Fatalf("Validate() errors = %v, want %v", errs, tt.wantErr)
			}
		})
	}
}

func containsError(errs []error, target error) bool {
	for _, err := range errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

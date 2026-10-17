package id

import (
	"context"
	"testing"
	"time"
)

func TestSonyflakeInvalidStartTime(t *testing.T) {
	generator := NewSonyflake(WithSonyflakeStartTime(time.Now().Add(time.Hour)))
	if generator.Error == nil {
		t.Fatal("future start time should fail initialization")
	}
	if got := generator.Id(context.Background()); got != 0 {
		t.Fatalf("invalid generator returned %d", got)
	}
}

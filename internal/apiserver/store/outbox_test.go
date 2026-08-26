package store

import (
	"errors"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
)

func TestCheckOutboxLeaseUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rowsAffected int64
		updateErr    error
		wantErr      error
	}{
		{name: "updated", rowsAffected: 1},
		{name: "lease lost", wantErr: outbox.ErrLeaseLost},
		{name: "database error", updateErr: errors.New("database unavailable"), wantErr: errOutboxUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkOutboxLeaseUpdate("event-1", 2, tt.rowsAffected, tt.updateErr)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkOutboxLeaseUpdate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckInboxLeaseUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rowsAffected int64
		updateErr    error
		wantErr      error
	}{
		{name: "updated", rowsAffected: 1},
		{name: "lease lost", wantErr: consumer.ErrInboxLeaseLost},
		{name: "database error", updateErr: errors.New("database unavailable"), wantErr: errInboxUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkInboxLeaseUpdate("users", "event-1", 2, tt.rowsAffected, tt.updateErr)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkInboxLeaseUpdate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckDeadLetterLeaseUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		rowsAffected int64
		updateErr    error
		wantErr      error
	}{
		{name: "updated", rowsAffected: 1},
		{name: "lease lost", wantErr: outbox.ErrLeaseLost},
		{name: "database error", updateErr: errors.New("database unavailable"), wantErr: errDeadLetterUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkDeadLetterLeaseUpdate("dlq-1", 2, tt.rowsAffected, tt.updateErr)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("checkDeadLetterLeaseUpdate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

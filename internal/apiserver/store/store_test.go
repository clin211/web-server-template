package store

import (
	"testing"

	"gorm.io/gorm"
)

func TestNewStoreDoesNotShareProcessGlobalState(t *testing.T) {
	t.Parallel()

	firstDB := &gorm.DB{}
	secondDB := &gorm.DB{}
	first := NewStore(firstDB)
	second := NewStore(secondDB)

	if first == second {
		t.Fatal("NewStore() returned a process-wide singleton")
	}
	if first.core != firstDB || second.core != secondDB {
		t.Fatal("NewStore() did not retain its explicitly injected database")
	}
}

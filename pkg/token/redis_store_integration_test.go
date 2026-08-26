package token

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestIntegrationRedisSecurityState(t *testing.T) {
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("TEST_REDIS_ADDR is not set")
	}
	client := redis.NewClient(&redis.Options{Addr: address, Password: os.Getenv("TEST_REDIS_PASSWORD"), DB: 15})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis: %v", err)
		}
	})
	ctx := context.Background()
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush Redis test database: %v", err)
	}
	store := NewRedisRefreshTokenStore(client)

	for attempt := 1; attempt <= 3; attempt++ {
		allowed, err := store.AllowLogin(ctx, "192.0.2.1", 2, time.Minute)
		if err != nil {
			t.Fatalf("AllowLogin() error = %v", err)
		}
		if allowed != (attempt <= 2) {
			t.Fatalf("attempt %d allowed = %v", attempt, allowed)
		}
	}
	consumed, err := store.Consume(ctx, "refresh-id", time.Minute)
	if err != nil || !consumed {
		t.Fatalf("first Consume() = %v, %v", consumed, err)
	}
	consumed, err = store.Consume(ctx, "refresh-id", time.Minute)
	if err != nil || consumed {
		t.Fatalf("second Consume() = %v, %v", consumed, err)
	}
}

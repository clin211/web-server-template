package store

import (
	"context"
	"crypto/sha256"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/messaging/consumer"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
)

func TestIntegrationInboxClaimUsesSingleLease(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test disabled in short mode")
	}
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	store := &datastore{core: db}
	hash := sha256.Sum256([]byte("integration-event"))
	identity := consumer.MessageIdentity{
		ConsumerName: "integration-consumer", EventID: "integration-event-1", EventType: "example.created.v1",
		SourceTopic: "events", SourcePartition: 0, SourceOffset: time.Now().UnixNano(), PayloadHash: hash[:],
	}
	if err := db.Where("consumer_name = ? AND event_id = ?", identity.ConsumerName, identity.EventID).Delete(&model.InboxEventM{}).Error; err != nil {
		t.Fatalf("cleanup Inbox: %v", err)
	}

	results := make(chan consumer.Claim, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claim, claimErr := store.Inbox().Claim(context.Background(), identity, time.Minute)
			if claimErr != nil {
				t.Errorf("Claim() error = %v", claimErr)
				return
			}
			results <- claim
		}()
	}
	wg.Wait()
	close(results)
	acquired := 0
	for claim := range results {
		if claim.Disposition == consumer.ClaimAcquired {
			acquired++
		}
	}
	if acquired != 1 {
		t.Fatalf("acquired claims = %d, want 1", acquired)
	}
	if err := db.Where("consumer_name = ? AND event_id = ?", identity.ConsumerName, identity.EventID).Delete(&model.InboxEventM{}).Error; err != nil {
		t.Fatalf("cleanup Inbox after test: %v", err)
	}
	_ = known.InboxStatusProcessing
}

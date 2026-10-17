package consumer

import (
	"context"
	"github.com/google/uuid"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestIntegrationKafkaManualCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test disabled in short mode")
	}
	brokers := os.Getenv("TEST_KAFKA_BROKERS")
	topic := os.Getenv("TEST_KAFKA_TOPIC")
	if brokers == "" || topic == "" {
		t.Skip("TEST_KAFKA_BROKERS and TEST_KAFKA_TOPIC are required")
	}
	topic += "-test-" + uuid.NewString()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := kafka.DialContext(ctx, "tcp", strings.Split(brokers, ",")[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.CreateTopics(kafka.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.DeleteTopics(topic); err != nil {
			t.Error(err)
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	writer := &kafka.Writer{Addr: kafka.TCP(brokers), Topic: topic, Balancer: &kafka.LeastBytes{}}
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte("integration"), Value: []byte("integration-payload")}); err != nil {
		t.Fatalf("write Kafka message: %v", err)
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{brokers}, Topic: topic, GroupID: "integration-consumer-" + time.Now().Format("150405.000000000"), MinBytes: 1, MaxBytes: 1 << 20})
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	message, err := reader.FetchMessage(ctx)
	if err != nil {
		t.Fatalf("fetch Kafka message: %v", err)
	}
	if string(message.Value) != "integration-payload" {
		t.Fatalf("message value = %q", message.Value)
	}
	if err := reader.CommitMessages(ctx, message); err != nil {
		t.Fatalf("commit Kafka message: %v", err)
	}
}

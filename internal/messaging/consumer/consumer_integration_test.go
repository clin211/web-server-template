package consumer

import (
	"context"
	"os"
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
	writer := &kafka.Writer{Addr: kafka.TCP(brokers), Topic: topic, Balancer: &kafka.LeastBytes{}}
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte("integration"), Value: []byte("integration-payload")}); err != nil {
		t.Fatalf("write Kafka message: %v", err)
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{brokers}, Topic: topic, GroupID: "integration-consumer-" + time.Now().Format("150405.000000000"), MinBytes: 1, MaxBytes: 1 << 20})
	defer reader.Close()
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

package options

import (
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestNewKafkaOptionsUsesReliableWriterDefaults(t *testing.T) {
	t.Parallel()

	options := NewKafkaOptions()
	if options.WriterOptions.Async {
		t.Error("writer async = true, want false")
	}
	if options.WriterOptions.RequiredAcks != int(kafka.RequireAll) {
		t.Errorf("required acks = %d, want %d", options.WriterOptions.RequiredAcks, kafka.RequireAll)
	}
}

func TestKafkaOptionsRejectsAsyncWriter(t *testing.T) {
	t.Parallel()

	options := NewKafkaOptions()
	options.WriterOptions.Async = true
	if err := options.Validate(); len(err) == 0 {
		t.Fatal("Validate() returned no error for async writer")
	}
	options.Brokers = []string{"127.0.0.1:9092"}
	options.Topic = "events"
	if _, err := options.Writer(); err == nil {
		t.Fatal("Writer() error = nil for async writer")
	}
}

func TestKafkaOptionsWriterUsesKeyHashBalancer(t *testing.T) {
	t.Parallel()

	options := NewKafkaOptions()
	options.Brokers = []string{"127.0.0.1:9092"}
	options.Topic = "domain-events"

	writer, err := options.Writer()
	if err != nil {
		t.Fatalf("Writer() error = %v", err)
	}
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if _, ok := writer.Balancer.(*kafka.Hash); !ok {
		t.Fatalf("writer balancer = %T, want *kafka.Hash", writer.Balancer)
	}
	partitions := []int{0, 1, 2, 3}
	first := writer.Balancer.Balance(kafka.Message{Key: []byte("user-1")}, partitions...)
	for range 10 {
		if got := writer.Balancer.Balance(kafka.Message{Key: []byte("user-1")}, partitions...); got != first {
			t.Fatalf("same key partition = %d, want %d", got, first)
		}
	}
}

func TestKafkaOptionsReaderUsesManualCommitConfiguration(t *testing.T) {
	t.Parallel()

	options := NewKafkaOptions()
	options.Brokers = []string{"127.0.0.1:9092"}
	options.Topic = "domain-events"
	options.ReaderOptions.Topic = "inbound-events"
	options.ReaderOptions.GroupID = "apiserver"

	config, err := options.ReaderConfig()
	if err != nil {
		t.Fatalf("ReaderConfig() error = %v", err)
	}
	if config.GroupID != "apiserver" || config.Topic != "inbound-events" {
		t.Fatalf("reader identity = %q/%q", config.GroupID, config.Topic)
	}
	if config.Topic == options.Topic {
		t.Fatalf("reader topic = writer topic %q, want independently configured topics", config.Topic)
	}
	if config.CommitInterval != 0 {
		t.Fatalf("commit interval = %s, want synchronous commits", config.CommitInterval)
	}
}

func TestKafkaOptionsReaderConfigRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	options := NewKafkaOptions()
	options.ReaderOptions.Topic = "inbound-events"
	options.ReaderOptions.GroupID = "apiserver"

	if _, err := options.ReaderConfig(); err == nil {
		t.Fatal("ReaderConfig() error = nil, want invalid broker error")
	}
}

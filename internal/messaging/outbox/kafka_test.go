package outbox

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
)

type messageWriter struct {
	messages []kafka.Message
	err      error
}

func (w *messageWriter) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	w.messages = append(w.messages, messages...)
	return w.err
}

func TestKafkaPublisherMapsEventToMessage(t *testing.T) {
	t.Parallel()

	writer := &messageWriter{}
	publisher, err := NewKafkaPublisher(writer)
	if err != nil {
		t.Fatalf("NewKafkaPublisher() error = %v", err)
	}
	event := Event{
		EventID:     "event-1",
		EventType:   "example.created.v1",
		AggregateID: "user-1",
		TraceID:     "trace-1",
		Payload:     []byte(`{"event_id":"event-1"}`),
	}

	if err := publisher.Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(writer.messages))
	}
	message := writer.messages[0]
	if got := string(message.Key); got != event.AggregateID {
		t.Errorf("message key = %q, want %q", got, event.AggregateID)
	}
	if got := string(message.Value); got != string(event.Payload) {
		t.Errorf("message value = %q, want %q", got, event.Payload)
	}
	wantHeaders := map[string]string{
		"event_id": event.EventID, "event_type": event.EventType, "trace_id": event.TraceID,
	}
	for _, header := range message.Headers {
		delete(wantHeaders, header.Key)
	}
	if len(wantHeaders) != 0 {
		t.Errorf("missing Kafka headers: %v", wantHeaders)
	}
}

func TestNewKafkaPublisherRejectsNilWriter(t *testing.T) {
	t.Parallel()

	if _, err := NewKafkaPublisher(nil); err == nil {
		t.Fatal("NewKafkaPublisher() error = nil, want validation error")
	}
}

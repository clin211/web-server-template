package outbox

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestNewEventBuildsVersionedEnvelope(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 21, 8, 0, 0, 0, time.UTC)
	event, err := NewEvent(NewEventInput{
		EventID:       "event-1",
		EventType:     "example.created.v1",
		Producer:      "apiserver",
		AggregateType: "example",
		AggregateID:   "example-1",
		TraceID:       "trace-1",
		OccurredAt:    now,
		Payload:       map[string]string{"example_id": "example-1"},
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(event.Payload, &envelope); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	for key, want := range map[string]string{
		"event_id": "event-1", "event_type": "example.created.v1", "producer": "apiserver",
		"aggregate_type": "example", "aggregate_id": "example-1", "trace_id": "trace-1",
	} {
		if got := envelope[key]; got != want {
			t.Errorf("envelope[%q] = %v, want %q", key, got, want)
		}
	}
}

func TestParseEnvelopeRestoresEventContract(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"event_id":"event-1","event_type":"example.created.v1","occurred_at":"2026-08-21T08:00:00Z","producer":"apiserver","trace_id":"trace-1","aggregate_type":"example","aggregate_id":"example-1","payload":{"example_id":"example-1"}}`)
	envelope, err := ParseEnvelope(payload)
	if err != nil {
		t.Fatalf("ParseEnvelope() error = %v", err)
	}
	if envelope.EventID != "event-1" || envelope.EventType != "example.created.v1" || envelope.Producer != "apiserver" {
		t.Fatalf("envelope = %+v", envelope)
	}
	if got := string(envelope.Payload); got != `{"example_id":"example-1"}` {
		t.Fatalf("payload = %s", got)
	}
}

func TestParseEnvelopeRejectsMalformedOrIncompletePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{name: "malformed", payload: `{`, wantErr: errEnvelopeMalformed},
		{name: "missing event id", payload: `{"event_type":"example.created.v1"}`, wantErr: errEventIDRequired},
		{name: "missing payload", payload: `{"event_id":"event-1","event_type":"example.created.v1","occurred_at":"2026-08-21T08:00:00Z","producer":"apiserver","aggregate_type":"example","aggregate_id":"example-1"}`, wantErr: errEnvelopePayloadRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseEnvelope([]byte(tt.payload))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseEnvelope() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEventRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input NewEventInput
	}{
		{name: "event id", input: NewEventInput{EventType: "example.created.v1", AggregateType: "example", AggregateID: "example-1"}},
		{name: "event type", input: NewEventInput{EventID: "event-1", AggregateType: "example", AggregateID: "example-1"}},
		{name: "aggregate type", input: NewEventInput{EventID: "event-1", EventType: "example.created.v1", AggregateID: "example-1"}},
		{name: "aggregate id", input: NewEventInput{EventID: "event-1", EventType: "example.created.v1", AggregateType: "example"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewEvent(tt.input); err == nil {
				t.Fatal("NewEvent() error = nil, want validation error")
			}
		})
	}
}

package options

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestOTelOptionsGetResource(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		environment string
	}{
		{name: "first instance", serviceName: "service-a", environment: "development"},
		{name: "second instance", serviceName: "service-b", environment: "production"},
	}

	resources := make([]resourceSnapshot, 0, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOTelOptions()
			opts.ServiceName = tt.serviceName
			opts.Environment = tt.environment

			res := opts.GetResource()
			if res == nil {
				t.Fatal("GetResource() returned nil")
			}
			if res != opts.GetResource() {
				t.Fatal("GetResource() did not reuse instance cache")
			}

			serviceName, ok := res.Set().Value(attribute.Key("service.name"))
			if !ok {
				t.Fatal("service.name attribute missing")
			}
			if got := serviceName.AsString(); got != tt.serviceName {
				t.Fatalf("service.name = %q, want %q", got, tt.serviceName)
			}

			environment, ok := res.Set().Value(attribute.Key("deployment.environment"))
			if !ok {
				t.Fatal("deployment.environment attribute missing")
			}
			if got := environment.AsString(); got != tt.environment {
				t.Fatalf("deployment.environment = %q, want %q", got, tt.environment)
			}

			resources = append(resources, resourceSnapshot{name: tt.name, ptr: res, serviceName: tt.serviceName})
		})
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}
	if resources[0].ptr == resources[1].ptr {
		t.Fatal("different OTelOptions instances unexpectedly shared the same resource pointer")
	}
}

type resourceSnapshot struct {
	name        string
	ptr         any
	serviceName string
}

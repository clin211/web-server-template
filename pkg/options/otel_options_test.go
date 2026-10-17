package options

import (
	"errors"
	"go.opentelemetry.io/otel"
	apimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"io"
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

func TestInitFileMetricsExporterFailure(t *testing.T) {
	t.Setenv("OTEL_GO_X_OBSERVABILITY", "true")
	original := otel.GetMeterProvider()
	t.Cleanup(func() { otel.SetMeterProvider(original) })
	otel.SetMeterProvider(failingMeterProvider{MeterProvider: noop.NewMeterProvider()})
	opts := NewOTelOptions()
	opts.OutputMode = OutputModeFile
	opts.OutputDir = t.TempDir()
	t.Cleanup(func() {
		if err := opts.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	if err := opts.initMetrics(t.Context()); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("initMetrics error = %v, want exporter construction error", err)
	}
	if opts.GetMeterProvider() != nil {
		t.Fatal("failed initialization installed a meter provider")
	}
}

type failingMeterProvider struct{ apimetric.MeterProvider }

func (p failingMeterProvider) Meter(name string, opts ...apimetric.MeterOption) apimetric.Meter {
	return failingMeter{Meter: p.MeterProvider.Meter(name, opts...)}
}

type failingMeter struct{ apimetric.Meter }

func (m failingMeter) Int64UpDownCounter(name string, opts ...apimetric.Int64UpDownCounterOption) (apimetric.Int64UpDownCounter, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestOTelShutdownPreservesAllCloseErrors(t *testing.T) {
	opts := NewOTelOptions()
	opts.files = []io.Closer{errorCloser{io.ErrClosedPipe}, errorCloser{io.ErrUnexpectedEOF}}
	err := opts.Shutdown(t.Context())
	if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Shutdown error = %v, want both close failures in the error chain", err)
	}
	if err := opts.Shutdown(t.Context()); err != nil {
		t.Fatalf("second Shutdown = %v, want no repeated close", err)
	}
}

type errorCloser struct{ err error }

func (c errorCloser) Close() error { return c.err }

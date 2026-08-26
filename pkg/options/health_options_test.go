package options

import "testing"

func TestNewHealthOptionsKeepsProfilerOnLoopbackByDefault(t *testing.T) {
	opts := NewHealthOptions()
	if opts.HTTPProfile {
		t.Fatal("HTTPProfile = true, want explicit opt-in")
	}
	if opts.HealthCheckAddress != "127.0.0.1:20250" {
		t.Fatalf("HealthCheckAddress = %q, want loopback listener", opts.HealthCheckAddress)
	}
}

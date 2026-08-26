package options

import (
	"strings"
	"testing"
)

func TestDatabaseOptionsRequirePasswords(t *testing.T) {
	postgres := NewPostgreSQLOptions()
	if errs := postgres.Validate(); len(errs) == 0 {
		t.Fatal("PostgreSQL Validate() returned no error for empty password")
	}
	redis := NewRedisOptions()
	if errs := redis.Validate(); len(errs) == 0 {
		t.Fatal("Redis Validate() returned no error for empty password")
	}
}

func TestPostgreSQLOptionsValidateSSLMode(t *testing.T) {
	tests := []struct {
		name      string
		sslMode   string
		wantError string
	}{
		{name: "local development may disable TLS", sslMode: "disable"},
		{name: "production may require TLS", sslMode: "require"},
		{name: "custom CA modes are unsupported", sslMode: "verify-full", wantError: "must be one of disable, require"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := NewPostgreSQLOptions()
			options.Password = "strong-local-password"
			options.SSLMode = tt.sslMode
			errs := options.Validate()
			if tt.wantError == "" {
				if len(errs) != 0 {
					t.Fatalf("Validate() errors = %v, want none", errs)
				}
				return
			}
			for _, err := range errs {
				if strings.Contains(err.Error(), tt.wantError) {
					return
				}
			}
			t.Fatalf("Validate() errors = %v, want error containing %q", errs, tt.wantError)
		})
	}
}

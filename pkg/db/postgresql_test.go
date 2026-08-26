package db

import (
	"strings"
	"testing"
)

func TestPostgreSQLDSNUsesStructuredAddressAndEscaping(t *testing.T) {
	opts := &PostgreSQLOptions{
		Addr:     "[2001:db8::1]:5432",
		Username: "user@example",
		Password: "p@ss word",
		Database: "app",
		SSLMode:  "require",
	}
	dsn := opts.DSN()
	for _, want := range []string{"postgres://", "user%40example", "p%40ss%20word", "[2001:db8::1]:5432", "sslmode=require"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("DSN = %q, missing %q", dsn, want)
		}
	}
}

func TestPostgreSQLDSNDefaultsPort(t *testing.T) {
	opts := &PostgreSQLOptions{Addr: "db.internal", Database: "app", SSLMode: "disable"}
	if got := opts.DSN(); !strings.Contains(got, "db.internal:5432") {
		t.Fatalf("DSN = %q, want default port", got)
	}
}

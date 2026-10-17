package gormslog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSQLLogNeverInterpolatesCredentials(t *testing.T) {
	var output bytes.Buffer
	log := New(slog.New(slog.NewJSONHandler(&output, nil)), logger.Info)
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 dbname=unused user=unused password=unused"), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO "user" (password) VALUES (?)`, "private-password-hash").Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "private-password-hash") {
		t.Fatal("SQL log leaked credential")
	}
	if !strings.Contains(output.String(), "INSERT") {
		t.Fatal("missing diagnostic SQL")
	}
}

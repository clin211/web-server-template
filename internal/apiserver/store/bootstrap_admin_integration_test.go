package store_test

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

func runBootstrapAdmin(db *gorm.DB, user string) ([]byte, error) {
	var database string
	if err := db.Raw("SELECT current_database()").Scan(&database).Error; err != nil {
		return nil, err
	}
	u, err := url.Parse(os.Getenv("TEST_POSTGRES_DSN"))
	if err != nil {
		return nil, err
	}
	password, _ := u.User.Password()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "../../../scripts/bootstrap-admin.sh", user)
	cmd.Env = append(os.Environ(), "PGHOST="+u.Hostname(), "PGPORT="+u.Port(), "PGDATABASE="+database, "PGUSER="+u.User.Username(), "PGPASSWORD="+password, "PGSSLMODE="+u.Query().Get("sslmode"))
	return cmd.CombinedOutput()
}

func bootstrapAdmin(t *testing.T, db *gorm.DB, user string) {
	t.Helper()
	if output, err := runBootstrapAdmin(db, user); err != nil {
		t.Fatalf("bootstrap administrator script: %v\n%s", err, output)
	}
}

func assertBootstrapCounts(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, query := range []string{
		"SELECT count(*) FROM user_role WHERE role_id='role_admin'",
		"SELECT count(*) FROM audit_log WHERE action='rbac.bootstrap' AND outcome='success'",
	} {
		var count int64
		if err := db.Raw(query).Scan(&count).Error; err != nil || count != want {
			t.Fatalf("bootstrap count=%d, want=%d, error=%v", count, want, err)
		}
	}
}

func TestIntegrationBootstrapAdminScript(t *testing.T) {
	t.Run("grant and audit once", func(t *testing.T) {
		db, _ := testDatabase(t)
		bootstrapAdmin(t, db, "owner")
		assertBootstrapCounts(t, db, 1)
		for _, user := range []string{"owner", "other"} {
			output, err := runBootstrapAdmin(db, user)
			if err == nil || !strings.Contains(string(output), "administrator already initialized") {
				t.Fatalf("second bootstrap must fail: %v\n%s", err, output)
			}
		}
		assertBootstrapCounts(t, db, 1)
	})
	t.Run("invalid account and migration state", func(t *testing.T) {
		db, _ := testDatabase(t)
		if err := db.Exec(`UPDATE "user" SET status=1 WHERE user_id='other'`).Error; err != nil {
			t.Fatal(err)
		}
		for _, user := range []string{"missing", "other", "owner'; --"} {
			output, err := runBootstrapAdmin(db, user)
			if err == nil || !strings.Contains(string(output), "account must exist and be active") {
				t.Fatalf("invalid account must fail: %v\n%s", err, output)
			}
			assertBootstrapCounts(t, db, 0)
		}
		if err := db.Exec("INSERT INTO goose_db_version(version_id,is_applied) VALUES(2,false)").Error; err != nil {
			t.Fatal(err)
		}
		output, err := runBootstrapAdmin(db, "owner")
		if err == nil || !strings.Contains(string(output), "migration version incompatible") {
			t.Fatalf("incompatible migration state must fail: %v\n%s", err, output)
		}
		assertBootstrapCounts(t, db, 0)
	})
	t.Run("quoted account is treated as a literal", func(t *testing.T) {
		db, _ := testDatabase(t)
		user := "quoted'; --"
		if err := db.Exec(`INSERT INTO "user"(user_id,username,password,nickname) VALUES(?,?,?,?)`, user, "quoted", "test-hash", "quoted").Error; err != nil {
			t.Fatal(err)
		}
		bootstrapAdmin(t, db, user)
		assertBootstrapCounts(t, db, 1)
		var owner string
		if err := db.Raw("SELECT user_id FROM user_role WHERE role_id='role_admin'").Scan(&owner).Error; err != nil || owner != user {
			t.Fatalf("granted account=%q, error=%v", owner, err)
		}
	})
	t.Run("audit failure rolls back grant", func(t *testing.T) {
		db, _ := testDatabase(t)
		if err := db.Exec("ALTER TABLE audit_log ADD CONSTRAINT reject_bootstrap CHECK(action<>'rbac.bootstrap')").Error; err != nil {
			t.Fatal(err)
		}
		if output, err := runBootstrapAdmin(db, "owner"); err == nil || !strings.Contains(string(output), "reject_bootstrap") {
			t.Fatalf("audit failure must abort bootstrap: %v\n%s", err, output)
		}
		assertBootstrapCounts(t, db, 0)
	})
	t.Run("concurrent invocations grant only one administrator", func(t *testing.T) {
		db, _ := testDatabase(t)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		start := make(chan struct{})
		for _, user := range []string{"owner", "other"} {
			wg.Go(func() {
				<-start
				_, err := runBootstrapAdmin(db, user)
				results <- err
			})
		}
		close(start)
		wg.Wait()
		close(results)
		succeeded := 0
		for err := range results {
			if err == nil {
				succeeded++
			}
		}
		if succeeded != 1 {
			t.Fatalf("successful invocations=%d, want=1", succeeded)
		}
		assertBootstrapCounts(t, db, 1)
	})
}

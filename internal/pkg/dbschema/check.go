// Package dbschema checks Goose history without executing migrations or importing Goose.
package dbschema

//go:generate python3 ../../../scripts/generate-migration-manifest.py --sql-dir ../../../migrations --output manifest_gen.go

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var errVersionIncompatible = errors.New("database migration version incompatible with binary; run goose -dir migrations up")

// Check requires the migration versions shipped with this binary. Only Goose's
// latest record for each version counts, including any rollback and reapplication.
// It never creates tables or applies SQL.
func Check(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT version_id FROM goose_db_version
WHERE id IN (SELECT max(id) FROM goose_db_version GROUP BY version_id)
AND is_applied ORDER BY version_id`)
	if err != nil {
		return fmt.Errorf("read Goose history (run goose -dir migrations up first): %w", err)
	}
	var closeErr error
	checkErr := func() error {
		defer func() { closeErr = rows.Close() }()
		return checkVersions(rows, append([]int64{0}, Manifest()...))
	}()
	return errors.Join(checkErr, rows.Err(), closeErr)
}

func checkVersions(rows *sql.Rows, expected []int64) error {
	i := 0
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("read Goose version: %w", err)
		}
		if i >= len(expected) || version != expected[i] {
			return errVersionIncompatible
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read Goose history: %w", err)
	}
	if i != len(expected) {
		return errVersionIncompatible
	}
	return nil
}

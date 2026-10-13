package database

import (
	"context"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/maintenance"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func migrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("FILEX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires FILEX_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := "upgrade_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	c, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	c.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		db.Close()
		if _, e := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
		if e := admin.Close(ctx); e != nil {
			t.Error(e)
		}
	})
	return db
}

func migrationSource(t *testing.T) fstest.MapFS {
	t.Helper()
	source := fstest.MapFS{}
	entries, e := migrations.ReadDir("migrations")
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		name := "migrations/" + entry.Name()
		raw, e := fs.ReadFile(migrations, name)
		if e != nil {
			t.Fatal(e)
		}
		source[name] = &fstest.MapFile{Data: raw}
	}
	return source
}

func TestIntegrationMigrationPrefixAndStartupRejection(t *testing.T) {
	ctx := context.Background()
	db := migrationDB(t)
	plan, e := CheckUpgrade(ctx, db)
	if e != nil || !plan.Compatible || plan.Applied != 0 || len(plan.Pending) != 18 {
		t.Fatal("fresh plan", plan, e)
	}
	var ledger bool
	if e = db.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.schema_migrations') IS NOT NULL").Scan(&ledger); e != nil || ledger {
		t.Fatal("preflight created ledger", e)
	}
	// Upgrade a real prefix with all remaining historical DDL, not fake ledger rows.
	source := migrationSource(t)
	delete(source, "migrations/0016_administrative_exports.sql")
	delete(source, "migrations/0017_native_webdav.sql")
	delete(source, "migrations/0018_branding.sql")
	if e = migrate(ctx, db, source); e != nil {
		t.Fatal(e)
	}
	if e = Migrate(ctx, db); e == nil {
		t.Fatal("pending DDL accepted on open installation")
	}
	if _, e = db.Exec(ctx, "UPDATE ops_control SET mode='maintenance'"); e != nil {
		t.Fatal(e)
	}
	before, _ := maintenance.Read(ctx, db)
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- Migrate(ctx, db) }()
	}
	for range 2 {
		if e = <-results; e != nil {
			t.Fatal("concurrent startup", e)
		}
	}
	after, e := maintenance.Read(ctx, db)
	if e != nil || after.Generation == before.Generation || after.Mode == "open" {
		t.Fatal("upgrade fence", after, e)
	}
	plan, e = CheckUpgrade(ctx, db)
	if e != nil || plan.Applied != 18 || len(plan.Pending) != 0 {
		t.Fatal(plan, e)
	}
	if e = Migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	same, _ := maintenance.Read(ctx, db)
	if same.Generation != after.Generation {
		t.Fatal("idempotent startup fenced runtime")
	}
	for name, sql := range map[string]string{
		"unknown":  "INSERT INTO schema_migrations(name,sha256) VALUES('9999_newer.sql','unknown')",
		"checksum": "UPDATE schema_migrations SET sha256='changed' WHERE name='0001_core.sql'",
		"hole":     "DELETE FROM schema_migrations WHERE name='0002_identity.sql'",
	} {
		t.Run(name, func(t *testing.T) {
			broken := migrationDB(t)
			if e := Migrate(ctx, broken); e != nil {
				t.Fatal(e)
			}
			if _, e := broken.Exec(ctx, sql); e != nil {
				t.Fatal(e)
			}
			if p, e := CheckUpgrade(ctx, broken); e == nil || p.Compatible {
				t.Fatal("preflight accepted incompatible ledger", p, e)
			}
			if e := Migrate(ctx, broken); e == nil {
				t.Fatal("startup accepted incompatible ledger")
			}
		})
	}
}

func TestIntegrationMigrationDrainAndAtomicFailure(t *testing.T) {
	ctx := context.Background()
	db := migrationDB(t)
	if e := Migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	source := migrationSource(t)
	source["migrations/0019_test_only.sql"] = &fstest.MapFile{Data: []byte("CREATE TABLE upgrade_probe(id int PRIMARY KEY); INSERT INTO upgrade_probe VALUES(1)")}
	source["migrations/0020_test_only.sql"] = &fstest.MapFile{Data: []byte("INSERT INTO upgrade_missing_relation VALUES(1)")}
	if _, e := db.Exec(ctx, "UPDATE ops_control SET mode='maintenance'"); e != nil {
		t.Fatal(e)
	}
	before, _ := maintenance.Read(ctx, db)
	gate, e := maintenance.New(ctx, db)
	if e != nil {
		t.Fatal(e)
	}
	defer gate.Close()
	_, release, e := gate.Acquire(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	short, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	e = migrate(short, db, source)
	cancel()
	if e == nil {
		t.Fatal("DDL bypassed active runtime lock")
	}
	release()
	if e = migrate(ctx, db, source); e == nil {
		t.Fatal("fault injection unexpectedly passed")
	}
	var exists bool
	if e = db.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.upgrade_probe') IS NOT NULL").Scan(&exists); e != nil || exists {
		t.Fatal("partial DDL survived rollback", e)
	}
	plan, e := CheckUpgrade(ctx, db)
	if e != nil || plan.Applied != 18 || len(plan.Pending) != 0 {
		t.Fatal("partial migration ledger survived", plan, e)
	}
	after, _ := maintenance.Read(ctx, db)
	if after.Generation != before.Generation || after.Mode != "maintenance" {
		t.Fatal("failed migration changed gate", after)
	}
	delete(source, "migrations/0020_test_only.sql")
	if e = migrate(ctx, db, source); e != nil {
		t.Fatal("migration retry failed", e)
	}
	if _, _, e = gate.Acquire(ctx, true); e != maintenance.ErrRestart {
		t.Fatal("old process survived DDL", e)
	}
	if e = Migrate(ctx, db); e == nil {
		t.Fatal("older executable accepted upgraded schema")
	}
}

package database

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"filex.local/filex/internal/config"
	"filex.local/filex/internal/maintenance"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	c, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	c.MaxConns = 10
	c.MinConns = 0
	c.MaxConnLifetime = time.Hour
	c.ConnConfig.RuntimeParams["statement_timeout"] = "10000"
	c.ConnConfig.RuntimeParams["lock_timeout"] = "3000"
	c.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "10000"
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, errors.New("database connection failed")
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, errors.New("database unavailable")
	}
	return p, nil
}

func Migrate(ctx context.Context, p *pgxpool.Pool) error {
	return migrate(ctx, p, migrations)
}

func migrate(ctx context.Context, p *pgxpool.Pool, source fs.FS) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184993)"); err != nil {
		return err
	}
	plan, err := migrationPlan(ctx, tx, source)
	if err != nil {
		return err
	}
	if len(plan.Pending) == 0 {
		return tx.Commit(ctx)
	}
	var controlled bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.ops_control') IS NOT NULL").Scan(&controlled); err != nil {
		return err
	}
	checkClosed := func(lock bool) error {
		if !controlled {
			return nil
		}
		q := "SELECT mode FROM ops_control WHERE singleton"
		if lock {
			q += " FOR UPDATE"
		}
		var mode string
		if e := tx.QueryRow(ctx, q).Scan(&mode); e != nil {
			return e
		}
		if mode == "open" {
			return errors.New("pending migrations require a closed maintenance gate")
		}
		return nil
	}
	if err = checkClosed(false); err != nil {
		return err
	}
	if err = maintenance.ExclusiveTransaction(ctx, tx); err != nil {
		return err
	}
	// Reopening may have raced with the initial read. Hold the row through DDL.
	if err = checkClosed(true); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, sha256 text NOT NULL, applied_at timestamptz NOT NULL DEFAULT clock_timestamp())"); err != nil {
		return err
	}
	for _, name := range plan.Pending {
		b, err := fs.ReadFile(source, "migrations/"+name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		hash := hex.EncodeToString(sum[:])
		if _, err = tx.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(name,sha256) VALUES($1,$2)", name, hash); err != nil {
			return err
		}
	}
	// Existing installations must reconcile the post-DDL state with fresh
	// processes. Fresh installs keep the initial gate defined by the migrations.
	if plan.Applied > 0 || controlled {
		var hasControl bool
		if err = tx.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.ops_control') IS NOT NULL").Scan(&hasControl); err != nil {
			return err
		}
		if hasControl {
			if _, err = tx.Exec(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='open' THEN 'maintenance' ELSE mode END,generation=gen_random_uuid(),reconciliation_id=NULL,changed_at=clock_timestamp() WHERE singleton"); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
func Settings(ctx context.Context, p *pgxpool.Pool) (config.Settings, error) {
	var b []byte
	if err := p.QueryRow(ctx, "SELECT value FROM app_settings WHERE active").Scan(&b); err != nil {
		return config.Settings{}, err
	}
	return config.ParseSettings(b)
}

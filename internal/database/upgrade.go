package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpgradePlan struct {
	Applied    int      `json:"applied"`
	Pending    []string `json:"pending"`
	Compatible bool     `json:"compatible"`
}

// CheckUpgrade never creates or changes schema; mismatched/unknown applied
// migrations prohibit downgrading a newer database with an older executable.
func CheckUpgrade(ctx context.Context, db *pgxpool.Pool) (UpgradePlan, error) {
	tx, e := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return UpgradePlan{}, e
	}
	defer tx.Rollback(context.Background())
	plan, e := migrationPlan(ctx, tx, migrations)
	if e != nil {
		return plan, e
	}
	return plan, tx.Commit(ctx)
}

// The applied ledger must be a complete prefix, never an arbitrary subset.
// Startup uses the same validation while holding the migration lock.
func migrationPlan(ctx context.Context, tx pgx.Tx, source fs.FS) (UpgradePlan, error) {
	plan := UpgradePlan{Pending: []string{}, Compatible: false}
	var exists bool
	if e := tx.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.schema_migrations') IS NOT NULL").Scan(&exists); e != nil {
		return plan, e
	}
	applied := map[string]string{}
	if exists {
		rows, e := tx.Query(ctx, "SELECT name,sha256 FROM schema_migrations")
		if e != nil {
			return plan, e
		}
		for rows.Next() {
			var name, hash string
			if e = rows.Scan(&name, &hash); e != nil {
				rows.Close()
				return plan, e
			}
			applied[name] = hash
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return plan, e
		}
	}
	entries, e := fs.ReadDir(source, "migrations")
	if e != nil {
		return plan, e
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		raw, e := fs.ReadFile(source, "migrations/"+entry.Name())
		if e != nil {
			return plan, e
		}
		digest := sha256.Sum256(raw)
		if got, ok := applied[entry.Name()]; ok {
			if got != hex.EncodeToString(digest[:]) {
				return plan, errors.New("migration checksum mismatch")
			}
			if len(plan.Pending) != 0 {
				return plan, errors.New("migration history is not a complete prefix")
			}
			plan.Applied++
			delete(applied, entry.Name())
		} else {
			plan.Pending = append(plan.Pending, entry.Name())
		}
	}
	if len(applied) > 0 {
		return plan, errors.New("database contains unknown newer migrations")
	}
	plan.Compatible = true
	return plan, nil
}

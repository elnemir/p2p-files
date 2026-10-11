package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	plan := UpgradePlan{Pending: []string{}, Compatible: true}
	tx, e := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return plan, e
	}
	defer tx.Rollback(context.Background())
	var exists bool
	if e = tx.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.schema_migrations') IS NOT NULL").Scan(&exists); e != nil {
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
	entries, e := migrations.ReadDir("migrations")
	if e != nil {
		return plan, e
	}
	for _, entry := range entries {
		raw, e := migrations.ReadFile("migrations/" + entry.Name())
		if e != nil {
			return plan, e
		}
		digest := sha256.Sum256(raw)
		if got, ok := applied[entry.Name()]; ok {
			if got != hex.EncodeToString(digest[:]) {
				plan.Compatible = false
				return plan, errors.New("migration checksum mismatch")
			}
			plan.Applied++
			delete(applied, entry.Name())
		} else {
			plan.Pending = append(plan.Pending, entry.Name())
		}
	}
	if len(applied) > 0 {
		plan.Compatible = false
		return plan, errors.New("database contains unknown newer migrations")
	}
	return plan, tx.Commit(ctx)
}

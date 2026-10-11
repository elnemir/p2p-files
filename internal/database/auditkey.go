package database

import (
	"context"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditKey preserves the pre-rotation chain key when the master KEK changes.
func AuditKey(ctx context.Context, db *pgxpool.Pool, kek []byte) ([]byte, error) {
	tx, e := db.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184994)"); e != nil {
		return nil, e
	}
	var box []byte
	e = tx.QueryRow(ctx, "SELECT envelope FROM bootstrap_secrets WHERE name='audit_key'").Scan(&box)
	if errors.Is(e, pgx.ErrNoRows) {
		tag, err := db.Exec(ctx, "UPDATE crypto_usage SET seal_count=seal_count+1 WHERE name='kek' AND seal_count<1048576")
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() != 1 {
			return nil, errors.New("KEK usage budget exhausted")
		}
		key, err := hkdf.Key(sha256.New, kek, nil, "filex/audit/key/v1", 32)
		if err != nil {
			return nil, err
		}
		box, e = cryptobox.Seal(kek, "filex/bootstrap/v1/audit_key", key)
		if e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO bootstrap_secrets(name,envelope) VALUES('audit_key',$1)", box); e != nil {
			return nil, e
		}
	} else if e != nil {
		return nil, e
	}
	key, e := cryptobox.Open(kek, "filex/bootstrap/v1/audit_key", box)
	if e != nil {
		return nil, e
	}
	return key, tx.Commit(ctx)
}

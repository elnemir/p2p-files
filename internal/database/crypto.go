package database

import (
	"bytes"
	"context"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CheckKEK(ctx context.Context, p *pgxpool.Pool, key []byte) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184994)"); err != nil {
		return err
	}
	const aad = "filex/bootstrap/v1/key_check"
	expected := []byte("filex bootstrap key verification v1")
	var envelope []byte
	err = tx.QueryRow(ctx, "SELECT envelope FROM bootstrap_secrets WHERE name='key_check'").Scan(&envelope)
	if errors.Is(err, pgx.ErrNoRows) {
		envelope, err = cryptobox.Seal(key, aad, expected)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO bootstrap_secrets(name,envelope) VALUES('key_check',$1)", envelope); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	plaintext, err := cryptobox.Open(key, aad, envelope)
	if err != nil || !bytes.Equal(plaintext, expected) {
		return errors.New("critical key check failed")
	}
	return tx.Commit(ctx)
}

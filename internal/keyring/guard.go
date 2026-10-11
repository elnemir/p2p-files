package keyring

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

var ErrChanged = errors.New("active key changed; retry operation")

func RequireActive(ctx context.Context, tx pgx.Tx, id string) error {
	var value string
	e := tx.QueryRow(ctx, "SELECT id::text FROM key_versions WHERE id=$1 AND active FOR SHARE", id).Scan(&value)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrChanged
	}
	return e
}

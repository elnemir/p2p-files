package keyring

import (
	"context"
	"errors"
	"filex.local/filex/internal/cryptobox"
)

var ErrUsage = errors.New("encryption key usage exhausted; rotate the key")

// Reservations commit before encryption and are not refunded on failure.
func (r Ring) Reserve(ctx context.Context, id string, count int) error {
	if count < 1 || count > 1048576 {
		return ErrUsage
	}
	tag, e := r.DB.Exec(ctx, "UPDATE key_versions SET seal_count=seal_count+$2 WHERE id=$1 AND seal_count<=1048576-$2", id, count)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrUsage
	}
	return nil
}
func (r Ring) reserveKEK(ctx context.Context) error {
	tag, e := r.DB.Exec(ctx, "UPDATE crypto_usage SET seal_count=seal_count+1 WHERE name='kek' AND seal_count<1048576")
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrUsage
	}
	return nil
}
func (r Ring) Seal(ctx context.Context, id string, key []byte, aad string, plain []byte) ([]byte, error) {
	if e := r.Reserve(ctx, id, 1); e != nil {
		return nil, e
	}
	return cryptobox.Seal(key, aad, plain)
}

package keyring

import (
	"context"
	"crypto/ed25519"
)

// Signing retains every public key even when the active seed rotates.
func (r Ring) Signing(ctx context.Context) (string, []byte, error) {
	id, seed, e := r.Active(ctx, "signing")
	if e != nil {
		return "", nil, e
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	_, e = r.DB.Exec(ctx, "INSERT INTO signing_keys(id,seed_key_id,public_key) VALUES($1,$1,$2) ON CONFLICT(seed_key_id) DO NOTHING", id, []byte(pub))
	return id, seed, e
}

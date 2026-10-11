package storage

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
)

var ErrConfig = errors.New("invalid provider configuration")

// Register shares encrypted provisioning between the HTTP ADMIN and host CLI.
func (r Registry) Register(ctx context.Context, d Descriptor, purpose string, priority int, auditKey []byte, actor string) (string, error) {
	if purpose != "payload" && purpose != "artifacts" && purpose != "backups" {
		return "", ErrConfig
	}
	provider, e := d.Open()
	if e != nil {
		return "", ErrConfig
	}
	id := domain.NewID()
	var raw, box []byte
	var kid *string
	if d.Kind == "local" {
		raw, e = json.Marshal(map[string]string{"root": d.Root})
	} else {
		keyID, key, err := r.Keys.Active(ctx, "configuration")
		if err != nil {
			return "", err
		}
		kid = &keyID
		secret, err := json.Marshal(d.S3)
		if err != nil {
			return "", err
		}
		box, e = r.Keys.Seal(ctx, keyID, key, "filex/storage/v1/"+id, secret)
		raw = []byte(`{}`)
	}
	if e != nil {
		return "", e
	}
	probe := domain.NewID()
	if e = provider.Put(ctx, probe, []byte("filex-storage-probe")); e != nil {
		return "", e
	}
	b, e := provider.Get(ctx, probe)
	del := provider.Delete(ctx, probe)
	if e != nil {
		return "", e
	}
	if del != nil {
		return "", del
	}
	if string(b) != "filex-storage-probe" {
		return "", errors.New("provider probe mismatch")
	}
	tx, e := r.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	if kid != nil {
		if e = keyring.RequireActive(ctx, tx, *kid); e != nil {
			return "", e
		}
	}
	if _, e = tx.Exec(ctx, "INSERT INTO storage_providers(id,kind,config,priority,secret_key_id,secret_envelope,purpose) VALUES($1,$2,$3,$4,$5,$6,$7)", id, d.Kind, raw, priority, kid, box, purpose); e != nil {
		return "", e
	}
	if e = audit.Append(ctx, tx, auditKey, actor, "storage.created", id); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}

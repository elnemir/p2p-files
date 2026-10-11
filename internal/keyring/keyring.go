package keyring

import (
	"context"
	"crypto/rand"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Ring struct {
	DB  *pgxpool.Pool
	KEK []byte
}

func (r Ring) Active(ctx context.Context, class string) (string, []byte, error) {
	if class != "payload" && class != "artifacts" && class != "configuration" && class != "backups" && class != "signing" {
		return "", nil, errors.New("invalid key class")
	}
	tx, e := r.DB.Begin(ctx)
	if e != nil {
		return "", nil, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184997)"); e != nil {
		return "", nil, e
	}
	var id string
	var box []byte
	e = tx.QueryRow(ctx, "SELECT id::text,wrapped FROM key_versions WHERE class=$1 AND active", class).Scan(&id, &box)
	if errors.Is(e, pgx.ErrNoRows) {
		id = domain.NewID()
		key := make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return "", nil, e
		}
		if e = r.reserveKEK(ctx); e != nil {
			return "", nil, e
		}
		box, e = cryptobox.Seal(r.KEK, "filex/keys/v1/"+id, key)
		if e != nil {
			return "", nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO key_versions(id,class,version,wrapped,active) VALUES($1,$2,1,$3,true)", id, class, box); e != nil {
			return "", nil, e
		}
	} else if e != nil {
		return "", nil, e
	}
	key, e := cryptobox.Open(r.KEK, "filex/keys/v1/"+id, box)
	if e != nil {
		return "", nil, e
	}
	return id, key, tx.Commit(ctx)
}
func (r Ring) Get(ctx context.Context, id string) ([]byte, error) {
	var box []byte
	if e := r.DB.QueryRow(ctx, "SELECT wrapped FROM key_versions WHERE id=$1", id).Scan(&box); e != nil {
		return nil, e
	}
	return cryptobox.Open(r.KEK, "filex/keys/v1/"+id, box)
}
func (r Ring) Rotate(ctx context.Context, class string) (string, error) {
	return r.RotateWith(ctx, class, nil)
}
func (r Ring) RotateWith(ctx context.Context, class string, hook func(pgx.Tx, string) error) (string, error) {
	if _, _, e := r.Active(ctx, class); e != nil {
		return "", e
	}
	tx, e := r.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184997)"); e != nil {
		return "", e
	}
	var version int64
	if e = tx.QueryRow(ctx, "SELECT max(version)+1 FROM key_versions WHERE class=$1", class).Scan(&version); e != nil {
		return "", e
	}
	id := domain.NewID()
	key := make([]byte, 32)
	if _, e = rand.Read(key); e != nil {
		return "", e
	}
	if e = r.reserveKEK(ctx); e != nil {
		return "", e
	}
	box, e := cryptobox.Seal(r.KEK, "filex/keys/v1/"+id, key)
	if e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "UPDATE key_versions SET active=false WHERE class=$1", class); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO key_versions(id,class,version,wrapped,active) VALUES($1,$2,$3,$4,true)", id, class, version, box); e != nil {
		return "", e
	}
	if hook != nil {
		if e = hook(tx, id); e != nil {
			return "", e
		}
	}
	return id, tx.Commit(ctx)
}

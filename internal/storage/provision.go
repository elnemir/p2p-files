package storage

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/domain"
)

func (r Registry) ProvisionLocal(ctx context.Context, root string) (string, error) {
	return r.Provision(ctx, root, "payload")
}
func (r Registry) Provision(ctx context.Context, root, purpose string) (string, error) {
	p := Local{root}
	probe := domain.NewID()
	if e := p.Put(ctx, probe, []byte("filex-probe")); e != nil {
		return "", e
	}
	if e := p.Delete(ctx, probe); e != nil {
		return "", e
	}
	tx, e := r.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184998)"); e != nil {
		return "", e
	}
	var existing *string
	if e = tx.QueryRow(ctx, "SELECT min(id::text) FROM storage_providers WHERE kind='local' AND config->>'root'=$1 AND purpose=$2", root, purpose).Scan(&existing); e != nil {
		return "", e
	}
	if existing != nil {
		return *existing, tx.Commit(ctx)
	}
	raw, e := json.Marshal(map[string]string{"root": root})
	if e != nil {
		return "", e
	}
	id := domain.NewID()
	if _, e = tx.Exec(ctx, "INSERT INTO storage_providers(id,kind,config,purpose) VALUES($1,'local',$2,$3)", id, raw, purpose); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}

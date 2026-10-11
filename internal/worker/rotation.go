package worker

import (
	"context"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/cryptobox"
	"github.com/jackc/pgx/v5"
)

func (w Worker) Rotate(ctx context.Context) error {
	var job, class, target string
	e := w.DB.QueryRow(ctx, "SELECT id::text,class,target_key_id::text FROM rotation_jobs WHERE state='running' AND class<>'payload' ORDER BY checked_at,created_at LIMIT 1").Scan(&job, &class, &target)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if _, e = w.DB.Exec(ctx, "UPDATE rotation_jobs SET checked_at=clock_timestamp() WHERE id=$1", job); e != nil {
		return e
	}
	key, e := w.Keys.Get(ctx, target)
	if e != nil {
		return e
	}
	type ref struct{ table, key, box, context, where string }
	var refs []ref
	if class == "configuration" {
		refs = []ref{{"recipient_links", "key_id", "token_envelope", "link", "true"}, {"storage_providers", "secret_key_id", "secret_envelope", "storage", "secret_key_id IS NOT NULL"}, {"communication_configs", "key_id", "envelope", "communications", "true"}, {"security_incident_reports", "key_id", "envelope", "security_report", "true"}}
	} else if class == "artifacts" {
		refs = []ref{{"reports", "key_id", "snapshot_envelope", "report", "state<>'purged'"}}
	}
	for _, ref := range refs {
		tx, e := w.DB.Begin(ctx)
		if e != nil {
			return e
		}
		var jobState string
		if e = tx.QueryRow(ctx, "SELECT state FROM rotation_jobs WHERE id=$1 FOR UPDATE", job).Scan(&jobState); e != nil {
			tx.Rollback(ctx)
			return e
		}
		if jobState != "running" {
			tx.Rollback(ctx)
			return nil
		}
		var id, kid string
		var box []byte
		e = tx.QueryRow(ctx, "SELECT id::text,"+ref.key+"::text,"+ref.box+" FROM "+ref.table+" WHERE "+ref.key+"<>$1 AND "+ref.where+" ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1", target).Scan(&id, &kid, &box)
		if errors.Is(e, pgx.ErrNoRows) {
			tx.Rollback(ctx)
			continue
		}
		if e != nil {
			tx.Rollback(ctx)
			return e
		}
		old, e := w.Keys.Get(ctx, kid)
		if e != nil {
			tx.Rollback(ctx)
			return e
		}
		plain, e := cryptobox.Open(old, "filex/"+ref.context+"/v1/"+id, box)
		if e != nil {
			tx.Rollback(ctx)
			return e
		}
		box, e = w.Keys.Seal(ctx, target, key, "filex/"+ref.context+"/v1/"+id, plain)
		if e != nil {
			tx.Rollback(ctx)
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE "+ref.table+" SET "+ref.key+"=$2,"+ref.box+"=$3 WHERE id=$1", id, target, box); e != nil {
			tx.Rollback(ctx)
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE rotation_jobs SET processed=processed+1 WHERE id=$1", job); e != nil {
			tx.Rollback(ctx)
			return e
		}
		return tx.Commit(ctx)
	}
	if class == "artifacts" {
		store := artifacts.Store{DB: w.DB, Keys: w.Keys, Storage: w.Files.Storage}
		changed, e := store.RotateOne(ctx, target, job)
		if e != nil {
			return e
		}
		if changed {
			_, e = w.DB.Exec(ctx, "UPDATE rotation_jobs SET processed=processed+1 WHERE id=$1", job)
			return e
		}
		var pending bool
		if e = w.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM artifacts WHERE key_id<>$1 AND state='ready' AND expires_at>clock_timestamp())", target).Scan(&pending); e != nil {
			return e
		}
		if pending {
			return nil
		}
	}
	for _, ref := range refs {
		var pending bool
		if e = w.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM "+ref.table+" WHERE "+ref.key+"<>$1 AND "+ref.where+")", target).Scan(&pending); e != nil {
			return e
		}
		if pending {
			return nil
		}
	}
	_, e = w.DB.Exec(ctx, "UPDATE rotation_jobs SET state='completed' WHERE id=$1 AND state='running'", job)
	return e
}

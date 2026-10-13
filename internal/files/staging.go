package files

import (
	"context"
	"errors"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
)

// AbortStage never touches a visible resource, even after credential revocation.
// Administrative holds still win. It is used by failed requests and crash cleanup.
func (s Service) AbortStage(ctx context.Context, owner, id string) error {
	f, e := s.Get(ctx, owner, id)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var ok bool
	if e = tx.QueryRow(ctx, "SELECT state='draft' AND NOT frozen FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&ok); e != nil {
		return e
	}
	if !ok {
		return transfer.ErrConflict
	}
	tag, e := tx.Exec(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key='#cancelled:'||id::text WHERE id=$1 AND staged_until IS NOT NULL AND NOT frozen", id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrConflict
	}
	if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT $1,object_key,file_id FROM file_chunks WHERE file_id=$2 ON CONFLICT(object_key) DO NOTHING", f.ProviderID, id); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s Service) cleanupStages(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var parent string
	e = tx.QueryRow(ctx, `SELECT t.id::text FROM transfers t WHERE t.state='draft' AND NOT t.frozen AND EXISTS(SELECT 1 FROM files f WHERE f.transfer_id=t.id AND f.staged_until<=clock_timestamp() AND f.state NOT IN ('cancelled','deleted') AND NOT f.frozen) ORDER BY t.id FOR UPDATE OF t SKIP LOCKED LIMIT 1`).Scan(&parent)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	rows, e := tx.Query(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key='#cancelled:'||id::text WHERE transfer_id=$1 AND staged_until<=clock_timestamp() AND state NOT IN ('cancelled','deleted') AND NOT frozen RETURNING id::text", parent)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.id=$1 ON CONFLICT(object_key) DO NOTHING", id); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

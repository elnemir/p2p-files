package files

import (
	"context"
	"errors"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"time"
)

// Sweep takes the same parent lock as publish, completion, revoke and freezes.
func (s Service) Sweep(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	rows, e := tx.Query(ctx, `SELECT id::text FROM transfers WHERE state='active' AND (absolute_expires_at<=clock_timestamp() OR (NOT frozen AND expires_at<=clock_timestamp())) ORDER BY absolute_expires_at FOR UPDATE SKIP LOCKED LIMIT 50`)
	if e != nil {
		return e
	}
	var ids []string
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
		if _, e = tx.Exec(ctx, "UPDATE transfers SET state='expired',access_epoch=access_epoch+1 WHERE id=$1", id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE files SET state='deleting' WHERE transfer_id=$1 AND state NOT IN ('deleted','cancelled','integrity_failed')", id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.transfer_id=$1 AND f.state<>'integrity_failed' ON CONFLICT(object_key) DO NOTHING`, id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) VALUES($1,'transfer.expired',$2,'{}') ON CONFLICT(deduplication_key) DO NOTHING`, id, "expired:"+id); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

// Cleanup holds the transfer lock through bounded physical deletion. A hold that
// commits before this transaction prevents deletion; an already deleted byte cannot be restored.
func (s Service) Cleanup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if e := s.cleanupEmpty(ctx); e != nil {
		return e
	}
	var id int64
	var transferID *string
	e := s.DB.QueryRow(ctx, `SELECT j.id,f.transfer_id::text FROM deletion_jobs j LEFT JOIN files f ON f.id=j.file_id LEFT JOIN transfers t ON t.id=f.transfer_id WHERE j.completed_at IS NULL AND j.available_at<=clock_timestamp() AND NOT COALESCE(f.write_lease_until>clock_timestamp(),false) AND NOT COALESCE(f.frozen,false) AND NOT COALESCE(t.frozen,false) ORDER BY j.id LIMIT 1`).Scan(&id, &transferID)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if transferID != nil {
		var frozen bool
		if e = tx.QueryRow(ctx, "SELECT frozen FROM transfers WHERE id=$1 FOR UPDATE", *transferID).Scan(&frozen); e != nil {
			return e
		}
		if frozen {
			return nil
		}
	}
	// Keep the retention lock alive for the entire bounded storage request.
	// The pool's normal 10-second idle timeout is shorter than S3's deadline.
	if _, e = tx.Exec(ctx, "SET LOCAL idle_in_transaction_session_timeout='30s'"); e != nil {
		return e
	}
	var provider, object string
	var fileID *string
	e = tx.QueryRow(ctx, `SELECT j.provider_id::text,j.object_key,j.file_id::text FROM deletion_jobs j LEFT JOIN files f ON f.id=j.file_id WHERE j.id=$1 AND j.completed_at IS NULL AND j.available_at<=clock_timestamp() AND NOT COALESCE(f.write_lease_until>clock_timestamp(),false) AND NOT COALESCE(f.frozen,false) FOR UPDATE OF j SKIP LOCKED`, id).Scan(&provider, &object, &fileID)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	p, e := s.Storage.Get(ctx, provider)
	if e == nil {
		e = p.Delete(ctx, object)
	}
	if e != nil {
		if _, retry := tx.Exec(ctx, "UPDATE deletion_jobs SET attempts=attempts+1,available_at=clock_timestamp()+interval '1 minute' WHERE id=$1", id); retry != nil {
			return retry
		}
		if commit := tx.Commit(ctx); commit != nil {
			return commit
		}
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE deletion_jobs SET completed_at=clock_timestamp(),attempts=attempts+1 WHERE id=$1", id); e != nil {
		return e
	}
	if fileID != nil {
		if _, e = tx.Exec(ctx, `UPDATE files SET state='deleted' WHERE id=$1 AND state='deleting' AND NOT EXISTS(SELECT 1 FROM file_chunks ch LEFT JOIN deletion_jobs j ON j.object_key=ch.object_key WHERE ch.file_id=$1 AND j.completed_at IS NULL)`, *fileID); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (s Service) MarkFailure(ctx context.Context, f File, state string) error {
	if state != "integrity_failed" && state != "storage_missing" {
		return transfer.ErrConflict
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var parent string
	if e = tx.QueryRow(ctx, "SELECT state FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&parent); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "UPDATE files SET state=$2 WHERE id=$1 AND state IN ('ready','leased','finalizing') AND provider_id::text=$3 AND key_id::text=$4 AND salt=$5", f.ID, state, f.ProviderID, f.KeyID, f.Salt)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s Service) cleanupEmpty(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var parent string
	e = tx.QueryRow(ctx, "SELECT t.id::text FROM transfers t WHERE NOT t.frozen AND EXISTS(SELECT 1 FROM files f WHERE f.transfer_id=t.id AND f.state='deleting' AND NOT f.frozen AND NOT EXISTS(SELECT 1 FROM file_chunks c WHERE c.file_id=f.id)) ORDER BY t.id FOR UPDATE OF t SKIP LOCKED LIMIT 1").Scan(&parent)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE files SET state='deleted' WHERE transfer_id=$1 AND state='deleting' AND NOT frozen AND NOT EXISTS(SELECT 1 FROM file_chunks c WHERE c.file_id=files.id)", parent); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

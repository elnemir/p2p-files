package worker

import (
	"context"
	"errors"
	"filex.local/filex/internal/audit"
	"github.com/jackc/pgx/v5"
)

func (w Worker) Bulk(ctx context.Context) error {
	tx, e := w.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var job, actor, action, role string
	var admin, disabled bool
	var epoch, current int64
	e = tx.QueryRow(ctx, "SELECT j.id::text,j.owner_id::text,j.action,j.admin_scope,j.account_epoch,u.auth_epoch,u.role,u.disabled FROM bulk_jobs j JOIN users u ON u.id=j.owner_id WHERE j.state IN ('queued','running') ORDER BY j.created_at FOR UPDATE OF j SKIP LOCKED FOR SHARE OF u LIMIT 1").Scan(&job, &actor, &action, &admin, &epoch, &current, &role, &disabled)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if disabled || epoch != current || admin && role != "admin" {
		if _, e = tx.Exec(ctx, "UPDATE bulk_jobs SET state='cancelled' WHERE id=$1", job); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	var id string
	e = tx.QueryRow(ctx, "SELECT subject_id::text FROM bulk_items WHERE job_id=$1 AND state='pending' ORDER BY subject_id FOR UPDATE SKIP LOCKED LIMIT 1", job).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		if _, e = tx.Exec(ctx, "UPDATE bulk_jobs SET state='completed' WHERE id=$1", job); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	if e != nil {
		return e
	}
	var parent, owner, state string
	if action == "revoke" {
		e = tx.QueryRow(ctx, "SELECT id::text,owner_id::text,state FROM transfers WHERE id=$1 FOR UPDATE", id).Scan(&parent, &owner, &state)
	} else {
		e = tx.QueryRow(ctx, "SELECT t.id::text,t.owner_id::text,t.state FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id=$1 FOR UPDATE OF t", id).Scan(&parent, &owner, &state)
	}
	if e != nil {
		return e
	}
	result := "done"
	if owner != actor && !admin || action == "cancel_upload" && state != "draft" {
		result = "conflict"
	} else {
		if action == "revoke" {
			if _, e = tx.Exec(ctx, "UPDATE transfers SET state='revoked',access_epoch=access_epoch+1 WHERE id=$1", parent); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "UPDATE files SET state='deleting',upload_epoch=upload_epoch+1 WHERE transfer_id=$1 AND state NOT IN ('deleted','cancelled')", parent); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.transfer_id=$1 ON CONFLICT(object_key) DO NOTHING", parent); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) VALUES($1,'transfer.revoked',$2,'{}') ON CONFLICT(deduplication_key) DO NOTHING", parent, "revoked:"+parent); e != nil {
				return e
			}
		} else {
			if _, e = tx.Exec(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key=path_key||'#cancelled:'||id::text WHERE id=$1 AND state<>'cancelled'", id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.id=$1 ON CONFLICT(object_key) DO NOTHING", id); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", parent); e != nil {
				return e
			}
		}
		if e = audit.Append(ctx, tx, w.AuditKey, actor, "bulk."+action, id+":"+job); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE bulk_items SET state=$3 WHERE job_id=$1 AND subject_id=$2", job, id, result); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE bulk_jobs SET state=CASE WHEN EXISTS(SELECT 1 FROM bulk_items WHERE job_id=$1 AND state='pending') THEN 'running' ELSE 'completed' END WHERE id=$1", job); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

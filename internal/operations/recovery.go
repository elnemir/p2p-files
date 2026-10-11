package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/maintenance"
	"github.com/jackc/pgx/v5"
	"io"
)

func (s Service) runnerLock(ctx context.Context) (func(), error) {
	c, e := s.DB.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	var ok bool
	if e = c.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended(current_schema()||'/filex/operations/v1',0))").Scan(&ok); e != nil || !ok {
		c.Release()
		if e != nil {
			return nil, e
		}
		return nil, ErrConflict
	}
	state, e := s.State(ctx)
	if e != nil || state.Generation != s.Generation {
		cleanup, cancel := boundedCleanup()
		defer cancel()
		_ = c.Conn().Close(cleanup)
		c.Release()
		if e != nil {
			return nil, e
		}
		return nil, maintenance.ErrRestart
	}
	return func() {
		cleanup, cancel := boundedCleanup()
		defer cancel()
		if _, err := c.Exec(cleanup, "SELECT pg_advisory_unlock(hashtextextended(current_schema()||'/filex/operations/v1',0))"); err != nil {
			_ = c.Conn().Close(cleanup)
		}
		c.Release()
	}, nil
}
func (s Service) Run(ctx context.Context) error { return s.Execute(ctx, "", nil) }
func (s Service) Execute(ctx context.Context, requested string, recoveryKEK []byte) error {
	state, e := s.State(ctx)
	if e != nil {
		return e
	}
	if state.Generation != s.Generation {
		return maintenance.ErrRestart
	}
	unlock, e := s.runnerLock(ctx)
	if errors.Is(e, ErrConflict) {
		return nil
	}
	if e != nil {
		return e
	}
	defer unlock()
	return s.executeLocked(ctx, requested, recoveryKEK)
}
func (s Service) QueueAndExecute(ctx context.Context, actor, kind, source, provider, reason string, recoveryKEK []byte) (string, error) {
	state, e := s.State(ctx)
	if e != nil {
		return "", e
	}
	if state.Generation != s.Generation {
		return "", maintenance.ErrRestart
	}
	unlock, e := s.runnerLock(ctx)
	if e != nil {
		return "", e
	}
	defer unlock()
	id, e := s.Queue(ctx, actor, kind, source, provider, reason)
	if e != nil {
		return "", e
	}
	return id, s.executeLocked(ctx, id, recoveryKEK)
}
func (s Service) executeLocked(ctx context.Context, requested string, recoveryKEK []byte) error {
	var e error
	// A session lock is lost on crash. No previously running destructive job
	// is replayed implicitly; an atomic import has either committed or rolled back.
	if _, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET state='failed',error_code='interrupted',completed_at=clock_timestamp() WHERE state='running'"); e != nil {
		return e
	}
	var id, kind, actor, reason string
	var source *string
	e = s.DB.QueryRow(ctx, "UPDATE ops_jobs SET state='running',started_at=clock_timestamp() WHERE id=(SELECT id FROM ops_jobs WHERE state='queued' AND ($1='' OR id::text=$1) ORDER BY created_at,id LIMIT 1) RETURNING id::text,kind,actor,reason,source_id::text", requested).Scan(&id, &kind, &actor, &reason, &source)
	if errors.Is(e, pgx.ErrNoRows) {
		if requested != "" {
			return ErrConflict
		}
		return nil
	}
	if e != nil {
		return e
	}
	switch kind {
	case "backup":
		e = s.backup(ctx, id)
	case "restore":
		if source == nil {
			e = ErrInvalid
		} else {
			e = s.restore(ctx, id, *source, actor, reason, recoveryKEK)
		}
	case "reconcile":
		var report Reconciliation
		report, e = s.Reconcile(ctx, actor, reason)
		if e == nil {
			_, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET result=$2 WHERE id=$1", id, map[string]string{"reconciliationId": report.ID, "hash": report.Hash})
		}
	case "audit":
		e = errors.Join(s.CheckAudit(ctx), s.CheckCrypto(ctx))
	case "storage":
		e = s.CheckStorage(ctx)
	}
	status := "completed"
	var code *string
	if e != nil {
		status = "failed"
		v := "operation_failed"
		if errors.Is(e, ErrBackup) {
			v = "backup_invalid"
		}
		code = &v
	}
	// Recovery remains closed even if cancellation prevents this final write.
	cleanup, cancel := boundedCleanup()
	defer cancel()
	tx, finish := s.DB.Begin(cleanup)
	if finish != nil {
		return errors.Join(e, finish)
	}
	defer tx.Rollback(context.Background())
	if _, finish = tx.Exec(cleanup, "UPDATE ops_jobs SET state=$2,error_code=$3,completed_at=clock_timestamp() WHERE id=$1 AND state='running'", id, status, code); finish != nil {
		return errors.Join(e, finish)
	}
	if finish = s.event(cleanup, tx, actor, kind+"."+status, id, reason); finish != nil {
		return errors.Join(e, finish)
	}
	return errors.Join(e, tx.Commit(cleanup))
}
func (s Service) backup(ctx context.Context, id string) error {
	if e := database.CheckKEK(ctx, s.DB, s.Keys.KEK); e != nil {
		return e
	}
	p, e := s.jobProvider(ctx, id)
	if e != nil {
		return e
	}
	if e = s.writeArchive(ctx, id, p, func(w io.Writer) error { return s.snapshot(ctx, w) }); e != nil {
		return e
	}
	r, e := readArchive(ctx, p, id, s.Keys.KEK)
	if e != nil {
		return e
	}
	if e = s.validateSnapshot(ctx, r); e != nil {
		return e
	}
	_, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET progress=$2,result=$3 WHERE id=$1", id, r.manifest.Bytes, map[string]any{"sha256": r.manifest.SHA, "bytes": r.manifest.Bytes, "verified": true})
	return e
}
func (s Service) preBackup(ctx context.Context, parent, actor, reason string) (string, error) {
	id := domain.NewID()
	var provider string
	var box []byte
	if e := s.DB.QueryRow(ctx, "SELECT provider_id::text,provider_envelope FROM ops_jobs WHERE id=$1", parent).Scan(&provider, &box); e != nil {
		return "", e
	}
	raw, e := s.open("filex/ops_provider/v1/"+parent, box)
	if e != nil {
		return "", e
	}
	box, e = s.seal("filex/ops_provider/v1/"+id, raw)
	if e != nil {
		return "", e
	}
	if _, e = s.DB.Exec(ctx, "INSERT INTO ops_jobs(id,kind,state,actor,reason,provider_id,provider_envelope,started_at) VALUES($1,'backup','running',$2,$3,$4,$5,clock_timestamp())", id, actor, reason, provider, box); e != nil {
		return "", e
	}
	if _, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET pre_backup_id=$2 WHERE id=$1", parent, id); e != nil {
		return "", e
	}
	if e = s.backup(ctx, id); e != nil {
		_, mark := s.DB.Exec(ctx, "UPDATE ops_jobs SET state='failed',error_code='pre_backup_failed',completed_at=clock_timestamp() WHERE id=$1", id)
		return "", errors.Join(e, mark)
	}
	_, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET state='completed',completed_at=clock_timestamp() WHERE id=$1", id)
	return id, e
}
func (s Service) restore(ctx context.Context, id, source, actor, reason string, recoveryKEK []byte) error {
	if len(recoveryKEK) == 0 {
		recoveryKEK = s.Keys.KEK
	}
	if len(recoveryKEK) != 32 {
		return ErrInvalid
	}
	if e := s.SetMaintenance(ctx, actor, reason); e != nil {
		return e
	}
	unlock, e := maintenance.Exclusive(ctx, s.DB)
	if e != nil {
		return e
	}
	defer unlock()
	p, e := s.jobProvider(ctx, source)
	if e != nil {
		return e
	}
	r, e := readArchive(ctx, p, source, recoveryKEK)
	if e != nil {
		return e
	}
	if e = s.validateSnapshot(ctx, r); e != nil {
		return e
	}
	r, e = readArchive(ctx, p, source, recoveryKEK)
	if e != nil {
		return e
	}
	cryptoInfo, e := inspectCryptoSnapshot(r)
	if e != nil {
		return e
	}
	if e = s.rememberProviders(ctx); e != nil {
		return e
	}
	pre, e := s.preBackup(ctx, id, actor, "Pre-restore backup for "+id)
	if e != nil {
		return e
	}
	reserved, e := s.reserveRestoreWraps(ctx, cryptoInfo, recoveryKEK)
	if e != nil {
		return e
	}
	if _, e = s.DB.Exec(ctx, "UPDATE ops_control SET mode='restoring',restore_id=$1,reconciliation_id=NULL,changed_at=clock_timestamp()", id); e != nil {
		return e
	}
	r, e = readArchive(ctx, p, source, recoveryKEK)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='5min'; SET LOCAL idle_in_transaction_session_timeout='60s'"); e != nil {
		return e
	}
	if e = s.importSnapshot(ctx, tx, r, recoveryKEK); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE crypto_usage SET seal_count=$1 WHERE name='kek'", reserved); e != nil {
		return e
	}
	if e = freshRestoredKeys(ctx, tx, s.Keys.KEK); e != nil {
		return e
	}
	for _, q := range []string{
		"UPDATE web_sessions SET revoked_at=clock_timestamp()",
		"UPDATE recipient_sessions SET expires_at=clock_timestamp()",
		"UPDATE artifact_credentials SET expires_at=clock_timestamp(),state='completed',session_hash=NULL,writer_until=NULL,fence=fence+1",
		"UPDATE downloads SET state='cancelled',writer_until=NULL,fence=fence+1 WHERE state='leased'",
		"UPDATE files SET state=CASE WHEN state='leased' THEN 'ready' WHEN state IN ('uploading','finalizing') THEN 'cancelled' ELSE state END,write_lease_until=NULL,upload_epoch=upload_epoch+1",
		"UPDATE transfers SET access_epoch=access_epoch+1",
		"UPDATE artifacts SET state='deleting' WHERE state='writing'",
		"UPDATE artifacts SET maintenance_until=NULL,maintenance_epoch=maintenance_epoch+1",
		"UPDATE deletion_jobs SET lease_until=NULL,lease_epoch=lease_epoch+1",
		"UPDATE export_jobs SET state='cancelled',lease_until=NULL,fence=fence+1 WHERE state NOT IN ('completed','cancelled','failed')",
		"UPDATE bulk_jobs SET state='paused' WHERE state IN ('running','queued')",
		"UPDATE storage_migrations SET state='paused' WHERE state='running'",
		"UPDATE rotation_jobs SET state='paused' WHERE state='running'",
		"UPDATE reports SET render_state='pending',render_lease_until=NULL,render_fence=render_fence+1 WHERE render_state='running'",
		"UPDATE deliveries SET state='cancelled',lease_until=NULL,fence=fence+1 WHERE state IN ('pending','sending')",
		"UPDATE outbox SET delivered_at=clock_timestamp() WHERE delivered_at IS NULL",
	} {
		if _, e = tx.Exec(ctx, q); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode='reconcile',generation=gen_random_uuid(),reconciliation_id=NULL,restore_id=$1,changed_at=clock_timestamp()", id); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_jobs SET state='completed',result=$2,completed_at=clock_timestamp() WHERE id=$1", id, map[string]any{"requiresRestart": true, "restoredBackupId": source, "preBackupId": pre}); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "restore.committed", id, reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func rewrapApplication(ctx context.Context, tx pgx.Tx, old, new []byte) error {
	var check []byte
	if e := tx.QueryRow(ctx, "SELECT envelope FROM bootstrap_secrets WHERE name='key_check'").Scan(&check); e != nil {
		return e
	}
	plain, e := cryptobox.Open(old, "filex/bootstrap/v1/key_check", check)
	if e != nil || !bytes.Equal(plain, []byte("filex bootstrap key verification v1")) {
		return ErrBackup
	}
	for _, table := range []string{"key_versions", "bootstrap_secrets"} {
		idCol, boxCol, prefix := "id", "wrapped", "filex/keys/v1/"
		if table == "bootstrap_secrets" {
			idCol, boxCol, prefix = "name", "envelope", "filex/bootstrap/v1/"
		}
		rows, e := tx.Query(ctx, "SELECT "+idCol+"::text,"+boxCol+" FROM "+table)
		if e != nil {
			return e
		}
		type item struct {
			id  string
			box []byte
		}
		all := []item{}
		for rows.Next() {
			var v item
			if e = rows.Scan(&v.id, &v.box); e != nil {
				rows.Close()
				return e
			}
			all = append(all, v)
			if len(all) > 1048570 {
				rows.Close()
				return ErrBackup
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, v := range all {
			key, e := cryptobox.Open(old, prefix+v.id, v.box)
			if e != nil {
				return ErrBackup
			}
			if !bytes.Equal(old, new) {
				box, e := cryptobox.Seal(new, prefix+v.id, key)
				if e != nil {
					return e
				}
				if _, e = tx.Exec(ctx, "UPDATE "+table+" SET "+boxCol+"=$2 WHERE "+idCol+"::text=$1", v.id, box); e != nil {
					return e
				}
			}
		}
	}
	if !bytes.Equal(old, new) {
		_, e = tx.Exec(ctx, "UPDATE crypto_usage SET seal_count=(SELECT count(*)+3 FROM key_versions) WHERE name='kek'")
	}
	return e
}
func (s Service) rememberProviders(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, "SELECT id::text FROM storage_providers WHERE purpose<>'backups'")
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
		d, e := s.Storage.Describe(ctx, id)
		if e != nil {
			return e
		}
		raw, e := json.Marshal(d)
		if e != nil {
			return e
		}
		box, e := s.seal("filex/ops_inventory/v1/"+id, raw)
		if e != nil {
			return e
		}
		if _, e = s.DB.Exec(ctx, "INSERT INTO ops_provider_history(id,envelope) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET envelope=excluded.envelope", id, box); e != nil {
			return e
		}
	}
	return nil
}

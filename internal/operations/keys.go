package operations

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/maintenance"
	"github.com/jackc/pgx/v5"
	"io"
)

type cryptoSnapshot struct {
	InstanceID  string
	Keys, Usage int64
}

func inspectCryptoSnapshot(r io.Reader) (cryptoSnapshot, error) {
	var info cryptoSnapshot
	e := scanSnapshot(r, func(h snapshotHeader) error {
		if !domain.ValidID(h.InstanceID) {
			return ErrBackup
		}
		info.InstanceID = h.InstanceID
		return nil
	}, func(rec record) error {
		if rec.Table == "key_versions" {
			info.Keys++
		}
		if rec.Table == "crypto_usage" {
			var value struct {
				Name  string `json:"name"`
				Count int64  `json:"seal_count"`
			}
			if e := json.Unmarshal(rec.Row, &value); e != nil {
				return e
			}
			if value.Name == "kek" {
				info.Usage = value.Count
			}
		}
		return nil
	})
	if e != nil {
		return info, e
	}
	if info.Keys < 1 || info.Keys > 1048570 || info.Usage < 1 || info.Usage > 1048576 {
		return info, ErrBackup
	}
	return info, nil
}
func (s Service) reserveRestoreWraps(ctx context.Context, info cryptoSnapshot, recoveryKEK []byte) (int64, error) {
	var instance string
	var current int64
	if e := s.DB.QueryRow(ctx, "SELECT instance_id::text FROM ops_control WHERE singleton").Scan(&instance); e != nil {
		return 0, e
	}
	if bytes.Equal(recoveryKEK, s.Keys.KEK) && instance != info.InstanceID {
		return 0, ErrBackup
	}
	if e := s.DB.QueryRow(ctx, "SELECT seal_count FROM crypto_usage WHERE name='kek'").Scan(&current); e != nil {
		return 0, e
	}
	count := current + 4
	if bytes.Equal(recoveryKEK, s.Keys.KEK) {
		count = max(current, info.Usage) + 4
	} else {
		count += info.Keys + 2
	}
	if count > 1048576 {
		return 0, keyring.ErrUsage
	}
	tag, e := s.DB.Exec(ctx, "UPDATE crypto_usage SET seal_count=$2 WHERE name='kek' AND seal_count=$1", current, count)
	if e != nil {
		return 0, e
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrConflict
	}
	return count, nil
}
func freshRestoredKeys(ctx context.Context, tx pgx.Tx, kek []byte) error {
	if _, e := tx.Exec(ctx, "UPDATE key_versions SET active=false,seal_count=1048576 WHERE class<>'signing'"); e != nil {
		return e
	}
	for _, class := range []string{"payload", "artifacts", "configuration", "backups"} {
		var version int64
		if e := tx.QueryRow(ctx, "SELECT COALESCE(max(version),0)+1 FROM key_versions WHERE class=$1", class).Scan(&version); e != nil {
			return e
		}
		id := domain.NewID()
		key := make([]byte, 32)
		if _, e := rand.Read(key); e != nil {
			return e
		}
		box, e := cryptobox.Seal(kek, "filex/keys/v1/"+id, key)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO key_versions(id,class,version,wrapped,active) VALUES($1,$2,$3,$4,true)", id, class, version, box); e != nil {
			return e
		}
	}
	return nil
}

// Rewrap requires a fresh, independently generated KEK on stdin at the CLI.
// Historical backup descriptors retain their original KEK deliberately.
func (s Service) Rewrap(ctx context.Context, newKey []byte, provider, actor, reason string) error {
	if len(newKey) != 32 || bytes.Equal(newKey, s.Keys.KEK) || !validReason(reason) {
		return ErrInvalid
	}
	unlock, e := s.runnerLock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	if e = s.SetMaintenance(ctx, actor, reason); e != nil {
		return e
	}
	release, e := maintenance.Exclusive(ctx, s.DB)
	if e != nil {
		return e
	}
	defer release()
	backup, e := s.Queue(ctx, actor, "backup", "", provider, "Pre-KEK-rewrap backup: "+reason)
	if e != nil {
		return e
	}
	if _, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET state='running',started_at=clock_timestamp() WHERE id=$1", backup); e != nil {
		return e
	}
	if e = s.backup(ctx, backup); e != nil {
		cleanup, cancel := boundedCleanup()
		defer cancel()
		_, mark := s.DB.Exec(cleanup, "UPDATE ops_jobs SET state='failed',error_code='pre_backup_failed',completed_at=clock_timestamp() WHERE id=$1", backup)
		return errors.Join(e, mark)
	}
	if _, e = s.DB.Exec(ctx, "UPDATE ops_jobs SET state='completed',completed_at=clock_timestamp() WHERE id=$1", backup); e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = rewrapApplication(ctx, tx, s.Keys.KEK, newKey); e != nil {
		return e
	}
	var box []byte
	if e = tx.QueryRow(ctx, "SELECT envelope FROM ops_secrets WHERE name='ledger_key'").Scan(&box); e != nil {
		return e
	}
	key, e := cryptobox.Open(s.Keys.KEK, "filex/ops/v1/ledger_key", box)
	if e != nil {
		return e
	}
	box, e = cryptobox.Seal(newKey, "filex/ops/v1/ledger_key", key)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_secrets SET envelope=$1 WHERE name='ledger_key'", box); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode='reconcile',generation=gen_random_uuid(),reconciliation_id=NULL,changed_at=clock_timestamp()"); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "kek.rewrapped", backup, reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) VerifyBackup(ctx context.Context, id string, recoveryKey []byte) (map[string]any, error) {
	if len(recoveryKey) == 0 {
		recoveryKey = s.Keys.KEK
	}
	p, e := s.jobProvider(ctx, id)
	if e != nil {
		return nil, e
	}
	r, e := readArchive(ctx, p, id, recoveryKey)
	if e != nil {
		return nil, e
	}
	if e = s.validateSnapshot(ctx, r); e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "bytes": r.manifest.Bytes, "sha256": r.manifest.SHA, "verified": true}, nil
}
func (s Service) ImportBackup(ctx context.Context, actor, id, provider, reason string, recoveryKey []byte) error {
	if !validReason(reason) || !domain.ValidID(id) || !domain.ValidID(provider) {
		return ErrInvalid
	}
	if len(recoveryKey) == 0 {
		recoveryKey = s.Keys.KEK
	}
	var valid bool
	if e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM storage_providers WHERE id=$1 AND purpose='backups' AND enabled)", provider).Scan(&valid); e != nil {
		return e
	}
	if !valid {
		return ErrInvalid
	}
	d, e := s.Storage.Describe(ctx, provider)
	if e != nil {
		return e
	}
	p, e := d.Open()
	if e != nil {
		return e
	}
	r, e := readArchive(ctx, p, id, recoveryKey)
	if e != nil {
		return e
	}
	if e = s.validateSnapshot(ctx, r); e != nil {
		return e
	}
	raw, e := json.Marshal(d)
	if e != nil {
		return e
	}
	box, e := s.seal("filex/ops_provider/v1/"+id, raw)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "INSERT INTO ops_jobs(id,kind,state,actor,reason,provider_id,provider_envelope,progress,result,completed_at) VALUES($1,'backup','completed',$2,$3,$4,$5,$6,$7,clock_timestamp())", id, actor, reason, provider, box, r.manifest.Bytes, map[string]any{"verified": true, "sha256": r.manifest.SHA, "imported": true}); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "backup.imported", id, reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

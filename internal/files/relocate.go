package files

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"time"
)

// Relocate copies encrypted chunks under a renewable file lease. The final swap
// is atomic; a paused job, hold, download, expiry or stale epoch rejects it.
func (s Service) Relocate(ctx context.Context, owner, fileID, targetProvider, targetKey, jobID string, rotation bool) error {
	f, e := s.Get(ctx, owner, fileID)
	if e != nil {
		return e
	}
	if targetProvider == "" {
		targetProvider = f.ProviderID
	}
	if targetKey == "" {
		targetKey = f.KeyID
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var state string
	var frozen bool
	if e = tx.QueryRow(ctx, "SELECT state,frozen FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&state, &frozen); e != nil {
		return e
	}
	if frozen || (state != "active" && state != "draft") {
		return transfer.ErrConflict
	}
	var epoch int64
	if e = tx.QueryRow(ctx, "UPDATE files SET upload_epoch=upload_epoch+1,write_lease_until=clock_timestamp()+interval '60 seconds' WHERE id=$1 AND state='ready' AND NOT frozen AND provider_id=$2 AND key_id=$3 AND (write_lease_until IS NULL OR write_lease_until<clock_timestamp()) RETURNING upload_epoch", f.ID, f.ProviderID, f.KeyID).Scan(&epoch); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	defer s.clearLease(context.Background(), f.TransferID, f.ID, epoch)
	source, e := s.Storage.Get(ctx, f.ProviderID)
	if e != nil {
		return e
	}
	target, e := s.Storage.Get(ctx, targetProvider)
	if e != nil {
		return e
	}
	oldKey, e := s.Keys.Get(ctx, f.KeyID)
	if e != nil {
		return e
	}
	oldKey, e = subkey(oldKey, f.Salt, f.ID)
	if e != nil {
		return e
	}
	newKey, e := s.Keys.Get(ctx, targetKey)
	if e != nil {
		return e
	}
	salt := f.Salt
	if targetKey != f.KeyID {
		salt = make([]byte, 32)
		if _, e = rand.Read(salt); e != nil {
			return e
		}
	}
	newKey, e = subkey(newKey, salt, f.ID)
	if e != nil {
		return e
	}
	rows, e := s.DB.Query(ctx, "SELECT id::text,object_key,plain_offset,plain_length,chunk_index,plain_sha256 FROM file_chunks WHERE file_id=$1 ORDER BY plain_offset", f.ID)
	if e != nil {
		return e
	}
	type replacement struct{ ID, Old, New string }
	items := []replacement{}
	for rows.Next() {
		var item replacement
		var offset, index int64
		var length int
		var hash string
		if e = rows.Scan(&item.ID, &item.Old, &offset, &length, &index, &hash); e != nil {
			rows.Close()
			return e
		}
		if e = s.renewLease(ctx, f.TransferID, f.ID, epoch, jobID, rotation); e != nil {
			rows.Close()
			return e
		}
		data, e := source.Get(ctx, item.Old)
		if e != nil {
			rows.Close()
			return e
		}
		header := payloadHeader(f, item.ID, offset, length, index)
		plain, e := cryptobox.OpenPayload(oldKey, header, data)
		if e != nil {
			rows.Close()
			return e
		}
		sum := sha256.Sum256(plain)
		if hex.EncodeToString(sum[:]) != hash {
			rows.Close()
			return errors.New("migration integrity failed")
		}
		if targetKey != f.KeyID {
			header.KeyID = targetKey
			header.Salt = salt
			data, e = cryptobox.SealPayload(newKey, header, plain)
			if e != nil {
				rows.Close()
				return e
			}
		}
		item.New = domain.NewID()
		if _, e = s.DB.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id,available_at) VALUES($1,$2,$3,clock_timestamp()+interval '5 minutes')", targetProvider, item.New, f.ID); e != nil {
			rows.Close()
			return e
		}
		if e = target.Put(ctx, item.New, data); e != nil {
			rows.Close()
			return e
		}
		verify, e := target.Get(ctx, item.New)
		if e != nil {
			rows.Close()
			return e
		}
		if sha256.Sum256(verify) != sha256.Sum256(data) {
			rows.Close()
			return errors.New("migration copy verification failed")
		}
		items = append(items, item)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = tx.QueryRow(ctx, "SELECT state,frozen FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&state, &frozen); e != nil {
		return e
	}
	if frozen || (state != "active" && state != "draft") {
		return transfer.ErrConflict
	}
	table := "storage_migrations"
	if rotation {
		table = "rotation_jobs"
	}
	var jobState string
	if e = tx.QueryRow(ctx, "SELECT state FROM "+table+" WHERE id=$1 FOR UPDATE", jobID).Scan(&jobState); e != nil {
		return e
	}
	if jobState != "running" {
		return transfer.ErrConflict
	}
	tag, e := tx.Exec(ctx, "UPDATE files SET provider_id=$2,key_id=$3,salt=$4,write_lease_until=NULL WHERE id=$1 AND state='ready' AND NOT frozen AND upload_epoch=$5 AND write_lease_until>clock_timestamp() AND provider_id=$6 AND key_id=$7", f.ID, targetProvider, targetKey, salt, epoch, f.ProviderID, f.KeyID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrConflict
	}
	for _, item := range items {
		if _, e = tx.Exec(ctx, "UPDATE file_chunks SET object_key=$2 WHERE id=$1 AND object_key=$3", item.ID, item.New, item.Old); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "DELETE FROM deletion_jobs WHERE object_key=$1 AND completed_at IS NULL", item.New); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id,available_at) VALUES($1,$2,$3,clock_timestamp()+interval '5 minutes') ON CONFLICT(object_key) DO NOTHING", f.ProviderID, item.Old, f.ID); e != nil {
			return e
		}
	}
	counter := "moved_files"
	if rotation {
		counter = "processed"
	}
	if _, e = tx.Exec(ctx, "UPDATE "+table+" SET "+counter+"="+counter+"+1 WHERE id=$1", jobID); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) renewLease(ctx context.Context, parent, id string, epoch int64, job string, rotation bool) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var found string
	if e = tx.QueryRow(ctx, "SELECT id::text FROM transfers WHERE id=$1 AND NOT frozen AND state IN ('active','draft') FOR UPDATE", parent).Scan(&found); e != nil {
		return e
	}
	table := "storage_migrations"
	if rotation {
		table = "rotation_jobs"
	}
	var state string
	if e = tx.QueryRow(ctx, "SELECT state FROM "+table+" WHERE id=$1", job).Scan(&state); e != nil {
		return e
	}
	if state != "running" {
		return transfer.ErrConflict
	}
	tag, e := tx.Exec(ctx, "UPDATE files SET write_lease_until=clock_timestamp()+interval '60 seconds' WHERE id=$1 AND upload_epoch=$2 AND write_lease_until>clock_timestamp() AND state='ready' AND NOT frozen", id, epoch)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrConflict
	}
	return tx.Commit(ctx)
}
func (s Service) clearLease(ctx context.Context, parent, id string, epoch int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT id FROM transfers WHERE id=$1 FOR UPDATE", parent); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE files SET write_lease_until=NULL WHERE id=$1 AND upload_epoch=$2", id, epoch); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) Maintain(ctx context.Context) error {
	var job, source, target string
	var rotation bool
	e := s.DB.QueryRow(ctx, "SELECT id::text,source,target,rotation FROM (SELECT id,source_id::text source,target_id::text target,false rotation,checked_at,created_at FROM storage_migrations WHERE state='running' UNION ALL SELECT id,'' source,target_key_id::text target,true rotation,checked_at,created_at FROM rotation_jobs WHERE state='running' AND class='payload') jobs ORDER BY checked_at,created_at LIMIT 1").Scan(&job, &source, &target, &rotation)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	table := "storage_migrations"
	if rotation {
		table = "rotation_jobs"
	}
	if _, e = s.DB.Exec(ctx, "UPDATE "+table+" SET checked_at=clock_timestamp() WHERE id=$1", job); e != nil {
		return e
	}
	predicate := "f.provider_id=$1"
	filter := source
	if rotation {
		predicate = "f.key_id<>$1"
		filter = target
	}
	var id, owner string
	e = s.DB.QueryRow(ctx, "SELECT f.id::text,t.owner_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE "+predicate+" AND f.state='ready' AND NOT f.frozen AND NOT t.frozen AND t.state IN ('draft','active') AND (f.write_lease_until IS NULL OR f.write_lease_until<clock_timestamp()) ORDER BY f.id LIMIT 1", filter).Scan(&id, &owner)
	if errors.Is(e, pgx.ErrNoRows) {
		var remaining bool
		if e = s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM files f WHERE "+predicate+" AND f.state NOT IN ('deleted','cancelled','deleting'))", filter).Scan(&remaining); e != nil {
			return e
		}
		if !remaining {
			table := "storage_migrations"
			if rotation {
				table = "rotation_jobs"
			}
			_, e = s.DB.Exec(ctx, "UPDATE "+table+" SET state='completed' WHERE id=$1 AND state='running'", job)
		}
		return e
	}
	if e != nil {
		return e
	}
	if rotation {
		return s.Relocate(ctx, owner, id, "", target, job, true)
	}
	return s.Relocate(ctx, owner, id, target, "", job, false)
}

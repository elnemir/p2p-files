package artifacts

import (
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"fmt"
	"github.com/jackc/pgx/v5"
)

func (s Store) RotateOne(ctx context.Context, targetKey, job string) (bool, error) {
	var id string
	var epoch int64
	e := s.DB.QueryRow(ctx, `WITH next AS(SELECT a.id FROM artifacts a WHERE a.key_id<>$1 AND a.state='ready' AND a.expires_at>clock_timestamp() AND (a.maintenance_until IS NULL OR a.maintenance_until<clock_timestamp()) AND NOT EXISTS(SELECT 1 FROM artifact_credentials c WHERE c.artifact_id=a.id AND c.state IN ('new','leased') AND c.expires_at>clock_timestamp()) ORDER BY a.created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE artifacts a SET maintenance_epoch=maintenance_epoch+1,maintenance_until=clock_timestamp()+interval '3 minutes' FROM next WHERE a.id=next.id RETURNING a.id::text,a.maintenance_epoch`, targetKey).Scan(&id, &epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	defer s.DB.Exec(context.Background(), "UPDATE artifacts SET maintenance_until=NULL WHERE id=$1 AND maintenance_epoch=$2", id, epoch)
	a, e := s.Get(ctx, id)
	if e != nil {
		return false, e
	}
	old, e := s.Keys.Get(ctx, a.KeyID)
	if e != nil {
		return false, e
	}
	old, e = hkdf.Key(sha256.New, old, a.Salt, "filex/artifact/object/v1/"+id, 32)
	if e != nil {
		return false, e
	}
	key, e := s.Keys.Get(ctx, targetKey)
	if e != nil {
		return false, e
	}
	salt := make([]byte, 32)
	if _, e = rand.Read(salt); e != nil {
		return false, e
	}
	key, e = hkdf.Key(sha256.New, key, salt, "filex/artifact/object/v1/"+id, 32)
	if e != nil {
		return false, e
	}
	p, e := s.Storage.Get(ctx, a.Provider)
	if e != nil {
		return false, e
	}
	type item struct {
		Offset   int64
		Length   int
		Old, New string
	}
	items := []item{}
	rows, e := s.DB.Query(ctx, "SELECT offset_bytes,length_bytes,object_key FROM artifact_chunks WHERE artifact_id=$1 ORDER BY offset_bytes", id)
	if e != nil {
		return false, e
	}
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.Offset, &v.Length, &v.Old); e != nil {
			rows.Close()
			return false, e
		}
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return false, e
	}
	for i := range items {
		v := &items[i]
		b, e := p.Get(ctx, v.Old)
		if e != nil {
			return false, e
		}
		aad := "filex/artifact/v1/" + id + "_" + fmt.Sprint(v.Offset)
		plain, e := cryptobox.Open(old, aad, b)
		if e != nil || len(plain) != v.Length {
			return false, errors.New("artifact integrity failed")
		}
		box, e := cryptobox.Seal(key, aad, plain)
		if e != nil {
			return false, e
		}
		v.New = domain.NewID()
		if _, e = s.DB.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,available_at) VALUES($1,$2,clock_timestamp()+interval '5 minutes')", a.Provider, v.New); e != nil {
			return false, e
		}
		if e = p.Put(ctx, v.New, box); e != nil {
			return false, e
		}
		check, e := p.Get(ctx, v.New)
		if e != nil || sha256.Sum256(check) != sha256.Sum256(box) {
			return false, errors.New("artifact re-encryption verification failed")
		}
		tag, e := s.DB.Exec(ctx, "UPDATE artifacts SET maintenance_until=clock_timestamp()+interval '3 minutes' WHERE id=$1 AND maintenance_epoch=$2 AND state='ready' AND maintenance_until>clock_timestamp()", id, epoch)
		if e != nil {
			return false, e
		}
		if tag.RowsAffected() != 1 {
			return false, errors.New("artifact maintenance fence lost")
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(context.Background())
	var jobState string
	if e = tx.QueryRow(ctx, "SELECT state FROM rotation_jobs WHERE id=$1 FOR SHARE", job).Scan(&jobState); e != nil {
		return false, e
	}
	if jobState != "running" {
		return false, errors.New("rotation paused or cancelled")
	}
	var locked string
	if e = tx.QueryRow(ctx, "SELECT id::text FROM artifacts WHERE id=$1 AND state='ready' AND key_id=$2 AND maintenance_epoch=$3 AND maintenance_until>clock_timestamp() FOR UPDATE", id, a.KeyID, epoch).Scan(&locked); e != nil {
		return false, e
	}
	var active bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM artifact_credentials WHERE artifact_id=$1 AND state IN ('new','leased') AND expires_at>clock_timestamp())", id).Scan(&active); e != nil {
		return false, e
	}
	if active {
		return false, nil
	}
	for _, v := range items {
		if _, e = tx.Exec(ctx, "UPDATE artifact_chunks SET object_key=$3 WHERE artifact_id=$1 AND offset_bytes=$2", id, v.Offset, v.New); e != nil {
			return false, e
		}
		if _, e = tx.Exec(ctx, "DELETE FROM deletion_jobs WHERE object_key=$1", v.New); e != nil {
			return false, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,available_at) VALUES($1,$2,clock_timestamp()+interval '5 minutes') ON CONFLICT(object_key) DO NOTHING", a.Provider, v.Old); e != nil {
			return false, e
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE artifacts SET key_id=$2,salt=$3,maintenance_until=NULL WHERE id=$1", id, targetKey, salt); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

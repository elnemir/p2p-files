package artifacts

import (
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/storage"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"time"
)

const Chunk = 4 << 20
const MaxSize = 64 << 20

type Store struct {
	DB      *pgxpool.Pool
	Keys    keyring.Ring
	Storage storage.Registry
}
type Artifact struct {
	ID              string `json:"id"`
	Filename        string `json:"filename"`
	ContentType     string `json:"contentType"`
	Size            int64  `json:"size"`
	SHA256          string `json:"sha256"`
	Provider, KeyID string
	Salt            []byte
}

func (s Store) Create(ctx context.Context, owner, filename, media string, write func(io.Writer) error) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var provider string
	if e := s.DB.QueryRow(ctx, "SELECT id::text FROM storage_providers WHERE purpose='artifacts' AND enabled AND NOT draining").Scan(&provider); e != nil {
		return "", e
	}
	kid, key, e := s.Keys.Active(ctx, "artifacts")
	if e != nil {
		return "", e
	}
	id := domain.NewID()
	salt := make([]byte, 32)
	if _, e = rand.Read(salt); e != nil {
		return "", e
	}
	key, e = hkdf.Key(sha256.New, key, salt, "filex/artifact/object/v1/"+id, 32)
	if e != nil {
		return "", e
	}
	p, e := s.Storage.Get(ctx, provider)
	if e != nil {
		return "", e
	}
	if _, e = s.DB.Exec(ctx, "INSERT INTO artifacts(id,owner_id,provider_id,key_id,salt,filename,content_type) VALUES($1,$2,$3,$4,$5,$6,$7)", id, owner, provider, kid, salt, filename, media); e != nil {
		return "", e
	}
	writer := &chunkWriter{ctx: ctx, store: s, p: p, id: id, provider: provider, key: key}
	digest := sha256.New()
	if e = write(io.MultiWriter(writer, digest)); e == nil {
		e = writer.flush()
	}
	if e != nil {
		return "", e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	if e = keyring.RequireActive(ctx, tx, kid); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "UPDATE artifacts SET state='ready',size=$2,sha256=$3 WHERE id=$1 AND state='writing'", id, writer.offset, hex.EncodeToString(digest.Sum(nil))); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM deletion_jobs WHERE object_key IN (SELECT object_key FROM artifact_chunks WHERE artifact_id=$1)", id); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}

type chunkWriter struct {
	ctx          context.Context
	store        Store
	p            storage.Provider
	id, provider string
	key, buffer  []byte
	offset       int64
}

func (w *chunkWriter) Write(b []byte) (int, error) {
	n := len(b)
	if w.offset+int64(len(w.buffer))+int64(n) > MaxSize {
		return 0, errors.New("artifact limit exceeded")
	}
	for len(b) > 0 {
		take := min(Chunk-len(w.buffer), len(b))
		w.buffer = append(w.buffer, b[:take]...)
		b = b[take:]
		if len(w.buffer) == Chunk {
			if e := w.flush(); e != nil {
				return n - len(b), e
			}
		}
	}
	return n, nil
}
func (w *chunkWriter) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	if e := w.ctx.Err(); e != nil {
		return e
	}
	object := domain.NewID()
	box, e := cryptobox.Seal(w.key, "filex/artifact/v1/"+w.id+"_"+fmt.Sprint(w.offset), w.buffer)
	if e != nil {
		return e
	}
	if _, e = w.store.DB.Exec(w.ctx, "INSERT INTO deletion_jobs(provider_id,object_key,available_at) VALUES($1,$2,clock_timestamp()+interval '5 minutes')", w.provider, object); e != nil {
		return e
	}
	if e = w.p.Put(w.ctx, object, box); e != nil {
		return e
	}
	stored, e := w.p.Get(w.ctx, object)
	if e != nil {
		return e
	}
	if sha256.Sum256(stored) != sha256.Sum256(box) {
		return errors.New("artifact write verification failed")
	}
	if _, e = w.store.DB.Exec(w.ctx, "INSERT INTO artifact_chunks(artifact_id,offset_bytes,length_bytes,object_key) VALUES($1,$2,$3,$4)", w.id, w.offset, len(w.buffer), object); e != nil {
		return e
	}
	w.offset += int64(len(w.buffer))
	w.buffer = w.buffer[:0]
	return nil
}
func (s Store) Get(ctx context.Context, id string) (Artifact, error) {
	var a Artifact
	e := s.DB.QueryRow(ctx, "SELECT id::text,filename,content_type,size,COALESCE(sha256,''),provider_id::text,key_id::text,salt FROM artifacts WHERE id::text=$1 AND state='ready' AND expires_at>clock_timestamp()", id).Scan(&a.ID, &a.Filename, &a.ContentType, &a.Size, &a.SHA256, &a.Provider, &a.KeyID, &a.Salt)
	return a, e
}
func (s Store) Read(ctx context.Context, a Artifact, start, end int64, w io.Writer) error {
	if start < 0 || end < start || end > a.Size {
		return domain.ErrInvalid
	}
	key, e := s.Keys.Get(ctx, a.KeyID)
	if e != nil {
		return e
	}
	key, e = hkdf.Key(sha256.New, key, a.Salt, "filex/artifact/object/v1/"+a.ID, 32)
	if e != nil {
		return e
	}
	p, e := s.Storage.Get(ctx, a.Provider)
	if e != nil {
		return e
	}
	rows, e := s.DB.Query(ctx, "SELECT offset_bytes,length_bytes,object_key FROM artifact_chunks WHERE artifact_id=$1 AND offset_bytes<$3 AND offset_bytes+length_bytes>$2 ORDER BY offset_bytes", a.ID, start, end)
	if e != nil {
		return e
	}
	defer rows.Close()
	type part struct {
		offset int64
		length int
		object string
	}
	parts := []part{}
	for rows.Next() {
		var offset int64
		var length int
		var object string
		if e = rows.Scan(&offset, &length, &object); e != nil {
			return e
		}
		parts = append(parts, part{offset, length, object})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	cursor := start
	for _, part := range parts {
		offset, length, object := part.offset, part.length, part.object
		b, e := p.Get(ctx, object)
		if e != nil {
			return e
		}
		plain, e := cryptobox.Open(key, "filex/artifact/v1/"+a.ID+"_"+fmt.Sprint(offset), b)
		if e != nil || len(plain) != length {
			return errors.New("artifact integrity failed")
		}
		from, to := max(start, offset), min(end, offset+int64(length))
		if from != cursor {
			return errors.New("artifact gap")
		}
		n, e := w.Write(plain[from-offset : to-offset])
		if e != nil {
			return e
		}
		if int64(n) != to-from {
			return io.ErrShortWrite
		}
		cursor = to
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if cursor != end {
		return errors.New("artifact incomplete")
	}
	return nil
}
func (s Store) Cleanup(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "UPDATE artifacts SET state='deleting' WHERE state IN ('writing','ready') AND (expires_at<=clock_timestamp() OR (state='writing' AND created_at<clock_timestamp()-interval '5 minutes'))"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key) SELECT a.provider_id,ch.object_key FROM artifacts a JOIN artifact_chunks ch ON ch.artifact_id=a.id WHERE a.state='deleting' ON CONFLICT(object_key) DO NOTHING"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE artifacts SET state='deleted' WHERE state='deleting' AND NOT EXISTS(SELECT 1 FROM artifact_chunks ch LEFT JOIN deletion_jobs j ON j.object_key=ch.object_key WHERE ch.artifact_id=artifacts.id AND j.completed_at IS NULL)"); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

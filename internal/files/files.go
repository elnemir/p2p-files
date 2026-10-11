package files

import (
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/storage"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"hash/crc32"
	"io"
	"net/http"
	"time"
)

const ChunkSize = 4 * 1024 * 1024

var ErrOffset = errors.New("offset conflict")

type Service struct {
	DB      *pgxpool.Pool
	Keys    keyring.Ring
	Storage storage.Registry
	Scanner Scanner
}
type File struct {
	ID         string  `json:"id"`
	TransferID string  `json:"transferId"`
	Path       string  `json:"path"`
	Size       int64   `json:"size"`
	Offset     int64   `json:"offset"`
	State      string  `json:"state"`
	SHA256     *string `json:"sha256"`
	CRC32      *int64  `json:"-"`
	ProviderID string  `json:"-"`
	KeyID      string  `json:"-"`
	Salt       []byte  `json:"-"`
}

func (s Service) Create(ctx context.Context, owner, id, path string, size int64, c config.Settings) (transfer.Upload, error) {
	if size < 0 || size > 1<<40 {
		return transfer.Upload{}, domain.ErrInvalid
	}
	provider, _, e := s.Storage.Select(ctx)
	if e != nil {
		return transfer.Upload{}, e
	}
	key, _, e := s.Keys.Active(ctx, "payload")
	if e != nil {
		return transfer.Upload{}, e
	}
	salt := make([]byte, 32)
	if _, e = rand.Read(salt); e != nil {
		return transfer.Upload{}, e
	}
	return transfer.New(s.DB, c).ReserveUploadWith(ctx, owner, id, path, size, func(tx pgx.Tx, fileID string) error {
		pathKey, e := domain.PathKey(path, c.MaxPathBytes)
		if e != nil {
			return e
		}
		var conflict bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM draft_folders WHERE transfer_id=$1 AND (path_key=$2 OR starts_with(path_key,$2||'/')))", id, pathKey).Scan(&conflict); e != nil {
			return e
		}
		if conflict {
			return transfer.ErrConflict
		}
		if e := keyring.RequireActive(ctx, tx, key); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, "UPDATE files SET provider_id=$2,key_id=$3,salt=$4 WHERE id=$1", fileID, provider, key, salt)
		return e
	})
}
func (s Service) Get(ctx context.Context, owner, id string) (File, error) {
	var f File
	e := s.DB.QueryRow(ctx, `SELECT f.id::text,f.transfer_id::text,f.path,f.size,f.upload_offset,f.state,f.sha256,f.crc32,COALESCE(f.provider_id::text,''),COALESCE(f.key_id::text,''),f.salt FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id::text=$1 AND t.owner_id::text=$2`, id, owner).Scan(&f.ID, &f.TransferID, &f.Path, &f.Size, &f.Offset, &f.State, &f.SHA256, &f.CRC32, &f.ProviderID, &f.KeyID, &f.Salt)
	return f, e
}
func subkey(root, salt []byte, id string) ([]byte, error) {
	return hkdf.Key(sha256.New, root, salt, "filex/payload/object/v1/"+id, 32)
}
func (s Service) Append(ctx context.Context, owner, id string, offset int64, data []byte) (int64, error) {
	if len(data) > ChunkSize {
		return 0, domain.ErrInvalid
	}
	f, e := s.Get(ctx, owner, id)
	if e != nil {
		return 0, e
	}
	if f.State != "uploading" || f.Offset != offset || int64(len(data)) > f.Size-offset {
		return 0, ErrOffset
	}
	if len(data) == 0 {
		return offset, s.mutate(ctx, f.TransferID, func(tx pgx.Tx) error {
			var state string
			return tx.QueryRow(ctx, "SELECT state FROM files WHERE id=$1 AND state='uploading' AND upload_offset=$2", id, offset).Scan(&state)
		})
	}
	key, e := s.Keys.Get(ctx, f.KeyID)
	if e != nil {
		return 0, e
	}
	key, e = subkey(key, f.Salt, f.ID)
	if e != nil {
		return 0, e
	}
	p, e := s.Storage.Get(ctx, f.ProviderID)
	if e != nil {
		return 0, e
	}
	var epoch int64
	e = s.mutate(ctx, f.TransferID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `UPDATE files SET upload_epoch=upload_epoch+1,write_lease_until=clock_timestamp()+interval '60 seconds' WHERE id=$1 AND upload_offset=$2 AND upload_epoch<1048576 AND state='uploading' AND (write_lease_until IS NULL OR write_lease_until<clock_timestamp()) RETURNING upload_epoch`, id, offset).Scan(&epoch)
	})
	if errors.Is(e, pgx.ErrNoRows) {
		return 0, ErrOffset
	}
	if e != nil {
		return 0, e
	}
	defer s.clearLease(context.WithoutCancel(ctx), f.TransferID, id, epoch)
	object := domain.NewID()
	chunkID := domain.NewID()
	cipher, e := cryptobox.SealPayload(key, payloadHeader(f, chunkID, offset, len(data), epoch), data)
	if e != nil {
		return 0, e
	}
	// Register compensation before the write: a crash cannot leave an untracked object.
	if _, e = s.DB.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id,available_at) VALUES($1,$2,$3,clock_timestamp()+interval '5 minutes')", f.ProviderID, object, id); e != nil {
		return 0, e
	}
	if e = p.Put(ctx, object, cipher); e != nil {
		return 0, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback(context.Background())
	var state string
	if e = tx.QueryRow(ctx, "SELECT state FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&state); e != nil {
		return 0, e
	}
	if state != "draft" {
		return 0, transfer.ErrConflict
	}
	tag, e := tx.Exec(ctx, "UPDATE files SET upload_offset=upload_offset+$3,write_lease_until=NULL WHERE id=$1 AND upload_epoch=$2 AND write_lease_until>clock_timestamp() AND upload_offset=$4 AND state='uploading'", id, epoch, len(data), offset)
	if e != nil {
		return 0, e
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrOffset
	}
	hash := sha256.Sum256(data)
	_, e = tx.Exec(ctx, "INSERT INTO file_chunks(id,file_id,chunk_index,plain_offset,plain_length,object_key,plain_sha256) VALUES($1,$2,$3,$4,$5,$6,$7)", chunkID, id, epoch, offset, len(data), object, hex.EncodeToString(hash[:]))
	if e != nil {
		return 0, e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM deletion_jobs WHERE object_key=$1 AND completed_at IS NULL AND lease_until IS NULL", object); e != nil {
		return 0, e
	}
	if e = tx.Commit(ctx); e != nil {
		return 0, e
	}
	return offset + int64(len(data)), nil
}
func (s Service) Read(ctx context.Context, f File, start, end int64, w io.Writer) error {
	if start < 0 || end < start || end > f.Size {
		return domain.ErrInvalid
	}
	key, e := s.Keys.Get(ctx, f.KeyID)
	if e != nil {
		return e
	}
	key, e = subkey(key, f.Salt, f.ID)
	if e != nil {
		return e
	}
	p, e := s.Storage.Get(ctx, f.ProviderID)
	if e != nil {
		return e
	}
	cursor := start
	for cursor < end {
		rows, e := s.DB.Query(ctx, "SELECT id::text,chunk_index,plain_offset,plain_length,object_key,plain_sha256 FROM file_chunks WHERE file_id=$1 AND plain_offset<$3 AND plain_offset+plain_length>$2 ORDER BY plain_offset LIMIT 64", f.ID, cursor, end)
		if e != nil {
			return e
		}
		type chunk struct {
			id               string
			index            int64
			offset           int64
			length           int
			object, checksum string
		}
		chunks := []chunk{}
		for rows.Next() {
			var ch chunk
			if e = rows.Scan(&ch.id, &ch.index, &ch.offset, &ch.length, &ch.object, &ch.checksum); e != nil {
				rows.Close()
				return e
			}
			chunks = append(chunks, ch)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(chunks) == 0 {
			break
		}
		for _, ch := range chunks {
			offset, length, object, checksum := ch.offset, ch.length, ch.object, ch.checksum
			b, e := p.Get(ctx, object)
			if e != nil {
				if errors.Is(e, storage.ErrMissing) {
					if mark := s.MarkFailure(ctx, f, "storage_missing"); mark != nil {
						return mark
					}
				}
				return e
			}
			plain, e := cryptobox.OpenPayload(key, payloadHeader(f, ch.id, offset, length, ch.index), b)
			if e != nil {
				if mark := s.MarkFailure(ctx, f, "integrity_failed"); mark != nil {
					return mark
				}
				return e
			}
			hash := sha256.Sum256(plain)
			if len(plain) != length || hex.EncodeToString(hash[:]) != checksum {
				if mark := s.MarkFailure(ctx, f, "integrity_failed"); mark != nil {
					return mark
				}
				return errors.New("payload integrity failed")
			}
			from, to := max(cursor, offset), min(end, offset+int64(length))
			if from != cursor {
				return errors.New("payload gap")
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
	}
	if cursor != end {
		return errors.New("payload truncated")
	}
	return nil
}
func (s Service) Finalize(ctx context.Context, owner, id string, c config.Settings) error {
	f, e := s.Get(ctx, owner, id)
	if e != nil {
		return e
	}
	if f.State == "ready" {
		return nil
	}
	if f.Offset != f.Size {
		return ErrOffset
	}
	store := transfer.New(s.DB, c)
	if _, e = store.BeginFinalize(ctx, owner, f.TransferID, id); e != nil {
		return e
	}
	sha := sha256.New()
	crc := crc32.NewIEEE()
	sniff := &prefix{limit: 512}
	if e = s.Read(ctx, f, 0, f.Size, io.MultiWriter(sha, crc, sniff)); e != nil {
		return e
	}
	media := http.DetectContentType(sniff.b)
	if !permitted(c.FileTypeMode, c.FileTypes, media) {
		e = s.mutate(ctx, f.TransferID, func(tx pgx.Tx) error {
			_, e := tx.Exec(ctx, "UPDATE files SET state='rejected' WHERE id=$1 AND state='finalizing'", id)
			return e
		})
		if e != nil {
			return e
		}
		return domain.ErrInvalid
	}
	if s.Scanner != nil {
		if e = s.Scanner.Scan(ctx, media, func(w io.Writer) error { return s.Read(ctx, f, 0, f.Size, w) }); e != nil {
			return e
		}
	}
	sum := hex.EncodeToString(sha.Sum(nil))
	if e = s.mutate(ctx, f.TransferID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, "UPDATE files SET crc32=$2,content_type=$3 WHERE id=$1 AND state='finalizing'", id, int64(crc.Sum32()), media)
		return e
	}); e != nil {
		return e
	}
	_, e = store.Finalize(ctx, owner, f.TransferID, id, sum)
	return e
}

type prefix struct {
	b     []byte
	limit int
}

func (p *prefix) Write(b []byte) (int, error) {
	n := len(b)
	take := min(n, p.limit-len(p.b))
	p.b = append(p.b, b[:take]...)
	return n, nil
}

// Every file mutation acquires its parent first, matching publication and hold order.
func (s Service) mutate(ctx context.Context, parent string, fn func(pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var state string
	var frozen bool
	if e = tx.QueryRow(ctx, "SELECT state,frozen FROM transfers WHERE id=$1 FOR UPDATE", parent).Scan(&state, &frozen); e != nil {
		return e
	}
	if state != "draft" || frozen {
		return transfer.ErrConflict
	}
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func payloadHeader(f File, id string, offset int64, length int, index int64) cryptobox.PayloadHeader {
	return cryptobox.PayloadHeader{ObjectID: id, FileID: f.ID, KeyID: f.KeyID, Salt: f.Salt, Size: uint64(f.Size), Offset: uint64(offset), Length: uint32(length), Index: uint64(index)}
}

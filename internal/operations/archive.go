package operations

import (
	"context"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/storage"
	"fmt"
	stdhash "hash"
	"io"
)

const archiveChunk = 4 << 20
const maxArchive = 1 << 30

type archiveDescriptor struct {
	Version    int    `json:"version"`
	ID         string `json:"id"`
	KeyID      string `json:"keyId"`
	Salt       []byte `json:"salt"`
	WrappedKey []byte `json:"wrappedKey"`
	Manifest   []byte `json:"manifest"`
}
type archivePart struct {
	Key   string `json:"key"`
	Bytes int    `json:"bytes"`
	SHA   string `json:"sha256"`
}
type archiveManifest struct {
	Bytes int64         `json:"bytes"`
	SHA   string        `json:"sha256"`
	Parts []archivePart `json:"parts"`
}
type archiveWriter struct {
	ctx      context.Context
	p        storage.Provider
	id       string
	key      []byte
	buffer   []byte
	sum      stdhash.Hash
	manifest archiveManifest
}

func (w *archiveWriter) Write(b []byte) (int, error) {
	n := len(b)
	if w.manifest.Bytes+int64(n) > maxArchive {
		return 0, ErrBackup
	}
	w.sum.Write(b)
	w.manifest.Bytes += int64(n)
	for len(b) > 0 {
		take := min(archiveChunk-len(w.buffer), len(b))
		w.buffer = append(w.buffer, b[:take]...)
		b = b[take:]
		if len(w.buffer) == archiveChunk {
			if e := w.flush(); e != nil {
				return 0, e
			}
		}
	}
	return n, nil
}
func (w *archiveWriter) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	index := len(w.manifest.Parts)
	object := fmt.Sprintf("backup-%s-%04d", w.id, index)
	box, e := cryptobox.Seal(w.key, fmt.Sprintf("filex/backups/v1/%s_chunk_%d", w.id, index), w.buffer)
	if e != nil {
		return e
	}
	if e = w.p.Put(w.ctx, object, box); e != nil {
		return e
	}
	w.manifest.Parts = append(w.manifest.Parts, archivePart{Key: object, Bytes: len(w.buffer), SHA: hash(box)})
	w.buffer = w.buffer[:0]
	return nil
}
func (s Service) writeArchive(ctx context.Context, id string, p storage.Provider, write func(io.Writer) error) error {
	keyID, root, e := s.Keys.Active(ctx, "backups")
	if e != nil {
		return e
	}
	var wrapped []byte
	if e = s.DB.QueryRow(ctx, "SELECT wrapped FROM key_versions WHERE id=$1", keyID).Scan(&wrapped); e != nil {
		return e
	}
	return writeArchive(ctx, id, p, keyID, root, wrapped, s.Keys.KEK, write)
}
func writeArchive(ctx context.Context, id string, p storage.Provider, keyID string, root, wrapped, kek []byte, write func(io.Writer) error) error {
	salt := make([]byte, 32)
	if _, e := rand.Read(salt); e != nil {
		return e
	}
	key, e := hkdf.Key(sha256.New, root, salt, "filex/backups/v1/"+id, 32)
	if e != nil {
		return e
	}
	w := &archiveWriter{ctx: ctx, p: p, id: id, key: key, sum: sha256.New(), manifest: archiveManifest{Parts: []archivePart{}}}
	if e = write(w); e != nil {
		return e
	}
	if e = w.flush(); e != nil {
		return e
	}
	w.manifest.SHA = fmt.Sprintf("%x", w.sum.Sum(nil))
	raw, e := json.Marshal(w.manifest)
	if e != nil {
		return e
	}
	box, e := cryptobox.Seal(key, "filex/backups/v1/"+id+"_manifest", raw)
	if e != nil {
		return e
	}
	desc, e := json.Marshal(archiveDescriptor{Version: 1, ID: id, KeyID: keyID, Salt: salt, WrappedKey: wrapped, Manifest: box})
	if e != nil {
		return e
	}
	// The descriptor is the completion marker; partially written chunks cannot be imported.
	if e = p.Put(ctx, "backup-"+id, desc); e != nil {
		return e
	}
	r, e := readArchive(ctx, p, id, kek)
	if e != nil {
		return e
	}
	_, e = io.Copy(io.Discard, r)
	return e
}

type archiveReader struct {
	ctx      context.Context
	p        storage.Provider
	id       string
	key      []byte
	manifest archiveManifest
	part     int
	buffer   []byte
	sum      stdhash.Hash
	size     int64
	verified bool
}

func readArchive(ctx context.Context, p storage.Provider, id string, kek []byte) (*archiveReader, error) {
	if !domain.ValidID(id) {
		return nil, ErrBackup
	}
	raw, e := p.Get(ctx, "backup-"+id)
	if e != nil {
		return nil, e
	}
	var d archiveDescriptor
	if e = json.Unmarshal(raw, &d); e != nil || d.Version != 1 || d.ID != id || !domain.ValidID(d.KeyID) || len(d.Salt) != 32 {
		return nil, ErrBackup
	}
	root, e := cryptobox.Open(kek, "filex/keys/v1/"+d.KeyID, d.WrappedKey)
	if e != nil {
		return nil, ErrBackup
	}
	key, e := hkdf.Key(sha256.New, root, d.Salt, "filex/backups/v1/"+id, 32)
	if e != nil {
		return nil, e
	}
	plain, e := cryptobox.Open(key, "filex/backups/v1/"+id+"_manifest", d.Manifest)
	if e != nil {
		return nil, ErrBackup
	}
	var m archiveManifest
	if e = json.Unmarshal(plain, &m); e != nil || m.Bytes < 1 || m.Bytes > maxArchive || len(m.Parts) < 1 || len(m.Parts) > maxArchive/archiveChunk || len(m.SHA) != 64 {
		return nil, ErrBackup
	}
	var size int64
	for i, p := range m.Parts {
		if p.Key != fmt.Sprintf("backup-%s-%04d", id, i) || p.Bytes < 1 || p.Bytes > archiveChunk || len(p.SHA) != 64 || (i < len(m.Parts)-1 && p.Bytes != archiveChunk) {
			return nil, ErrBackup
		}
		size += int64(p.Bytes)
	}
	if size != m.Bytes {
		return nil, ErrBackup
	}
	return &archiveReader{ctx: ctx, p: p, id: id, key: key, manifest: m, sum: sha256.New()}, nil
}
func (r *archiveReader) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if len(r.buffer) == 0 {
		if r.part == len(r.manifest.Parts) {
			if !r.verified {
				if r.size != r.manifest.Bytes || fmt.Sprintf("%x", r.sum.Sum(nil)) != r.manifest.SHA {
					return 0, ErrBackup
				}
				r.verified = true
			}
			return 0, io.EOF
		}
		part := r.manifest.Parts[r.part]
		box, e := r.p.Get(r.ctx, part.Key)
		if e != nil {
			return 0, e
		}
		if hash(box) != part.SHA {
			return 0, ErrBackup
		}
		plain, e := cryptobox.Open(r.key, fmt.Sprintf("filex/backups/v1/%s_chunk_%d", r.id, r.part), box)
		if e != nil || len(plain) != part.Bytes {
			return 0, ErrBackup
		}
		r.buffer = plain
		r.sum.Write(plain)
		r.size += int64(len(plain))
		r.part++
	}
	n := copy(b, r.buffer)
	r.buffer = r.buffer[n:]
	return n, nil
}

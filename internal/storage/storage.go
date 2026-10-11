package storage

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/keyring"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var ErrMissing = errors.New("storage object missing")
var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type Provider interface {
	Put(context.Context, string, []byte) error
	Get(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}
type Local struct{ Root string }

func (l Local) root() (*os.Root, error) {
	if !filepath.IsAbs(l.Root) {
		return nil, errors.New("absolute managed storage root required")
	}
	return os.OpenRoot(l.Root)
}
func (l Local) Put(ctx context.Context, key string, b []byte) error {
	if !keyPattern.MatchString(key) || len(b) > 4194816 {
		return errors.New("invalid object")
	}
	r, e := l.root()
	if e != nil {
		return e
	}
	defer r.Close()
	f, e := r.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	return f.Close()
}
func (l Local) Get(ctx context.Context, key string) ([]byte, error) {
	if !keyPattern.MatchString(key) {
		return nil, errors.New("invalid object key")
	}
	r, e := l.root()
	if e != nil {
		return nil, e
	}
	defer r.Close()
	f, e := r.Open(key)
	if errors.Is(e, os.ErrNotExist) {
		return nil, ErrMissing
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 4194817))
	if len(b) > 4194816 {
		return nil, errors.New("object too large")
	}
	return b, e
}
func (l Local) Delete(ctx context.Context, key string) error {
	if !keyPattern.MatchString(key) {
		return errors.New("invalid object key")
	}
	r, e := l.root()
	if e != nil {
		return e
	}
	defer r.Close()
	e = r.Remove(key)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	return e
}

type Registry struct {
	DB   *pgxpool.Pool
	Keys keyring.Ring
}

func (r Registry) Get(ctx context.Context, id string) (Provider, error) {
	var kind string
	var raw, box []byte
	var keyID *string
	if e := r.DB.QueryRow(ctx, "SELECT kind,config,secret_key_id::text,secret_envelope FROM storage_providers WHERE id=$1", id).Scan(&kind, &raw, &keyID, &box); e != nil {
		return nil, e
	}
	if kind == "local" {
		var c struct {
			Root string `json:"root"`
		}
		if e := json.Unmarshal(raw, &c); e != nil {
			return nil, e
		}
		return Local{c.Root}, nil
	}
	if kind == "s3" && keyID != nil {
		key, e := r.Keys.Get(ctx, *keyID)
		if e != nil {
			return nil, e
		}
		plain, e := cryptobox.Open(key, "filex/storage/v1/"+id, box)
		if e != nil {
			return nil, e
		}
		var c S3Config
		if e = json.Unmarshal(plain, &c); e != nil {
			return nil, e
		}
		return NewS3(c)
	}
	return nil, errors.New("provider unavailable")
}
func (r Registry) Select(ctx context.Context) (string, Provider, error) {
	rows, e := r.DB.Query(ctx, "SELECT id::text FROM storage_providers WHERE enabled AND NOT draining AND purpose='payload' ORDER BY priority DESC,id")
	if e != nil {
		return "", nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return "", nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return "", nil, e
	}
	for _, id := range ids {
		p, e := r.Get(ctx, id)
		if e != nil {
			continue
		}
		if l, ok := p.(Local); ok {
			root, e := l.root()
			if e != nil {
				continue
			}
			root.Close()
		}
		if remote, ok := p.(S3); ok {
			if e := remote.Check(ctx); e != nil {
				continue
			}
		}
		return id, p, nil
	}
	if e = rows.Err(); e != nil {
		return "", nil, e
	}
	return "", nil, errors.New("no available storage provider")
}

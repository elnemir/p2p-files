package httpapi

import (
	"context"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/bundle"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/transfer"
	"io"
	"net/http"
	"strconv"
)

// A transfer hold provides a stable retained set without consuming recipient rights.
func (a *App) forensicZIP(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var input struct {
		Reason string `json:"reason"`
	}
	if e := body(w, r, &input); e != nil {
		return e
	}
	if len(input.Reason) < 5 || len(input.Reason) > 2000 {
		return auth.ErrInvalid
	}
	ctx := r.Context()
	tx, e := a.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var owner string
	var held bool
	if e = tx.QueryRow(ctx, "SELECT owner_id::text,frozen FROM transfers WHERE id::text=$1 FOR UPDATE", r.PathValue("id")).Scan(&owner, &held); e != nil {
		return e
	}
	if !held {
		return transfer.ErrConflict
	}
	rows, e := tx.Query(ctx, "SELECT id::text,path,size,sha256,crc32 FROM files WHERE transfer_id::text=$1 AND state NOT IN ('deleted','cancelled') ORDER BY path", r.PathValue("id"))
	if e != nil {
		return e
	}
	entries := []bundle.Entry{}
	for rows.Next() {
		var ent bundle.Entry
		var hash *string
		var crc *int64
		if e = rows.Scan(&ent.FileID, &ent.Path, &ent.Size, &hash, &crc); e != nil {
			rows.Close()
			return e
		}
		if hash == nil || crc == nil {
			rows.Close()
			return transfer.ErrConflict
		}
		ent.SHA256 = *hash
		ent.CRC32 = uint32(*crc)
		entries = append(entries, ent)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(entries) == 0 {
		return transfer.ErrConflict
	}
	if e = adminEvent(ctx, tx, a.Key, p.ID, "forensic.zip", r.PathValue("id"), input.Reason); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	kid, seed, e := a.Keys.Signing(ctx)
	if e != nil {
		return e
	}
	z, e := bundle.New(entries, seed, kid)
	if e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="forensic.zip"`)
	w.Header().Set("Content-Length", strconv.FormatInt(z.Size(), 10))
	writer := forensicZIPWriter{adminWriter: adminWriter{a, r, p, w}, ctx: ctx, parent: r.PathValue("id")}
	if e = writer.held(); e != nil {
		w.Header().Del("Content-Length")
		return e
	}
	w.WriteHeader(200)
	e = z.WriteRange(0, z.Size(), writer, func(ent bundle.Entry, from, to int64, out io.Writer) error {
		f, e := a.Files.Get(ctx, owner, ent.FileID)
		if e != nil {
			return e
		}
		return a.Files.Read(ctx, f, from, to, out)
	})
	if e != nil {
		panic(http.ErrAbortHandler)
	}
	return nil
}

type forensicZIPWriter struct {
	adminWriter
	ctx    context.Context
	parent string
}

func (w forensicZIPWriter) held() error {
	var frozen bool
	if e := w.a.DB.QueryRow(w.ctx, "SELECT frozen FROM transfers WHERE id=$1", w.parent).Scan(&frozen); e != nil {
		return e
	}
	if !frozen {
		return transfer.ErrConflict
	}
	return nil
}
func (w forensicZIPWriter) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		if e := w.held(); e != nil {
			return total, e
		}
		chunk := b[:min(len(b), 32768)]
		n, e := w.adminWriter.Write(chunk)
		total += n
		if e != nil {
			return total, e
		}
		b = b[n:]
		if e = w.held(); e != nil {
			return total, e
		}
	}
	return total, nil
}

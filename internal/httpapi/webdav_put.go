package httpapi

import (
	"context"
	"errors"
	"filex.local/filex/internal/licensing"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
)

func (d davAccess) put(w http.ResponseWriter, r *http.Request, name string) (result error) {
	if name == "" || strings.HasSuffix(r.URL.Path, "/") {
		return davError(405)
	}
	if r.Header.Get("Content-Range") != "" {
		return davError(400)
	}
	if v := r.Header.Get("Content-Encoding"); v != "" && v != "identity" {
		return davError(415)
	}
	size := r.ContentLength
	if hint := r.Header.Get("X-Expected-Entity-Length"); hint != "" {
		n, e := strconv.ParseInt(hint, 10, 64)
		if e != nil || n < 0 || (size >= 0 && size != n) {
			return davError(400)
		}
		size = n
	}
	if size > 1<<40 {
		return davError(413)
	}
	original, e := d.lookup(r.Context(), d.app.DB, name)
	exists := e == nil
	if e != nil && !errors.Is(e, davError(404)) {
		return e
	}
	if exists && (original.Directory || original.State != "ready") {
		return davError(409)
	}
	staged := []string{}
	committed := ""
	key, e := domain.PathKey(name, d.settings.MaxPathBytes)
	if e != nil {
		return e
	}
	guard := func(ctx context.Context, tx pgx.Tx) error {
		if _, e := licensing.Guard(ctx, tx, "webdav"); e != nil {
			return e
		}
		if e := d.guard(ctx, tx); e != nil {
			return e
		}
		if e := d.conditions(ctx, tx, r, name); e != nil {
			return e
		}
		if e := transfer.CheckDAVLocks(ctx, tx, d.transfer, key, false); e != nil {
			return e
		}
		for _, id := range staged {
			tag, e := tx.Exec(ctx, "UPDATE files SET staged_until=clock_timestamp()+interval '5 minutes' WHERE id=$1 AND staged_until>clock_timestamp() AND state NOT IN ('cancelled','deleted')", id)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return transfer.ErrConflict
			}
		}
		return nil
	}
	service := d.app.Files
	service.WriteGuard = guard
	service.ReadGuard = func(ctx context.Context) error {
		tx, e := d.app.DB.Begin(ctx)
		if e != nil {
			return e
		}
		defer tx.Rollback(ctx)
		if e = d.lock(ctx, tx); e != nil {
			return e
		}
		if e = guard(ctx, tx); e != nil {
			return e
		}
		return tx.Commit(ctx)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()
		for _, id := range staged {
			if id != committed {
				if e := d.app.Files.AbortStage(ctx, d.owner, id); e != nil {
					result = errors.Join(result, e)
				}
			}
		}
	}()
	reserve := func(length int64) (string, error) {
		reserveCtx := r.Context()
		if size < 0 && len(staged) == 0 {
			reserveCtx = licensing.WithStreaming(reserveCtx)
		}
		u, e := service.CreateStaged(reserveCtx, d.owner, d.transfer, name, length, d.settings, func(tx pgx.Tx, _ string) error {
			if e := d.parentExists(r.Context(), tx, name); e != nil {
				return e
			}
			nodes, e := d.nodes(r.Context(), tx)
			if e != nil {
				return e
			}
			if !exists && len(nodes) >= davMaxNodes {
				return davError(507)
			}
			return nil
		})
		if e != nil {
			return "", e
		}
		staged = append(staged, u.ID)
		return u.ID, nil
	}
	declared := size
	if declared < 0 {
		declared = 1 << 40
	}
	id, e := reserve(declared)
	if e != nil {
		return e
	}
	controller := http.NewResponseController(w)
	buf := make([]byte, files.ChunkSize)
	offset := int64(0)
	for {
		if e = controller.SetReadDeadline(time.Now().Add(30 * time.Second)); e != nil && !errors.Is(e, http.ErrNotSupported) {
			return e
		}
		want := len(buf)
		if size >= 0 {
			want = int(min(int64(want), size-offset))
			if want == 0 {
				break
			}
		}
		n, readErr := io.ReadFull(r.Body, buf[:want])
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return davError(400)
		}
		if n > 0 {
			if offset+int64(n) > 1<<40 {
				return davError(413)
			}
			offset, e = service.Append(r.Context(), d.owner, id, offset, buf[:n])
			if e != nil {
				return e
			}
		}
		if readErr != nil {
			if size >= 0 && offset != size {
				return davError(400)
			}
			break
		}
	}
	if size >= 0 {
		var extra [1]byte
		if n, e := r.Body.Read(extra[:]); n != 0 || e != io.EOF {
			return davError(400)
		}
	}
	if size < 0 {
		provisional, e := service.Get(r.Context(), d.owner, id)
		if e != nil {
			return e
		}
		finalID, e := reserve(offset)
		if e != nil {
			return e
		}
		writer := &davAppendWriter{service: service, ctx: r.Context(), owner: d.owner, id: finalID}
		if e = service.Read(r.Context(), provisional, 0, offset, writer); e != nil {
			return e
		}
		id = finalID
	}
	if e = service.Finalize(r.Context(), d.owner, id, d.settings); e != nil {
		return e
	}
	tx, e := d.app.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	if e = d.lock(r.Context(), tx); e != nil {
		return e
	}
	if e = guard(r.Context(), tx); e != nil {
		return e
	}
	current, e := d.lookup(r.Context(), tx, name)
	if exists {
		if e != nil || current.ID != original.ID || current.etag() != original.etag() {
			return davError(412)
		}
	} else if e == nil {
		return davError(412)
	} else if !errors.Is(e, davError(404)) {
		return e
	}
	if !exists {
		if e = transfer.CheckDAVMembership(r.Context(), tx, d.transfer, key); e != nil {
			return e
		}
	}
	if e = d.parentExists(r.Context(), tx, name); e != nil {
		return e
	}
	// A new file may have become an ancestor while this hidden candidate uploaded.
	var conflict bool
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted') AND (starts_with($2,path_key||'/') OR starts_with(path_key,$2||'/'))) OR EXISTS(SELECT 1 FROM draft_folders WHERE transfer_id=$1 AND (path_key=$2 OR starts_with(path_key,$2||'/')))`, d.transfer, key).Scan(&conflict); e != nil {
		return e
	}
	if conflict {
		return davError(409)
	}
	if exists {
		if e = d.cancelInTx(r.Context(), tx, original.ID); e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "UPDATE files SET created_at=o.created_at,dav_properties=o.dav_properties FROM files o WHERE files.id=$1 AND o.id=$2", id, original.ID); e != nil {
			return e
		}
	}
	var count int
	if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM files WHERE transfer_id=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted')", d.transfer).Scan(&count); e != nil {
		return e
	}
	if count >= d.settings.MaxFiles {
		return davError(507)
	}
	if _, e = tx.Exec(r.Context(), "UPDATE files SET path_key=$2,staged_until=NULL,modified_at=clock_timestamp() WHERE id=$1 AND state='ready'", id, key); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "UPDATE transfers SET revision=revision+1 WHERE id=$1", d.transfer); e != nil {
		return e
	}
	if _, e = d.nodes(r.Context(), tx); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	committed = id
	f, e := service.Get(r.Context(), d.owner, id)
	if e != nil {
		return e
	}
	w.Header().Set("ETag", davFileETag(f.ID, f.State, f.Offset, f.SHA256))
	w.Header().Set("Location", davHref(d.transfer, name, false))
	status := 201
	if exists {
		status = 204
	}
	w.WriteHeader(status)
	return nil
}

type davAppendWriter struct {
	service   files.Service
	ctx       context.Context
	owner, id string
	offset    int64
}

func (w *davAppendWriter) Write(b []byte) (int, error) {
	off, e := w.service.Append(w.ctx, w.owner, w.id, w.offset, b)
	if e != nil {
		return 0, e
	}
	w.offset = off
	return len(b), nil
}

func (d davAccess) cancelInTx(ctx context.Context, tx pgx.Tx, id string) error {
	tag, e := tx.Exec(ctx, "UPDATE files SET state='cancelled',upload_epoch=upload_epoch+1,path_key='#cancelled:'||id::text WHERE id=$1 AND transfer_id=$2 AND NOT frozen", id, d.transfer)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return davError(403)
	}
	_, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.id=$1 ON CONFLICT(object_key) DO NOTHING", id)
	return e
}

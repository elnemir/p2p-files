package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

func (d davAccess) read(w http.ResponseWriter, r *http.Request, name string) error {
	n, e := d.lookup(r.Context(), d.app.DB, name)
	if e != nil {
		return e
	}
	if n.Directory {
		if r.Method != "HEAD" {
			return davError(405)
		}
		w.Header().Set("Content-Type", n.contentType())
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(200)
		return nil
	}
	if n.State != "ready" {
		return davError(409)
	}
	f, e := d.app.Files.Get(r.Context(), d.owner, n.ID)
	if e != nil {
		return e
	}
	rangeValue := r.Header.Get("Range")
	if v := r.Header.Get("If-Range"); v != "" && v != n.etag() {
		date, err := http.ParseTime(v)
		if err != nil || n.Modified.Unix() > date.Unix() {
			rangeValue = ""
		}
	}
	start, end, status, e := byteRange(rangeValue, n.Size)
	if e != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", n.Size))
		return davError(416)
	}
	w.Header().Set("Content-Type", n.contentType())
	w.Header().Set("Content-Length", strconv.FormatInt(end-start, 10))
	w.Header().Set("ETag", n.etag())
	w.Header().Set("Last-Modified", n.Modified.UTC().Format(http.TimeFormat))
	w.Header().Set("Accept-Ranges", "bytes")
	if status == 206 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, n.Size))
	}
	writer := &davReadWriter{d: d, w: w, r: r, id: n.ID, status: status}
	if r.Method == "HEAD" || n.Size == 0 {
		if e = writer.check(nil); e != nil {
			return e
		}
		return nil
	}
	if e = d.app.Files.Read(r.Context(), f, start, end, writer); e != nil {
		if writer.started {
			panic(http.ErrAbortHandler)
		}
		return e
	}
	return nil
}

type davReadWriter struct {
	d       davAccess
	w       http.ResponseWriter
	r       *http.Request
	id      string
	status  int
	started bool
}

func (w *davReadWriter) check(b []byte) error {
	ctx, cancel := context.WithTimeout(w.r.Context(), 5*time.Second)
	defer cancel()
	if e := w.d.check(ctx); e != nil {
		return e
	}
	tx, e := w.d.app.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var allowed bool
	if e = tx.QueryRow(ctx, "SELECT state='draft' AND NOT frozen FROM transfers WHERE id=$1 FOR SHARE", w.d.transfer).Scan(&allowed); e != nil {
		return e
	}
	if !allowed {
		return davError(403)
	}
	if e = w.d.guard(ctx, tx); e != nil {
		return e
	}
	if e = tx.QueryRow(ctx, "SELECT state='ready' AND NOT frozen AND staged_until IS NULL FROM files WHERE id=$1 FOR SHARE", w.id).Scan(&allowed); e != nil {
		return e
	}
	if !allowed {
		return davError(403)
	}
	if e = http.NewResponseController(w.w).SetWriteDeadline(time.Now().Add(5 * time.Second)); e != nil && !errors.Is(e, http.ErrNotSupported) {
		return e
	}
	if !w.started {
		w.w.WriteHeader(w.status)
		w.started = true
	}
	if len(b) > 0 {
		n, e := w.w.Write(b)
		if e != nil {
			return e
		}
		if n != len(b) {
			return io.ErrShortWrite
		}
	}
	return tx.Commit(ctx)
}
func (w *davReadWriter) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		n := min(len(b), 64<<10)
		if e := w.check(b[:n]); e != nil {
			return total, e
		}
		total += n
		b = b[n:]
	}
	return total, nil
}

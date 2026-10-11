package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/bundle"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/transfer"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type recipient struct {
	ID, TransferID, OwnerID string
	Epoch                   int64
}

func (a *App) recipient(r *http.Request) (recipient, error) {
	cookie, e := r.Cookie("filexRecipient")
	if e != nil || len(cookie.Value) != 43 {
		return recipient{}, auth.ErrCredentials
	}
	var p recipient
	e = a.DB.QueryRow(r.Context(), `SELECT s.id::text,t.id::text,t.owner_id::text,t.access_epoch FROM recipient_sessions s JOIN recipient_links l ON l.id=s.link_id JOIN transfers t ON t.id=l.transfer_id JOIN users u ON u.id=t.owner_id WHERE s.token_hash=$1 AND s.expires_at>clock_timestamp() AND s.link_epoch=l.epoch AND l.revoked_at IS NULL AND t.state='active' AND NOT t.frozen AND NOT u.disabled AND t.expires_at>clock_timestamp() AND t.absolute_expires_at>clock_timestamp()`, auth.Digest(cookie.Value)).Scan(&p.ID, &p.TransferID, &p.OwnerID, &p.Epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return p, auth.ErrCredentials
	}
	return p, e
}
func (a *App) recipientRoutes(m *http.ServeMux) {
	wrap := func(fn func(http.ResponseWriter, *http.Request, recipient) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p, e := a.recipient(r)
			if e != nil {
				failure(w, e)
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" {
				c, e := database.Settings(r.Context(), a.DB)
				if e != nil {
					failure(w, e)
					return
				}
				if r.Header.Get("Origin") != c.PublicOrigin {
					failure(w, auth.ErrForbidden)
					return
				}
			}
			if e = fn(w, r, p); e != nil {
				failure(w, e)
			}
		}
	}
	m.HandleFunc("POST /api/v1/recipient/session", a.recipientLogin)
	m.HandleFunc("GET /api/v1/recipient/files", wrap(a.recipientFiles))
	m.HandleFunc("POST /api/v1/recipient/downloads", wrap(a.acquireDownload))
	m.HandleFunc("GET /api/v1/recipient/downloads/{id}/content", wrap(a.downloadContent))
	m.HandleFunc("HEAD /api/v1/recipient/downloads/{id}/content", wrap(a.downloadHead))
}
func (a *App) recipientLogin(w http.ResponseWriter, r *http.Request) {
	c, e := database.Settings(r.Context(), a.DB)
	if e != nil {
		failure(w, e)
		return
	}
	if r.Header.Get("Origin") != c.PublicOrigin {
		failure(w, auth.ErrForbidden)
		return
	}
	var v struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if e = body(w, r, &v); e != nil {
		failure(w, e)
		return
	}
	if len(v.Token) != 43 {
		failure(w, auth.ErrCredentials)
		return
	}
	// Capability-specific attempts are durable and do not expose recipient correlation to USER.
	var attempts int
	e = a.DB.QueryRow(r.Context(), `INSERT INTO auth_limits(key,window_start,count) VALUES($1,clock_timestamp(),1) ON CONFLICT(key) DO UPDATE SET count=CASE WHEN auth_limits.window_start<clock_timestamp()-interval '1 minute' THEN 1 ELSE auth_limits.count+1 END,window_start=CASE WHEN auth_limits.window_start<clock_timestamp()-interval '1 minute' THEN clock_timestamp() ELSE auth_limits.window_start END RETURNING count`, auth.Digest("recipient:"+v.Token)).Scan(&attempts)
	if e != nil {
		failure(w, e)
		return
	}
	if attempts > 10 {
		failure(w, auth.ErrRate)
		return
	}
	var link string
	var epoch int64
	var hash *string
	var expires time.Time
	e = a.DB.QueryRow(r.Context(), `SELECT l.id::text,l.epoch,l.password_hash,LEAST(t.expires_at,clock_timestamp()+interval '12 hours') FROM recipient_links l JOIN transfers t ON t.id=l.transfer_id JOIN users u ON u.id=t.owner_id WHERE l.token_hash=$1 AND l.revoked_at IS NULL AND t.state='active' AND NOT t.frozen AND NOT u.disabled AND t.expires_at>clock_timestamp() AND t.absolute_expires_at>clock_timestamp()`, auth.Digest(v.Token)).Scan(&link, &epoch, &hash, &expires)
	if e != nil || (hash != nil && !auth.Verify(*hash, v.Password)) {
		failure(w, auth.ErrCredentials)
		return
	}
	token := auth.Token()
	_, e = a.DB.Exec(r.Context(), "INSERT INTO recipient_sessions(id,link_id,token_hash,link_epoch,expires_at) VALUES($1,$2,$3,$4,$5)", domain.NewID(), link, auth.Digest(token), epoch, expires)
	if e != nil {
		failure(w, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "filexRecipient", Value: token, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteStrictMode, Expires: expires})
	_ = reply(w, 200, map[string]string{"status": "ready"})
}
func (a *App) recipientFiles(w http.ResponseWriter, r *http.Request, p recipient) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,path,size,state FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted','deleting') ORDER BY path", p.TransferID)
	if e != nil {
		return e
	}
	type item struct {
		ID    string
		Ref   string `json:"ref"`
		Path  string `json:"path"`
		Size  int64  `json:"size"`
		State string `json:"state"`
	}
	items := []item{}
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.ID, &v.Path, &v.Size, &v.State); e != nil {
			rows.Close()
			return e
		}
		h := hmac.New(sha256.New, a.Key)
		h.Write([]byte("filex/ref/v1/" + p.ID + "/" + v.ID))
		v.Ref = hex.EncodeToString(h.Sum(nil))
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	out := []map[string]any{}
	for _, v := range items {
		if _, e = a.DB.Exec(r.Context(), "INSERT INTO recipient_file_refs(session_id,file_id,ref_hash) VALUES($1,$2,$3) ON CONFLICT(session_id,file_id) DO UPDATE SET ref_hash=excluded.ref_hash", p.ID, v.ID, auth.Digest(v.Ref)); e != nil {
			return e
		}
		out = append(out, map[string]any{"ref": v.Ref, "path": v.Path, "size": v.Size, "state": v.State})
	}
	return reply(w, 200, out)
}
func (a *App) acquireDownload(w http.ResponseWriter, r *http.Request, p recipient) error {
	var v struct {
		Ref string `json:"fileRef"`
		All bool   `json:"all"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.All == (v.Ref != "") {
		return auth.ErrInvalid
	}
	var signingID string
	var seed []byte
	var e error
	if v.All {
		signingID, seed, e = a.Keys.Signing(r.Context())
		if e != nil {
			return e
		}
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	var epoch int64
	if e = tx.QueryRow(r.Context(), "SELECT access_epoch FROM transfers WHERE id=$1 AND state='active' AND NOT frozen AND expires_at>clock_timestamp() AND absolute_expires_at>clock_timestamp() FOR UPDATE", p.TransferID).Scan(&epoch); e != nil {
		return e
	}
	if e = lockRecipient(r.Context(), tx, p); e != nil {
		return e
	}
	// Expired leases are recovered as a set while holding the parent lock.
	if _, e = tx.Exec(r.Context(), `UPDATE files SET state='ready' WHERE transfer_id=$1 AND state='leased' AND NOT EXISTS(SELECT 1 FROM downloads d JOIN download_files df ON df.download_id=d.id WHERE df.file_id=files.id AND d.state='leased' AND d.lease_until>clock_timestamp())`, p.TransferID); e != nil {
		return e
	}
	rows, e := tx.Query(r.Context(), `SELECT f.id::text,f.path,f.size,f.crc32,f.sha256,f.state,f.frozen FROM files f WHERE f.transfer_id=$1 AND (($2 AND f.state NOT IN ('cancelled','deleting','deleted')) OR (NOT $2 AND EXISTS(SELECT 1 FROM recipient_file_refs refs WHERE refs.session_id=$3 AND refs.ref_hash=$4 AND refs.file_id=f.id))) ORDER BY f.path FOR UPDATE`, p.TransferID, v.All, p.ID, auth.Digest(v.Ref))
	if e != nil {
		return e
	}
	entries := []bundle.Entry{}
	var size int64
	for rows.Next() {
		var ent bundle.Entry
		var crc *int64
		var sha *string
		var state string
		var frozen bool
		if e = rows.Scan(&ent.FileID, &ent.Path, &ent.Size, &crc, &sha, &state, &frozen); e != nil {
			rows.Close()
			return e
		}
		if state != "ready" || frozen || crc == nil || sha == nil {
			rows.Close()
			return transfer.ErrConflict
		}
		ent.CRC32 = uint32(*crc)
		ent.SHA256 = *sha
		entries = append(entries, ent)
		size += ent.Size
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(entries) == 0 {
		return transfer.ErrNotFound
	}
	kind := "file"
	var bundleRaw []byte
	if v.All {
		kind = "zip"
		b, e := bundle.New(entries, seed, signingID)
		if e != nil {
			return e
		}
		bundleRaw, e = json.Marshal(b)
		if e != nil {
			return e
		}
		size = b.Size()
	}
	download := domain.NewID()
	if _, e = tx.Exec(r.Context(), "INSERT INTO downloads(id,recipient_session_id,transfer_id,transfer_epoch,kind,lease_until,total_size,bundle) VALUES($1,$2,$3,$4,$5,clock_timestamp()+interval '5 minutes',$6,$7)", download, p.ID, p.TransferID, epoch, kind, size, bundleRaw); e != nil {
		return e
	}
	for _, ent := range entries {
		if _, e = tx.Exec(r.Context(), "INSERT INTO download_files(download_id,file_id) VALUES($1,$2)", download, ent.FileID); e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "UPDATE files SET state='leased' WHERE id=$1", ent.FileID); e != nil {
			return e
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 201, map[string]string{"id": download, "url": "/api/v1/recipient/downloads/" + download + "/content"})
}
func byteRange(value string, size int64) (int64, int64, int, error) {
	if value == "" {
		return 0, size, 200, nil
	}
	if !strings.HasPrefix(value, "bytes=") || strings.Contains(value, ",") || size == 0 {
		return 0, 0, 0, auth.ErrInvalid
	}
	pair := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(pair) != 2 {
		return 0, 0, 0, auth.ErrInvalid
	}
	if pair[0] == "" {
		n, e := strconv.ParseInt(pair[1], 10, 64)
		if e != nil || n <= 0 {
			return 0, 0, 0, auth.ErrInvalid
		}
		return max(int64(0), size-n), size, 206, nil
	}
	start, e := strconv.ParseInt(pair[0], 10, 64)
	if e != nil || start < 0 || start >= size {
		return 0, 0, 0, auth.ErrInvalid
	}
	end := size
	if pair[1] != "" {
		n, e := strconv.ParseInt(pair[1], 10, 64)
		if e != nil || n < start {
			return 0, 0, 0, auth.ErrInvalid
		}
		end = min(size, n+1)
		if n == int64(^uint64(0)>>1) {
			end = size
		}
	}
	return start, end, 206, nil
}

type downloadData struct {
	Size   int64
	Kind   string
	Bundle bundle.Bundle
	File   files.File
}

func (a *App) downloadData(r *http.Request, p recipient) (downloadData, error) {
	var d downloadData
	var raw []byte
	e := a.DB.QueryRow(r.Context(), "SELECT kind,total_size,bundle FROM downloads WHERE id::text=$1 AND recipient_session_id=$2 AND state='leased' AND lease_until>clock_timestamp()", r.PathValue("id"), p.ID).Scan(&d.Kind, &d.Size, &raw)
	if e != nil {
		return d, e
	}
	if d.Kind == "zip" {
		e = json.Unmarshal(raw, &d.Bundle)
	} else {
		var id string
		e = a.DB.QueryRow(r.Context(), "SELECT file_id::text FROM download_files WHERE download_id::text=$1", r.PathValue("id")).Scan(&id)
		if e == nil {
			d.File, e = a.Files.Get(r.Context(), p.OwnerID, id)
		}
	}
	return d, e
}
func (a *App) downloadHead(w http.ResponseWriter, r *http.Request, p recipient) error {
	d, e := a.downloadData(r, p)
	if e != nil {
		return e
	}
	w.Header().Set("Content-Length", strconv.FormatInt(d.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(200)
	return nil
}

type guardedWriter struct {
	a     *App
	r     *http.Request
	p     recipient
	id    string
	fence int64
	w     http.ResponseWriter
}

func (g guardedWriter) guard() error {
	ctx, cancel := context.WithTimeout(g.r.Context(), 2*time.Second)
	defer cancel()
	var ok bool
	e := g.a.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM downloads d JOIN transfers t ON t.id=d.transfer_id JOIN users u ON u.id=t.owner_id JOIN recipient_sessions s ON s.id=d.recipient_session_id JOIN recipient_links l ON l.id=s.link_id WHERE d.id::text=$1 AND d.fence=$2 AND d.recipient_session_id=$3 AND d.state='leased' AND d.lease_until>clock_timestamp() AND d.writer_until>clock_timestamp() AND t.state='active' AND NOT t.frozen AND NOT u.disabled AND t.access_epoch=d.transfer_epoch AND t.expires_at>clock_timestamp() AND t.absolute_expires_at>clock_timestamp() AND s.expires_at>clock_timestamp() AND l.epoch=s.link_epoch AND l.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM download_files df JOIN files f ON f.id=df.file_id WHERE df.download_id=d.id AND (f.frozen OR f.state<>'leased')))`, g.id, g.fence, g.p.ID).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return transfer.ErrConflict
	}
	return nil
}
func (g guardedWriter) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		if e := g.guard(); e != nil {
			return total, e
		}
		chunk := b[:min(32768, len(b))]
		_ = http.NewResponseController(g.w).SetWriteDeadline(time.Now().Add(2 * time.Second))
		n, e := g.w.Write(chunk)
		total += n
		if e != nil {
			return total, e
		}
		if n != len(chunk) {
			return total, io.ErrShortWrite
		}
		b = b[n:]
		if e = g.guard(); e != nil {
			return total, e
		}
		tag, e := g.a.DB.Exec(g.r.Context(), "UPDATE downloads SET writer_until=clock_timestamp()+interval '10 seconds',lease_until=clock_timestamp()+interval '5 minutes' WHERE id::text=$1 AND fence=$2 AND state='leased'", g.id, g.fence)
		if e != nil {
			return total, e
		}
		if tag.RowsAffected() != 1 {
			return total, transfer.ErrConflict
		}
	}
	return total, nil
}
func (a *App) downloadContent(w http.ResponseWriter, r *http.Request, p recipient) error {
	d, e := a.downloadData(r, p)
	if e != nil {
		return e
	}
	start, end, status, e := byteRange(r.Header.Get("Range"), d.Size)
	if e != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", d.Size))
		w.WriteHeader(416)
		return nil
	}
	id := r.PathValue("id")
	var fence int64
	e = a.DB.QueryRow(r.Context(), "UPDATE downloads SET fence=fence+1,writer_until=clock_timestamp()+interval '10 seconds' WHERE id::text=$1 AND recipient_session_id=$2 AND state='leased' AND lease_until>clock_timestamp() AND (writer_until IS NULL OR writer_until<clock_timestamp()) RETURNING fence", id, p.ID).Scan(&fence)
	if errors.Is(e, pgx.ErrNoRows) {
		return transfer.ErrConflict
	}
	if e != nil {
		return e
	}
	defer a.DB.Exec(context.Background(), "UPDATE downloads SET writer_until=NULL WHERE id::text=$1 AND fence=$2", id, fence)
	writer := guardedWriter{a, r, p, id, fence, w}
	if e = writer.guard(); e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	name := path.Base(d.File.Path)
	if d.Kind == "zip" {
		name = "FileX.zip"
		w.Header().Set("Content-Type", "application/zip")
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(end-start, 10))
	if status == 206 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, d.Size))
	}
	w.WriteHeader(status)
	if d.Kind == "zip" {
		e = d.Bundle.WriteRange(start, end, writer, func(ent bundle.Entry, from, to int64, w io.Writer) error {
			f, e := a.Files.Get(r.Context(), p.OwnerID, ent.FileID)
			if e != nil {
				return e
			}
			return a.Files.Read(r.Context(), f, from, to, w)
		})
	} else {
		e = a.Files.Read(r.Context(), d.File, start, end, writer)
	}
	if e != nil {
		panic(http.ErrAbortHandler)
	}
	if e = writer.guard(); e != nil {
		panic(http.ErrAbortHandler)
	}
	if e = a.finishDownload(r.Context(), p, id, fence, start, end); e != nil {
		panic(http.ErrAbortHandler)
	}
	return nil
}
func (a *App) finishDownload(ctx context.Context, p recipient, id string, fence, start, end int64) error {
	tx, e := a.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var epoch int64
	if e = tx.QueryRow(ctx, "SELECT access_epoch FROM transfers WHERE id=$1 AND state='active' AND NOT frozen AND expires_at>clock_timestamp() AND absolute_expires_at>clock_timestamp() FOR UPDATE", p.TransferID).Scan(&epoch); e != nil {
		return e
	}
	if e = lockRecipient(ctx, tx, p); e != nil {
		return e
	}
	var size int64
	var raw []byte
	if e = tx.QueryRow(ctx, "SELECT total_size,coverage FROM downloads WHERE id::text=$1 AND recipient_session_id=$2 AND fence=$3 AND transfer_epoch=$4 AND state='leased' AND lease_until>clock_timestamp() AND writer_until>clock_timestamp() FOR UPDATE", id, p.ID, fence, epoch).Scan(&size, &raw); e != nil {
		return e
	}
	var intervals [][2]int64
	if e = json.Unmarshal(raw, &intervals); e != nil {
		return e
	}
	intervals = append(intervals, [2]int64{start, end})
	intervals = mergeIntervals(intervals)
	raw, e = json.Marshal(intervals)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE downloads SET coverage=$2 WHERE id::text=$1", id, raw); e != nil {
		return e
	}
	if len(intervals) == 1 && intervals[0][0] == 0 && intervals[0][1] == size {
		var count int64
		if e = tx.QueryRow(ctx, "SELECT count(*) FROM download_files df JOIN files f ON f.id=df.file_id WHERE df.download_id::text=$1 AND (f.frozen OR f.state<>'leased')", id).Scan(&count); e != nil {
			return e
		}
		if count > 0 {
			return transfer.ErrConflict
		}
		tag, e := tx.Exec(ctx, "UPDATE files SET state='deleting' WHERE id IN (SELECT file_id FROM download_files WHERE download_id::text=$1)", id)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE downloads SET state='completed',completed_at=clock_timestamp() WHERE id::text=$1", id); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id JOIN download_files df ON df.file_id=f.id WHERE df.download_id::text=$1 ON CONFLICT(object_key) DO NOTHING", id); e != nil {
			return e
		}
		var payloadBytes int64
		if e = tx.QueryRow(ctx, "SELECT COALESCE(sum(f.size),0) FROM files f JOIN download_files df ON df.file_id=f.id WHERE df.download_id::text=$1", id).Scan(&payloadBytes); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO download_journal(transfer_id,download_id,file_count,total_bytes,status) VALUES($1,$2,$3,$4,'completed')", p.TransferID, id, tag.RowsAffected(), payloadBytes); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) VALUES($1,'download.completed',$2,jsonb_build_object('downloadId',$3::text)) ON CONFLICT(deduplication_key) DO NOTHING`, p.TransferID, "download:"+id, id); e != nil {
			return e
		}

		if _, e = tx.Exec(ctx, "UPDATE transfers SET state='completed' WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND state NOT IN ('deleted','deleting','cancelled'))", p.TransferID); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func mergeIntervals(v [][2]int64) [][2]int64 {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j][0] < v[j-1][0]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
	out := [][2]int64{}
	for _, x := range v {
		if len(out) == 0 || x[0] > out[len(out)-1][1] {
			out = append(out, x)
		} else {
			out[len(out)-1][1] = max(out[len(out)-1][1], x[1])
		}
	}
	return out
}

// Every access mutation takes the transfer lock before these shared identity locks.
func lockRecipient(ctx context.Context, tx pgx.Tx, p recipient) error {
	var id string
	return tx.QueryRow(ctx, `SELECT s.id::text FROM recipient_sessions s JOIN recipient_links l ON l.id=s.link_id JOIN transfers t ON t.id=l.transfer_id JOIN users u ON u.id=t.owner_id WHERE s.id=$1 AND t.id=$2 AND s.expires_at>clock_timestamp() AND s.link_epoch=l.epoch AND l.revoked_at IS NULL AND NOT u.disabled FOR SHARE OF s,l,u`, p.ID, p.TransferID).Scan(&id)
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) artifactRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/artifacts/{id}/credentials", a.protect("reports:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if p.Bearer {
			return auth.ErrForbidden
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(r.Context())
		var id string
		e = tx.QueryRow(r.Context(), "SELECT id::text FROM artifacts WHERE id::text=$1 AND owner_id=$2 AND state='ready' AND expires_at>clock_timestamp() AND NOT EXISTS(SELECT 1 FROM report_artifacts ra JOIN reports rep ON rep.id=ra.report_id WHERE ra.artifact_id=artifacts.id AND (rep.state<>'active' OR (rep.kind='admin' AND ($3<>'admin' OR NOT $4)))) AND NOT EXISTS(SELECT 1 FROM export_jobs ej WHERE ej.artifact_id=artifacts.id AND ej.kind='admin' AND ($3<>'admin' OR NOT $4)) FOR UPDATE", r.PathValue("id"), p.ID, p.Role, time.Since(p.Reauth) <= 5*time.Minute).Scan(&id)
		if e != nil {
			return e
		}
		token := auth.Token()
		_, e = tx.Exec(r.Context(), "INSERT INTO artifact_credentials(id,artifact_id,user_session_id,token_hash,expires_at) VALUES($1,$2,$3,$4,clock_timestamp()+interval '5 minutes')", domain.NewID(), id, p.SessionID, auth.Digest(token))
		if e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"token": token})
	}))
	m.HandleFunc("POST /api/v1/artifacts/{id}/claim", a.protect("reports:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if p.Bearer {
			return auth.ErrForbidden
		}
		var v struct {
			Token string `json:"token"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		session := auth.Token()
		tag, e := a.DB.Exec(r.Context(), "UPDATE artifact_credentials SET state='leased',session_hash=$4 WHERE token_hash=$1 AND artifact_id::text=$2 AND user_session_id=$3 AND state='new' AND expires_at>clock_timestamp()", auth.Digest(v.Token), r.PathValue("id"), p.SessionID, auth.Digest(session))
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return auth.ErrCredentials
		}
		path := "/api/v1/artifacts/" + r.PathValue("id")
		http.SetCookie(w, &http.Cookie{Name: "filexArtifact", Value: session, Path: path, HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteStrictMode, MaxAge: 300})
		return reply(w, 200, map[string]string{"url": path + "/content"})
	}))
	m.HandleFunc("GET /api/v1/artifacts/{id}/content", a.artifactContent)
	m.HandleFunc("HEAD /api/v1/artifacts/{id}/content", a.artifactContent)
}
func (a *App) artifactContent(w http.ResponseWriter, r *http.Request) {
	if e := a.artifactStream(w, r); e != nil {
		failure(w, e)
	}
}
func (a *App) artifactStream(w http.ResponseWriter, r *http.Request) error {
	cookie, e := r.Cookie("filexArtifact")
	if e != nil {
		return auth.ErrCredentials
	}
	sessionCookie, e := r.Cookie("filexSession")
	if e != nil {
		return auth.ErrCredentials
	}
	var credential string
	if e = a.DB.QueryRow(r.Context(), `SELECT c.id::text FROM artifact_credentials c JOIN web_sessions s ON s.id=c.user_session_id JOIN users u ON u.id=s.user_id JOIN artifacts ar ON ar.id=c.artifact_id WHERE c.artifact_id::text=$1 AND c.session_hash=$2 AND s.token_hash=$3 AND c.state='leased' AND c.expires_at>clock_timestamp() AND s.revoked_at IS NULL AND s.auth_epoch=u.auth_epoch AND NOT u.disabled AND s.expires_at>clock_timestamp() AND ar.state='ready' AND ar.expires_at>clock_timestamp() AND NOT EXISTS(SELECT 1 FROM export_jobs ej WHERE ej.artifact_id=ar.id AND ej.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes')) AND NOT EXISTS(SELECT 1 FROM report_artifacts ra JOIN reports rp ON rp.id=ra.report_id WHERE ra.artifact_id=ar.id AND (rp.state<>'active' OR (rp.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes'))))`, r.PathValue("id"), auth.Digest(cookie.Value), auth.Digest(sessionCookie.Value)).Scan(&credential); e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return auth.ErrCredentials
		}
		return e
	}
	ar, e := a.artifactStore().Get(r.Context(), r.PathValue("id"))
	if e != nil {
		return e
	}
	start, end, status, e := byteRange(r.Header.Get("Range"), ar.Size)
	if e != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", ar.Size))
		w.WriteHeader(416)
		return nil
	}
	var fence int64
	if e = a.DB.QueryRow(r.Context(), "UPDATE artifact_credentials SET fence=fence+1,writer_until=clock_timestamp()+interval '10 seconds' WHERE id=$1 AND state='leased' AND expires_at>clock_timestamp() AND (writer_until IS NULL OR writer_until<clock_timestamp()) RETURNING fence", credential).Scan(&fence); e != nil {
		return transfer.ErrConflict
	}
	defer a.DB.Exec(context.Background(), "UPDATE artifact_credentials SET writer_until=NULL WHERE id=$1 AND fence=$2", credential, fence)
	writer := artifactWriter{a: a, r: r, id: credential, fence: fence, w: w}
	if e = writer.guard(); e != nil {
		return e
	}
	w.Header().Set("Content-Type", ar.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": ar.Filename}))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatInt(end-start, 10))
	if status == 206 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, ar.Size))
	}
	w.WriteHeader(status)
	if r.Method == "HEAD" {
		return nil
	}
	if e = a.artifactStore().Read(r.Context(), ar, start, end, writer); e != nil {
		panic(http.ErrAbortHandler)
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		panic(http.ErrAbortHandler)
	}
	defer tx.Rollback(context.Background())
	var raw []byte
	e = tx.QueryRow(r.Context(), `SELECT c.coverage FROM artifact_credentials c JOIN artifacts ar ON ar.id=c.artifact_id JOIN web_sessions s ON s.id=c.user_session_id JOIN users u ON u.id=s.user_id WHERE c.id=$1 AND c.fence=$2 AND c.state='leased' AND c.expires_at>clock_timestamp() AND c.writer_until>clock_timestamp() AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.auth_epoch=u.auth_epoch AND NOT u.disabled AND ar.state='ready' AND ar.expires_at>clock_timestamp() AND NOT EXISTS(SELECT 1 FROM export_jobs ej WHERE ej.artifact_id=ar.id AND ej.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes')) AND NOT EXISTS(SELECT 1 FROM report_artifacts ra JOIN reports rp ON rp.id=ra.report_id WHERE ra.artifact_id=ar.id AND (rp.state<>'active' OR (rp.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes')))) FOR UPDATE OF c FOR SHARE OF ar,s,u`, credential, fence).Scan(&raw)
	if e != nil {
		panic(http.ErrAbortHandler)
	}
	var covered [][2]int64
	if e = json.Unmarshal(raw, &covered); e != nil {
		panic(http.ErrAbortHandler)
	}
	covered = mergeIntervals(append(covered, [2]int64{start, end}))
	raw, e = json.Marshal(covered)
	if e != nil {
		panic(http.ErrAbortHandler)
	}
	state := "leased"
	if len(covered) == 1 && covered[0][0] == 0 && covered[0][1] == ar.Size {
		state = "completed"
	}
	if _, e = tx.Exec(r.Context(), "UPDATE artifact_credentials SET coverage=$2,state=$3 WHERE id=$1", credential, raw, state); e != nil {
		panic(http.ErrAbortHandler)
	}
	if e = tx.Commit(r.Context()); e != nil {
		panic(http.ErrAbortHandler)
	}
	return nil
}

type artifactWriter struct {
	a     *App
	r     *http.Request
	id    string
	fence int64
	w     http.ResponseWriter
}

func (g artifactWriter) guard() error {
	var ok bool
	e := g.a.DB.QueryRow(g.r.Context(), `SELECT EXISTS(SELECT 1 FROM artifact_credentials c JOIN artifacts ar ON ar.id=c.artifact_id JOIN web_sessions s ON s.id=c.user_session_id JOIN users u ON u.id=s.user_id WHERE c.id=$1 AND c.fence=$2 AND c.state='leased' AND c.expires_at>clock_timestamp() AND c.writer_until>clock_timestamp() AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.auth_epoch=u.auth_epoch AND NOT u.disabled AND ar.state='ready' AND ar.expires_at>clock_timestamp() AND NOT EXISTS(SELECT 1 FROM export_jobs ej WHERE ej.artifact_id=ar.id AND ej.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes')) AND NOT EXISTS(SELECT 1 FROM report_artifacts ra JOIN reports rp ON rp.id=ra.report_id WHERE ra.artifact_id=ar.id AND (rp.state<>'active' OR (rp.kind='admin' AND (u.role<>'admin' OR s.reauth_at<=clock_timestamp()-interval '5 minutes')))))`, g.id, g.fence).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return auth.ErrForbidden
	}
	return nil
}
func (g artifactWriter) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		if e := g.guard(); e != nil {
			return total, e
		}
		_ = http.NewResponseController(g.w).SetWriteDeadline(time.Now().Add(2 * time.Second))
		chunk := b[:min(32768, len(b))]
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
		if _, e = g.a.DB.Exec(g.r.Context(), "UPDATE artifact_credentials SET writer_until=clock_timestamp()+interval '10 seconds' WHERE id=$1 AND fence=$2", g.id, g.fence); e != nil {
			return total, e
		}
	}
	return total, nil
}

package httpapi

import (
	"context"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

func (a *App) adminExportRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/v1/admin/exports", a.protect("admin:exports", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		owner := r.URL.Query().Get("ownerId")
		if owner != "" && !domain.ValidID(owner) {
			return auth.ErrInvalid
		}
		rows, e := a.DB.Query(r.Context(), `SELECT e.id::text,e.owner_id::text,e.kind,e.format,e.state,e.artifact_id::text,a.expires_at,e.created_at FROM export_jobs e LEFT JOIN artifacts a ON a.id=e.artifact_id WHERE ($1='' OR e.owner_id::text=$1) ORDER BY e.created_at DESC,e.id DESC LIMIT 100`, owner)
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, owner, kind, format, state string
			var artifact *string
			var expires *time.Time
			var created time.Time
			if e = rows.Scan(&id, &owner, &kind, &format, &state, &artifact, &expires, &created); e != nil {
				return e
			}
			out = append(out, map[string]any{"id": id, "ownerId": owner, "kind": kind, "format": format, "state": state, "artifactId": artifact, "expiresAt": expires, "createdAt": created})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
	m.HandleFunc("POST /api/v1/admin/exports/{id}/credentials", a.protect("admin:exports", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Reason string `json:"reason"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if utf8.RuneCountInString(strings.TrimSpace(v.Reason)) < 5 || utf8.RuneCountInString(v.Reason) > 2000 {
			return auth.ErrInvalid
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		var artifact string
		if e = tx.QueryRow(r.Context(), `SELECT a.id::text FROM export_jobs e JOIN artifacts a ON a.id=e.artifact_id WHERE e.id::text=$1 AND e.state='completed' AND a.state='ready' AND a.expires_at>clock_timestamp() FOR UPDATE OF a`, r.PathValue("id")).Scan(&artifact); e != nil {
			return e
		}
		token := auth.Token()
		if _, e = tx.Exec(r.Context(), `INSERT INTO artifact_credentials(id,artifact_id,user_session_id,token_hash,expires_at,admin_access) VALUES($1,$2,$3,$4,clock_timestamp()+interval '5 minutes',true)`, domain.NewID(), artifact, p.SessionID, auth.Digest(token)); e != nil {
			return e
		}
		if e = adminEvent(r.Context(), tx, a.Key, p.ID, "export.admin_access", r.PathValue("id"), v.Reason); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"artifactId": artifact, "token": token})
	}))
	m.HandleFunc("PATCH /api/v1/admin/exports/{id}/ttl", a.protect("admin:exports", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Reason string `json:"reason"`
			TTL    int    `json:"ttlSeconds"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if utf8.RuneCountInString(strings.TrimSpace(v.Reason)) < 5 || utf8.RuneCountInString(v.Reason) > 2000 || v.TTL < 60 || v.TTL > 604800 {
			return auth.ErrInvalid
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		tag, e := tx.Exec(r.Context(), `UPDATE artifacts a SET expires_at=clock_timestamp()+make_interval(secs=>$2::int) FROM export_jobs e WHERE e.artifact_id=a.id AND e.id::text=$1 AND e.state='completed' AND a.state='ready' AND a.expires_at>clock_timestamp()`, r.PathValue("id"), v.TTL)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		if e = adminEvent(r.Context(), tx, a.Key, p.ID, "export.ttl_changed", r.PathValue("id"), v.Reason); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
}

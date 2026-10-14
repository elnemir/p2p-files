package httpapi

import (
	"context"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"time"
)

func (a *App) exportRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/exports", a.protect("exports:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Kind        string `json:"kind"`
			Format      string `json:"format"`
			Compression string `json:"compression"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if v.Kind == "admin" && (p.Role != "admin" || p.Bearer || time.Since(p.Reauth) > 5*time.Minute) {
			return auth.ErrForbidden
		}
		if v.Kind != "user" && v.Kind != "admin" || v.Format != "csv" && v.Format != "json" || v.Compression != "auto" && v.Compression != "none" && v.Compression != "gzip" && v.Compression != "zip" {
			return auth.ErrInvalid
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		if _, e = tx.Exec(r.Context(), "SELECT id FROM users WHERE id=$1 FOR UPDATE", p.ID); e != nil {
			return e
		}
		var n int
		if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM export_jobs WHERE owner_id=$1 AND state NOT IN ('completed','cancelled','failed')", p.ID).Scan(&n); e != nil {
			return e
		}
		if n >= 10 {
			return auth.ErrRate
		}
		id := domain.NewID()
		if _, e = licensing.Guard(r.Context(), tx, ""); e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "INSERT INTO export_jobs(id,owner_id,kind,format,compression,auth_epoch) SELECT $1,id,$3,$4,$5,auth_epoch FROM users WHERE id=$2 AND NOT disabled AND ($3='user' OR role='admin')", id, p.ID, v.Kind, v.Format, v.Compression); e != nil {
			return e
		}
		if e = audit.Append(r.Context(), tx, a.Key, p.ID, "export.created", id); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 202, map[string]string{"id": id, "state": "queued"})
	}))
	m.HandleFunc("GET /api/v1/exports", a.protect("reports:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), "SELECT id::text,kind,format,compression,state,rows_count,processed,snapshot_at,artifact_id::text,error_code,created_at FROM export_jobs WHERE owner_id=$1 AND (kind='user' OR $2='admin') ORDER BY created_at DESC LIMIT 100", p.ID, p.Role)
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, kind, format, compression, state string
			var count, progress int
			var snapshot *time.Time
			var artifact, errorCode *string
			var at time.Time
			if e = rows.Scan(&id, &kind, &format, &compression, &state, &count, &progress, &snapshot, &artifact, &errorCode, &at); e != nil {
				return e
			}
			out = append(out, map[string]any{"id": id, "kind": kind, "format": format, "compression": compression, "state": state, "rowCount": count, "processed": progress, "snapshotAt": snapshot, "artifactId": artifact, "errorCode": errorCode, "createdAt": at})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
	m.HandleFunc("POST /api/v1/exports/{id}/{action}", a.protect("exports:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		action := r.PathValue("action")
		if action != "cancel" && action != "retry" {
			return transfer.ErrNotFound
		}
		query := "UPDATE export_jobs SET state='cancelled',fence=fence+1,lease_until=NULL WHERE id::text=$1 AND owner_id=$2 AND state IN ('queued','snapshotting','materialized','rendering')"
		if action == "retry" {
			query = "UPDATE export_jobs SET state=CASE WHEN snapshot_at IS NULL THEN 'queued' ELSE 'materialized' END,fence=fence+1,lease_until=NULL,error_code=NULL WHERE id::text=$1 AND owner_id=$2 AND state='failed' AND (snapshot_at IS NULL OR rows_count=(SELECT count(*) FROM export_rows WHERE job_id=export_jobs.id))"
		}
		if action == "retry" {
			query += " AND EXISTS(SELECT 1 FROM users u WHERE u.id=owner_id AND NOT u.disabled AND u.auth_epoch=export_jobs.auth_epoch AND (export_jobs.kind='user' OR (u.role='admin' AND $3)))"
		}
		var tag pgconn.CommandTag
		var e error
		if action == "retry" {
			tag, e = a.DB.Exec(r.Context(), query, r.PathValue("id"), p.ID, !p.Bearer && time.Since(p.Reauth) <= 5*time.Minute)
		} else {
			tag, e = a.DB.Exec(r.Context(), query, r.PathValue("id"), p.ID)
		}
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		w.WriteHeader(204)
		return nil
	}))
}

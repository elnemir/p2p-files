package httpapi

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/reports"
	"filex.local/filex/internal/transfer"
	"net/http"
	"time"
)

func (a *App) artifactStore() artifacts.Store {
	return artifacts.Store{DB: a.DB, Keys: a.Keys, Storage: a.Files.Storage}
}
func (a *App) reportService() reports.Service {
	return reports.Service{DB: a.DB, Keys: a.Keys, Artifacts: a.artifactStore(), AuditKey: a.Key}
}
func (a *App) reportRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/transfers/{id}/reports", a.protect("reports:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Kind     string `json:"kind"`
			Locale   string `json:"locale"`
			Timezone string `json:"timezone"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if v.Kind == "admin" && (p.Role != "admin" || p.Bearer || time.Since(p.Reauth) > 5*time.Minute) {
			return auth.ErrForbidden
		}
		id, hash, e := a.reportService().Create(r.Context(), p.ID, r.PathValue("id"), v.Kind, v.Locale, v.Timezone)
		if e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"id": id, "canonicalHash": hash, "renderState": "pending"})
	}))
	m.HandleFunc("GET /api/v1/reports", a.protect("reports:read", false, a.reportList))
	m.HandleFunc("GET /api/v1/reports/{id}", a.protect("reports:read", false, a.reportDetail))
	m.HandleFunc("POST /api/v1/reports/{id}/state", a.protect("reports:write", false, a.reportState))
	m.HandleFunc("POST /api/v1/reports/{id}/retry", a.protect("reports:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		tag, e := a.DB.Exec(r.Context(), "UPDATE reports SET render_state='pending' WHERE id::text=$1 AND owner_id=$2 AND state='active' AND (kind='user' OR ($3='admin' AND $4)) AND (render_state='failed' OR (render_state='ready' AND (SELECT count(*) FROM report_artifacts ra JOIN artifacts a ON a.id=ra.artifact_id WHERE ra.report_id=reports.id AND a.state='ready' AND a.expires_at>clock_timestamp())<3))", r.PathValue("id"), p.ID, p.Role, !p.Bearer && time.Since(p.Reauth) <= 5*time.Minute)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("GET /api/v1/verify/reports/{id}", func(w http.ResponseWriter, r *http.Request) {
		var hash, state string
		var revision int
		var created time.Time
		e := a.DB.QueryRow(r.Context(), "SELECT canonical_hash,state,revision,created_at FROM reports WHERE id::text=$1 AND kind='user' AND state='active'", r.PathValue("id")).Scan(&hash, &state, &revision, &created)
		if e != nil {
			failure(w, e)
			return
		}
		_ = reply(w, 200, map[string]any{"canonicalHash": hash, "state": state, "revision": revision, "createdAt": created})
	})
	m.HandleFunc("POST /api/v1/verify-report", func(w http.ResponseWriter, r *http.Request) {
		var v struct {
			ReportID      string `json:"reportId"`
			Revision      int    `json:"revision"`
			CanonicalHash string `json:"canonicalHash"`
		}
		if e := body(w, r, &v); e != nil {
			failure(w, e)
			return
		}
		var id string
		e := a.DB.QueryRow(r.Context(), "SELECT id::text FROM reports WHERE id::text=$1 AND revision=$2 AND canonical_hash=$3 AND kind='user' AND state='active'", v.ReportID, v.Revision, v.CanonicalHash).Scan(&id)
		if e != nil {
			failure(w, e)
			return
		}
		_ = reply(w, 200, map[string]any{"verified": true, "reportId": id, "revision": v.Revision, "canonicalHash": v.CanonicalHash})
	})
	a.artifactRoutes(m)
}
func (a *App) reportList(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,transfer_id::text,kind,revision,canonical_hash,state,render_state,created_at FROM reports WHERE owner_id=$1 AND (kind='user' OR $2='admin') ORDER BY created_at DESC LIMIT 100", p.ID, p.Role)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, transferID, kind, hash, state, render string
		var rev int
		var at time.Time
		if e = rows.Scan(&id, &transferID, &kind, &rev, &hash, &state, &render, &at); e != nil {
			return e
		}
		out = append(out, map[string]any{"id": id, "transferId": transferID, "kind": kind, "revision": rev, "canonicalHash": hash, "state": state, "renderState": render, "createdAt": at})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, out)
}
func (a *App) reportDetail(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var id, state, render, kind string
	if e := a.DB.QueryRow(r.Context(), "SELECT id::text,state,render_state,kind FROM reports WHERE id::text=$1 AND owner_id=$2", r.PathValue("id"), p.ID).Scan(&id, &state, &render, &kind); e != nil {
		return e
	}
	if kind == "admin" && (p.Role != "admin" || p.Bearer || time.Since(p.Reauth) > 5*time.Minute) {
		return auth.ErrForbidden
	}
	out := map[string]any{"id": id, "state": state, "renderState": render}
	if state != "purged" {
		raw, hash, e := a.reportService().Snapshot(r.Context(), id)
		if e != nil {
			return e
		}
		out["snapshot"] = json.RawMessage(raw)
		out["canonicalHash"] = hash
	}
	rows, e := a.DB.Query(r.Context(), "SELECT ra.format,ra.artifact_id::text,COALESCE(ar.sha256,'') FROM report_artifacts ra JOIN artifacts ar ON ar.id=ra.artifact_id WHERE ra.report_id=$1 AND ar.state='ready' AND ar.expires_at>clock_timestamp()", id)
	if e != nil {
		return e
	}
	formats := map[string]string{}
	hashes := map[string]string{}
	for rows.Next() {
		var f, id, hash string
		if e = rows.Scan(&f, &id, &hash); e != nil {
			rows.Close()
			return e
		}
		formats[f] = id
		hashes[f] = hash
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	out["artifacts"] = formats
	out["artifactHashes"] = hashes
	rows, e = a.DB.Query(r.Context(), "SELECT action,created_at FROM report_history WHERE report_id=$1 ORDER BY id", id)
	if e != nil {
		return e
	}
	history := []map[string]any{}
	for rows.Next() {
		var action string
		var at time.Time
		if e = rows.Scan(&action, &at); e != nil {
			rows.Close()
			return e
		}
		history = append(history, map[string]any{"action": action, "createdAt": at})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	out["history"] = history
	return reply(w, 200, out)
}
func (a *App) reportState(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		State string `json:"state"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.State != "active" && v.State != "trashed" && v.State != "purged" {
		return auth.ErrInvalid
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var current string
	id := r.PathValue("id")
	if e = tx.QueryRow(r.Context(), "SELECT state FROM reports WHERE id::text=$1 AND owner_id=$2 FOR UPDATE", id, p.ID).Scan(&current); e != nil {
		return e
	}
	if current == "purged" || (v.State == "purged" && current != "trashed") {
		return transfer.ErrConflict
	}
	if _, e = tx.Exec(r.Context(), "UPDATE reports SET state=$2,snapshot_envelope=CASE WHEN $2='purged' THEN NULL ELSE snapshot_envelope END,render_fence=render_fence+1 WHERE id::text=$1", id, v.State); e != nil {
		return e
	}
	if v.State == "purged" {
		if _, e = tx.Exec(r.Context(), "UPDATE artifacts SET state='deleting' WHERE id IN (SELECT artifact_id FROM report_artifacts WHERE report_id::text=$1)", id); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO report_history(report_id,actor_id,action) VALUES($1,$2,$3)", id, p.ID, v.State); e != nil {
		return e
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "report."+v.State, id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}

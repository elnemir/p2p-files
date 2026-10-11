package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"fmt"
	"github.com/skip2/go-qrcode"
	"net/http"
	"strings"
	"time"
)

type filter struct {
	Query  string `json:"query"`
	State  string `json:"state"`
	After  string `json:"after"`
	ViewID string `json:"viewId"`
}

func (f filter) valid() bool {
	return len(f.Query) <= 500 && (f.State == "" || f.State == "draft" || f.State == "active" || f.State == "completed" || f.State == "revoked" || f.State == "expired")
}
func (a *App) organizationRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/transfers/search", a.protect("transfers:read", false, a.searchTransfers))
	m.HandleFunc("POST /api/v1/views", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Name   string `json:"name"`
			Filter filter `json:"filter"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if len(v.Name) < 1 || len(v.Name) > 100 || !v.Filter.valid() || v.Filter.ViewID != "" {
			return auth.ErrInvalid
		}
		v.Filter.After = ""
		raw, e := json.Marshal(v.Filter)
		if e != nil {
			return e
		}
		id := domain.NewID()
		if _, e = a.DB.Exec(r.Context(), "INSERT INTO saved_views(id,owner_id,name,filter) VALUES($1,$2,$3,$4)", id, p.ID, v.Name, raw); e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"id": id})
	}))
	m.HandleFunc("GET /api/v1/views", a.protect("account:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), "SELECT id::text,name,filter FROM saved_views WHERE owner_id=$1 ORDER BY created_at DESC LIMIT 100", p.ID)
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, name string
			var raw json.RawMessage
			if e = rows.Scan(&id, &name, &raw); e != nil {
				return e
			}
			out = append(out, map[string]any{"id": id, "name": name, "filter": raw})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
	m.HandleFunc("DELETE /api/v1/views/{id}", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if _, e := a.DB.Exec(r.Context(), "DELETE FROM saved_views WHERE id::text=$1 AND owner_id=$2", r.PathValue("id"), p.ID); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("PATCH /api/v1/transfers/{id}", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Title   string `json:"title"`
			Message string `json:"message"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if len(v.Title) > 300 || len(v.Message) > 5000 {
			return auth.ErrInvalid
		}
		tag, e := a.DB.Exec(r.Context(), "UPDATE transfers SET title=$3,message=$4 WHERE id::text=$1 AND owner_id=$2", r.PathValue("id"), p.ID, v.Title, v.Message)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrNotFound
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("POST /api/v1/transfers/{id}/folders", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Path string `json:"path"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		key, e := domain.PathKey(v.Path, c.MaxPathBytes)
		if e != nil {
			return e
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		var parent string
		if e = tx.QueryRow(r.Context(), "SELECT id::text FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='draft' FOR UPDATE", r.PathValue("id"), p.ID).Scan(&parent); e != nil {
			return e
		}
		var conflict bool
		if e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND (path_key=$2 OR starts_with($2,path_key||'/')) AND state NOT IN ('cancelled','deleted'))", parent, key).Scan(&conflict); e != nil {
			return e
		}
		if conflict {
			return transfer.ErrConflict
		}
		id := domain.NewID()
		if _, e = tx.Exec(r.Context(), "INSERT INTO draft_folders(id,transfer_id,path,path_key) VALUES($1,$2,$3,$4)", id, parent, v.Path, key); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"id": id, "path": v.Path})
	}))
	m.HandleFunc("GET /api/v1/transfers/{id}/folders", a.protect("transfers:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), "SELECT f.id::text,f.path FROM draft_folders f JOIN transfers t ON t.id=f.transfer_id WHERE t.id::text=$1 AND t.owner_id=$2 ORDER BY f.path", r.PathValue("id"), p.ID)
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]string{}
		for rows.Next() {
			var id, path string
			if e = rows.Scan(&id, &path); e != nil {
				return e
			}
			out = append(out, map[string]string{"id": id, "path": path})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
	m.HandleFunc("GET /api/v1/transfers/{id}/links/{link}/qr", a.protect("transfers:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		links, e := a.ownerLinks(r, r.PathValue("id"), p.ID, c)
		if e != nil {
			return e
		}
		for _, link := range links {
			if link["id"] == r.PathValue("link") {
				qr, e := qrcode.New(link["url"].(string), qrcode.Medium)
				if e != nil {
					return e
				}
				if r.URL.Query().Get("format") == "svg" {
					w.Header().Set("Content-Type", "image/svg+xml")
					bitmap := qr.Bitmap()
					fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-label="QR"><rect width="100%%" height="100%%" fill="white"/><g fill="black">`, len(bitmap), len(bitmap))
					for y, row := range bitmap {
						for x, on := range row {
							if on {
								fmt.Fprintf(w, `<rect x="%d" y="%d" width="1" height="1"/>`, x, y)
							}
						}
					}
					_, e = fmt.Fprint(w, "</g></svg>")
					return e
				}
				png, e := qr.PNG(512)
				if e != nil {
					return e
				}
				w.Header().Set("Content-Type", "image/png")
				_, e = w.Write(png)
				return e
			}
		}
		return transfer.ErrNotFound
	}))
	m.HandleFunc("POST /api/v1/transfers/{id}/qr-bundle", a.protect("transfers:read", false, a.qrBundle))

	a.bulkRoutes(m)
}
func (a *App) searchTransfers(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var f filter
	if e := body(w, r, &f); e != nil {
		return e
	}
	if f.ViewID != "" {
		var raw []byte
		if e := a.DB.QueryRow(r.Context(), "SELECT filter FROM saved_views WHERE id::text=$1 AND owner_id=$2", f.ViewID, p.ID).Scan(&raw); e != nil {
			return e
		}
		after := f.After
		if e := json.Unmarshal(raw, &f); e != nil {
			return e
		}
		f.After = after
	}
	if !f.valid() {
		return auth.ErrInvalid
	}
	cursor := struct {
		At time.Time
		ID string
	}{At: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), ID: "ffffffff-ffff-ffff-ffff-ffffffffffff"}
	if f.After != "" {
		raw, e := base64.RawURLEncoding.DecodeString(f.After)
		if e != nil || json.Unmarshal(raw, &cursor) != nil || !domain.ValidID(cursor.ID) {
			return auth.ErrInvalid
		}
	}
	q := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(f.Query)
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,title,state,revision,created_at FROM transfers WHERE owner_id=$1 AND ($2='' OR state=$2) AND ($3='' OR title ILIKE '%'||$3||'%') AND (created_at,id)<($4,$5::uuid) ORDER BY created_at DESC,id DESC LIMIT 101", p.ID, f.State, q, cursor.At, cursor.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	next := ""
	for rows.Next() {
		var id, title, state string
		var rev int64
		var at time.Time
		if e = rows.Scan(&id, &title, &state, &rev, &at); e != nil {
			return e
		}
		if len(out) == 100 {
			last := out[99]
			raw, _ := json.Marshal(struct {
				At time.Time
				ID string
			}{last["createdAt"].(time.Time), last["id"].(string)})
			next = base64.RawURLEncoding.EncodeToString(raw)
			break
		}
		out = append(out, map[string]any{"id": id, "title": title, "state": state, "revision": rev, "createdAt": at})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, map[string]any{"items": out, "nextCursor": next})
}
func (a *App) bulkRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/bulk", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			IDs    []string `json:"ids"`
			Action string   `json:"action"`
			Admin  bool     `json:"adminScope"`
			Reason string   `json:"reason"`
			Dry    bool     `json:"dryRun"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if len(v.IDs) == 0 || len(v.IDs) > 1000 || (v.Action != "revoke" && v.Action != "cancel_upload") {
			return auth.ErrInvalid
		}
		if v.Admin && (p.Role != "admin" || p.Bearer || time.Since(p.Reauth) > 5*time.Minute || len(v.Reason) < 5) {
			return auth.ErrForbidden
		}
		for _, id := range v.IDs {
			if !domain.ValidID(id) {
				return auth.ErrInvalid
			}
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		id := domain.NewID()
		state := "queued"
		if v.Dry {
			state = "paused"
		}
		if _, e = tx.Exec(r.Context(), "INSERT INTO bulk_jobs(id,owner_id,action,state,admin_scope,account_epoch,reason) VALUES($1,$2,$3,$4,$5,$6,$7)", id, p.ID, v.Action, state, v.Admin, p.Epoch, v.Reason); e != nil {
			return e
		}
		query := "INSERT INTO bulk_items(job_id,subject_id) SELECT $1,id FROM transfers WHERE id=ANY($2::uuid[]) AND (owner_id=$3 OR $4)"
		if v.Action == "cancel_upload" {
			query = "INSERT INTO bulk_items(job_id,subject_id) SELECT $1,f.id FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id=ANY($2::uuid[]) AND (t.owner_id=$3 OR $4)"
		}
		tag, e := tx.Exec(r.Context(), query, id, v.IDs, p.ID, v.Admin)
		if e != nil {
			return e
		}
		seen := map[string]bool{}
		for _, x := range v.IDs {
			seen[x] = true
		}
		if tag.RowsAffected() != int64(len(seen)) {
			return auth.ErrForbidden
		}
		if e = audit.Append(r.Context(), tx, a.Key, p.ID, "bulk.selection_created", id); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 202, map[string]any{"id": id, "state": state, "selected": tag.RowsAffected()})
	}))
	m.HandleFunc("POST /api/v1/bulk/{id}/{action}", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		action := r.PathValue("action")
		state := ""
		if action == "run" {
			state = "queued"
		} else if action == "cancel" {
			state = "cancelled"
		} else {
			return auth.ErrInvalid
		}
		tag, e := a.DB.Exec(r.Context(), "UPDATE bulk_jobs SET state=$3 WHERE id::text=$1 AND owner_id=$2 AND ((state='paused' AND $3='queued') OR (state IN ('paused','queued','running') AND $3='cancelled'))", r.PathValue("id"), p.ID, state)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("GET /api/v1/bulk", a.protect("transfers:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), "SELECT j.id::text,j.action,j.state,count(i.subject_id),count(*) FILTER(WHERE i.state='done'),count(*) FILTER(WHERE i.state='conflict') FROM bulk_jobs j LEFT JOIN bulk_items i ON i.job_id=j.id WHERE j.owner_id=$1 GROUP BY j.id ORDER BY j.created_at DESC LIMIT 100", p.ID)
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, action, state string
			var total, done, conflicts int64
			if e = rows.Scan(&id, &action, &state, &total, &done, &conflicts); e != nil {
				return e
			}
			out = append(out, map[string]any{"id": id, "action": action, "state": state, "total": total, "done": done, "conflicts": conflicts})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
}

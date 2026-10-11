package httpapi

import (
	"context"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (a *App) maintenanceRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/admin/storage/migrations", a.protect("admin:storage", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Source string `json:"sourceId"`
			Target string `json:"targetId"`
			Dry    bool   `json:"dryRun"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if !domain.ValidID(v.Source) || !domain.ValidID(v.Target) || v.Source == v.Target {
			return auth.ErrInvalid
		}
		var valid bool
		if e := a.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM storage_providers s JOIN storage_providers t ON t.id=$2 WHERE s.id=$1 AND s.purpose='payload' AND t.purpose='payload' AND t.enabled AND NOT t.draining)", v.Source, v.Target).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return auth.ErrInvalid
		}
		var count, size int64
		if e := a.DB.QueryRow(r.Context(), "SELECT count(*),COALESCE(sum(size),0) FROM files WHERE provider_id=$1 AND state NOT IN ('deleted','cancelled','deleting')", v.Source).Scan(&count, &size); e != nil {
			return e
		}
		if v.Dry {
			return reply(w, 200, map[string]int64{"files": count, "bytes": size})
		}
		if _, e := a.Files.Storage.Get(r.Context(), v.Target); e != nil {
			return e
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		id := domain.NewID()
		if _, e = tx.Exec(r.Context(), "INSERT INTO storage_migrations(id,source_id,target_id) VALUES($1,$2,$3)", id, v.Source, v.Target); e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "UPDATE storage_providers SET draining=true WHERE id=$1", v.Source); e != nil {
			return e
		}
		if e = audit.Append(r.Context(), tx, a.Key, p.ID, "storage.migration_created", id); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"id": id, "state": "paused"})
	}))
	m.HandleFunc("PATCH /api/v1/admin/storage/migrations/{id}", a.protect("admin:storage", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			State string `json:"state"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if v.State != "running" && v.State != "paused" && v.State != "cancelled" {
			return auth.ErrInvalid
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(context.Background())
		tag, e := tx.Exec(r.Context(), "UPDATE storage_migrations SET state=$2 WHERE id::text=$1 AND state IN ('paused','running')", r.PathValue("id"), v.State)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		if e = audit.Append(r.Context(), tx, a.Key, p.ID, "storage.migration_"+v.State, r.PathValue("id")); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("PATCH /api/v1/admin/keys/rotations/{id}", a.protect("admin:keys", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			State string `json:"state"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if v.State != "running" && v.State != "paused" && v.State != "cancelled" {
			return auth.ErrInvalid
		}
		tx, e := a.DB.Begin(r.Context())
		if e != nil {
			return e
		}
		defer tx.Rollback(r.Context())
		tag, e := tx.Exec(r.Context(), "UPDATE rotation_jobs SET state=$2 WHERE id::text=$1 AND state IN ('running','paused') AND EXISTS(SELECT 1 FROM key_versions k WHERE k.id=target_key_id AND k.active)", r.PathValue("id"), v.State)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
		if e = audit.Append(r.Context(), tx, a.Key, p.ID, "rotation."+v.State, r.PathValue("id")); e != nil {
			return e
		}
		if e = tx.Commit(r.Context()); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("POST /api/v1/admin/keys/rotate", a.protect("admin:keys", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Class string `json:"class"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		id := domain.NewID()
		state := "running"
		if v.Class == "signing" {
			state = "completed"
		}
		kid, e := a.Keys.RotateWith(r.Context(), v.Class, func(tx pgx.Tx, kid string) error {
			var pending bool
			if e := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM rotation_jobs WHERE class=$1 AND state IN ('running','paused'))", v.Class).Scan(&pending); e != nil {
				return e
			}
			if pending {
				return transfer.ErrConflict
			}
			if _, e := tx.Exec(r.Context(), "INSERT INTO rotation_jobs(id,class,target_key_id,state) VALUES($1,$2,$3,$4)", id, v.Class, kid, state); e != nil {
				return e
			}
			return audit.Append(r.Context(), tx, a.Key, p.ID, "key.rotated", kid)
		})
		if e != nil {
			return auth.ErrInvalid
		}

		if v.Class == "signing" {
			if _, _, e = a.Keys.Signing(r.Context()); e != nil {
				return e
			}
		}
		return reply(w, 202, map[string]string{"id": id, "keyId": kid, "state": state})
	}))
	m.HandleFunc("GET /api/v1/admin/maintenance", a.protect("admin:storage", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), "SELECT id::text,'storage' AS kind,state,moved_files AS progress FROM storage_migrations UNION ALL SELECT id::text,class,state,processed FROM rotation_jobs LIMIT 100")
		if e != nil {
			return e
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, kind, state string
			var progress int
			if e = rows.Scan(&id, &kind, &state, &progress); e != nil {
				return e
			}
			out = append(out, map[string]any{"id": id, "kind": kind, "state": state, "progress": progress})
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
	m.HandleFunc("GET /api/v1/signing-keys/{id}", func(w http.ResponseWriter, r *http.Request) {
		var pub []byte
		if e := a.DB.QueryRow(r.Context(), "SELECT public_key FROM signing_keys WHERE id::text=$1", r.PathValue("id")).Scan(&pub); e != nil {
			failure(w, e)
			return
		}
		_ = reply(w, 200, map[string]any{"id": r.PathValue("id"), "publicKey": pub, "algorithm": "Ed25519"})
	})
}

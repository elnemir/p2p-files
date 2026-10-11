package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/storage"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"time"
)

func (a *App) adminFileRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/admin/transfers/{id}/hold", a.protect("admin:files", true, a.hold))
	m.HandleFunc("POST /api/v1/admin/transfers/{id}/forensic-zip", forensicForm(a.protect("admin:files", true, a.forensicZIP)))
	m.HandleFunc("POST /api/v1/admin/files/{id}/forensic", forensicForm(a.protect("admin:files", true, a.forensic)))
	m.HandleFunc("POST /api/v1/admin/files/{id}/check", a.protect("admin:files", true, a.checkFile))
	m.HandleFunc("POST /api/v1/admin/files/{id}/force-delete", a.protect("admin:files", true, a.forceDelete))
	m.HandleFunc("POST /api/v1/admin/storage", a.protect("admin:storage", true, a.addStorage))
	m.HandleFunc("GET /api/v1/admin/storage", a.protect("admin:storage", true, a.listStorage))
	m.HandleFunc("PATCH /api/v1/admin/storage/{id}", a.protect("admin:storage", true, a.editStorage))
	m.HandleFunc("POST /api/v1/transfers/{id}/links/{link}/rotate", a.protect("transfers:write", false, a.rotateLink))
	m.HandleFunc("DELETE /api/v1/transfers/{id}/links/{link}", a.protect("transfers:write", false, a.revokeLink))
	m.HandleFunc("GET /api/v1/transfers/{id}/journal", a.protect("transfers:read", false, a.journal))
}
func (a *App) hold(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Frozen bool   `json:"frozen"`
		FileID string `json:"fileId"`
		Reason string `json:"reason"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if len(v.Reason) < 5 || len(v.Reason) > 2000 {
		return auth.ErrInvalid
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	id := r.PathValue("id")
	var old bool
	var state string
	if e = tx.QueryRow(r.Context(), "SELECT frozen,state FROM transfers WHERE id::text=$1 FOR UPDATE", id).Scan(&old, &state); e != nil {
		return e
	}
	subject := id
	if v.FileID != "" {
		subject = v.FileID
		tag, e := tx.Exec(r.Context(), "UPDATE files SET frozen=$3 WHERE id::text=$1 AND transfer_id::text=$2 AND state<>'deleted'", v.FileID, id, v.Frozen)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return transfer.ErrConflict
		}
	} else if old != v.Frozen {
		if v.Frozen {
			_, e = tx.Exec(r.Context(), "UPDATE transfers SET frozen=true,remaining_ttl_seconds=CASE WHEN state='active' THEN GREATEST(0,extract(epoch FROM expires_at-clock_timestamp())) ELSE NULL END WHERE id::text=$1", id)
		} else {
			_, e = tx.Exec(r.Context(), "UPDATE transfers SET frozen=false,expires_at=CASE WHEN state='active' THEN LEAST(absolute_expires_at,clock_timestamp()+make_interval(secs=>COALESCE(remaining_ttl_seconds,0))) ELSE expires_at END,remaining_ttl_seconds=NULL WHERE id::text=$1", id)
		}
		if e != nil {
			return e
		}
	}
	// Epoch cancellation is necessary even for file holds because a ZIP may include that file.
	if _, e = tx.Exec(r.Context(), "UPDATE transfers SET access_epoch=access_epoch+1 WHERE id::text=$1", id); e != nil {
		return e
	}
	action := "hold.released"
	if v.Frozen {
		action = "hold.applied"
	}
	if e = adminEvent(r.Context(), tx, a.Key, p.ID, action, subject, v.Reason); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func adminEvent(ctx context.Context, tx pgx.Tx, key []byte, actor, action, subject, reason string) error {
	id := domain.NewID()
	if _, e := tx.Exec(ctx, "INSERT INTO administrative_actions(id,actor_id,subject_id,action,reason) VALUES($1,$2,$3,$4,$5)", id, actor, subject, action, reason); e != nil {
		return e
	}
	h := sha256.Sum256([]byte(reason))
	return audit.Append(ctx, tx, key, actor, action, subject+":"+id+":"+hex.EncodeToString(h[:]))
}
func (a *App) forensic(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Reason string `json:"reason"`
		Raw    bool   `json:"rawCiphertext"`
		Object string `json:"objectKey"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if len(v.Reason) < 5 || len(v.Reason) > 2000 {
		return auth.ErrInvalid
	}
	var owner string
	if e := a.DB.QueryRow(r.Context(), "SELECT t.owner_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id::text=$1", r.PathValue("id")).Scan(&owner); e != nil {
		return e
	}
	f, e := a.Files.Get(r.Context(), owner, r.PathValue("id"))
	if e != nil {
		return e
	}
	if f.State == "deleted" {
		return transfer.ErrConflict
	}
	if v.Raw && f.State != "integrity_failed" {
		return auth.ErrForbidden
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = adminEvent(r.Context(), tx, a.Key, p.ID, "forensic.read", f.ID, v.Reason); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	if v.Raw {
		var object string
		if e = a.DB.QueryRow(r.Context(), "SELECT object_key FROM file_chunks WHERE file_id=$1 AND object_key=$2", f.ID, v.Object).Scan(&object); e != nil {
			return e
		}
		provider, e := a.Files.Storage.Get(r.Context(), f.ProviderID)
		if e != nil {
			return e
		}
		b, e := provider.Get(r.Context(), object)
		if e != nil {
			return e
		}
		_, e = (adminWriter{a, r, p, w}).Write(b)
		return e
	}
	if e = a.Files.Read(r.Context(), f, 0, f.Size, adminWriter{a, r, p, w}); e != nil {
		panic(http.ErrAbortHandler)
	}
	return nil
}

type adminWriter struct {
	a *App
	r *http.Request
	p auth.Principal
	w http.ResponseWriter
}

func (g adminWriter) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		var ok bool
		e := g.a.DB.QueryRow(g.r.Context(), "SELECT EXISTS(SELECT 1 FROM web_sessions s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND u.role='admin' AND NOT u.disabled AND s.revoked_at IS NULL AND s.auth_epoch=u.auth_epoch AND s.expires_at>clock_timestamp() AND s.reauth_at>clock_timestamp()-interval '5 minutes')", g.p.SessionID).Scan(&ok)
		if e != nil {
			return total, e
		}
		if !ok {
			return total, auth.ErrForbidden
		}
		_ = http.NewResponseController(g.w).SetWriteDeadline(time.Now().Add(2 * time.Second))
		n, e := g.w.Write(b[:min(32768, len(b))])
		total += n
		if e != nil {
			return total, e
		}
		if n == 0 {
			return total, io.ErrShortWrite
		}
		b = b[n:]
	}
	return total, nil
}
func (a *App) checkFile(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Reason string `json:"reason"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if len(v.Reason) < 5 || len(v.Reason) > 2000 {
		return auth.ErrInvalid
	}
	var owner string
	if e := a.DB.QueryRow(r.Context(), "SELECT t.owner_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id::text=$1", r.PathValue("id")).Scan(&owner); e != nil {
		return e
	}
	f, e := a.Files.Get(r.Context(), owner, r.PathValue("id"))
	if e != nil {
		return e
	}
	h := sha256.New()
	if e = a.Files.Read(r.Context(), f, 0, f.Size, h); e != nil {
		return e
	}
	if f.SHA256 == nil || hex.EncodeToString(h.Sum(nil)) != *f.SHA256 {
		if e = a.Files.MarkFailure(r.Context(), f, "integrity_failed"); e != nil {
			return e
		}
		return transfer.ErrConflict
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var state string
	if e = tx.QueryRow(r.Context(), "SELECT state FROM transfers WHERE id=$1 FOR UPDATE", f.TransferID).Scan(&state); e != nil {
		return e
	}
	if state == "active" {
		if _, e = tx.Exec(r.Context(), "UPDATE files SET state='ready' WHERE id=$1 AND state IN ('storage_missing','integrity_failed')", f.ID); e != nil {
			return e
		}
	}
	if e = adminEvent(r.Context(), tx, a.Key, p.ID, "file.checked", f.ID, v.Reason); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 200, map[string]string{"status": "verified"})
}
func (a *App) forceDelete(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Reason string `json:"reason"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if len(v.Reason) < 5 || len(v.Reason) > 2000 {
		return auth.ErrInvalid
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var id string
	var hold bool
	if e = tx.QueryRow(r.Context(), "SELECT t.id::text,t.frozen FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id::text=$1 FOR UPDATE OF t", r.PathValue("id")).Scan(&id, &hold); e != nil {
		return e
	}
	if hold {
		return transfer.ErrConflict
	}
	if _, e = tx.Exec(r.Context(), "UPDATE transfers SET access_epoch=access_epoch+1 WHERE id=$1", id); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "UPDATE files SET state='deleting',frozen=false,upload_epoch=upload_epoch+1 WHERE id::text=$1", r.PathValue("id")); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.id::text=$1 ON CONFLICT(object_key) DO NOTHING", r.PathValue("id")); e != nil {
		return e
	}
	if e = adminEvent(r.Context(), tx, a.Key, p.ID, "file.force_deleted", r.PathValue("id"), v.Reason); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) addStorage(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Kind     string           `json:"kind"`
		Purpose  string           `json:"purpose"`
		Root     string           `json:"root"`
		S3       storage.S3Config `json:"s3"`
		Priority int              `json:"priority"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.Purpose == "" {
		v.Purpose = "payload"
	}
	if v.Purpose != "payload" && v.Purpose != "artifacts" {
		return auth.ErrInvalid
	}
	id := domain.NewID()
	var provider storage.Provider
	var raw []byte
	var box []byte
	var kid *string
	if v.Kind == "local" {
		provider = storage.Local{Root: v.Root}
		raw, _ = json.Marshal(map[string]string{"root": v.Root})
	} else if v.Kind == "s3" {
		s, e := storage.NewS3(v.S3)
		if e != nil {
			return auth.ErrInvalid
		}
		provider = s
		keyID, key, e := a.Keys.Active(r.Context(), "configuration")
		if e != nil {
			return e
		}
		kid = &keyID
		secret, e := json.Marshal(v.S3)
		if e != nil {
			return e
		}
		box, e = a.Keys.Seal(r.Context(), keyID, key, "filex/storage/v1/"+id, secret)
		if e != nil {
			return e
		}
		raw = []byte(`{}`)
	} else {
		return auth.ErrInvalid
	}
	probe := domain.NewID()
	if e := provider.Put(r.Context(), probe, []byte("filex-storage-probe")); e != nil {
		return e
	}
	b, e := provider.Get(r.Context(), probe)
	del := provider.Delete(r.Context(), probe)
	if e != nil {
		return e
	}
	if del != nil {
		return del
	}
	if string(b) != "filex-storage-probe" {
		return transfer.ErrConflict
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if kid != nil {
		if e = keyring.RequireActive(r.Context(), tx, *kid); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO storage_providers(id,kind,config,priority,secret_key_id,secret_envelope,purpose) VALUES($1,$2,$3,$4,$5,$6,$7)", id, v.Kind, raw, v.Priority, kid, box, v.Purpose); e != nil {
		return e
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "storage.created", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 201, map[string]string{"id": id})
}
func (a *App) listStorage(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,kind,purpose,priority,enabled,draining FROM storage_providers ORDER BY priority DESC,id")
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, kind, purpose string
		var priority int
		var enabled, draining bool
		if e = rows.Scan(&id, &kind, &purpose, &priority, &enabled, &draining); e != nil {
			return e
		}
		out = append(out, map[string]any{"id": id, "kind": kind, "purpose": purpose, "priority": priority, "enabled": enabled, "draining": draining})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, out)
}
func (a *App) editStorage(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Enabled  bool `json:"enabled"`
		Draining bool `json:"draining"`
		Priority int  `json:"priority"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	tag, e := tx.Exec(r.Context(), "UPDATE storage_providers SET enabled=$2,draining=$3,priority=$4 WHERE id::text=$1", r.PathValue("id"), v.Enabled, v.Draining, v.Priority)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrNotFound
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "storage.updated", r.PathValue("id")); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) rotateLink(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Password string `json:"password"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	hash := ""
	var e error
	if v.Password != "" {
		hash, e = auth.LinkSecret(v.Password)
		if e != nil {
			return e
		}
	}
	kid, key, e := a.Keys.Active(r.Context(), "configuration")
	if e != nil {
		return e
	}
	token := auth.Token()
	id := r.PathValue("link")
	box, e := a.Keys.Seal(r.Context(), kid, key, "filex/link/v1/"+id, []byte(token))
	if e != nil {
		return e
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var parent string
	if e = tx.QueryRow(r.Context(), "SELECT id::text FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='active' FOR UPDATE", r.PathValue("id"), p.ID).Scan(&parent); e != nil {
		return e
	}
	if e = keyring.RequireActive(r.Context(), tx, kid); e != nil {
		return e
	}
	tag, e := tx.Exec(r.Context(), "UPDATE recipient_links SET token_hash=$2,token_envelope=$3,password_hash=NULLIF($4,''),key_id=$5,epoch=epoch+1,revoked_at=NULL WHERE id::text=$1 AND transfer_id=$6", id, auth.Digest(token), box, hash, kid, parent)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrNotFound
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "link.rotated", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 200, map[string]string{"url": c.PublicOrigin + "/receive#token=" + token})
}
func (a *App) revokeLink(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var id string
	if e = tx.QueryRow(r.Context(), "SELECT id::text FROM transfers WHERE id::text=$1 AND owner_id=$2 FOR UPDATE", r.PathValue("id"), p.ID).Scan(&id); e != nil {
		return e
	}
	tag, e := tx.Exec(r.Context(), "UPDATE recipient_links SET revoked_at=clock_timestamp(),epoch=epoch+1 WHERE id::text=$1 AND transfer_id=$2", r.PathValue("link"), id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrNotFound
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "link.revoked", r.PathValue("link")); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) journal(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT j.occurred_at,j.file_count,j.total_bytes,j.status FROM download_journal j JOIN transfers t ON t.id=j.transfer_id WHERE t.id::text=$1 AND t.owner_id=$2 ORDER BY j.id DESC LIMIT 100", r.PathValue("id"), p.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var at time.Time
		var count, bytes int64
		var status string
		if e = rows.Scan(&at, &count, &bytes, &status); e != nil {
			return e
		}
		out = append(out, map[string]any{"occurredAt": at, "fileCount": count, "totalBytes": bytes, "status": status})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, out)
}

package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) transferRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/transfers", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		d, e := transfer.New(a.DB, c).CreateDraft(r.Context(), p.ID)
		if e != nil {
			return e
		}
		return reply(w, 201, d)
	}))
	mux.HandleFunc("GET /api/v1/transfers", a.protect("transfers:read", false, a.listTransfers))
	mux.HandleFunc("GET /api/v1/transfers/{id}", a.protect("transfers:read", false, a.transferDetail))
	mux.HandleFunc("POST /api/v1/transfers/{id}/publish", a.protect("transfers:write", false, a.publish))
	mux.HandleFunc("POST /api/v1/transfers/{id}/revoke", a.protect("transfers:write", false, a.revoke))
	mux.HandleFunc("OPTIONS /api/v1/transfers/{id}/uploads", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Tus-Resumable", "1.0.0")
		w.Header().Set("Tus-Version", "1.0.0")
		w.Header().Set("Tus-Extension", "creation,termination")
		w.Header().Set("Tus-Max-Size", "1099511627776")
		w.WriteHeader(204)
	})
	mux.HandleFunc("POST /api/v1/transfers/{id}/uploads", a.protect("transfers:write", false, a.createUpload))
	mux.HandleFunc("HEAD /api/v1/uploads/{id}", a.protect("transfers:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		w.Header().Set("Tus-Resumable", "1.0.0")
		f, e := a.Files.Get(r.Context(), p.ID, r.PathValue("id"))
		if e != nil {
			return e
		}
		w.Header().Set("Upload-Length", strconv.FormatInt(f.Size, 10))
		w.Header().Set("Upload-Offset", strconv.FormatInt(f.Offset, 10))
		w.WriteHeader(200)
		return nil
	}))
	mux.HandleFunc("PATCH /api/v1/uploads/{id}", a.protect("transfers:write", false, a.patchUpload))
	mux.HandleFunc("DELETE /api/v1/uploads/{id}", a.protect("transfers:write", false, a.cancelUpload))
	mux.HandleFunc("PATCH /api/v1/files/{id}", a.protect("transfers:write", false, a.renameFile))
	mux.HandleFunc("POST /api/v1/uploads/{id}/finalize", a.protect("transfers:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if e := a.Files.Finalize(r.Context(), p.ID, r.PathValue("id"), c); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
}
func (a *App) listTransfers(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	cursor := r.URL.Query().Get("cursor")
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,state,revision,title,manifest_hash FROM transfers WHERE owner_id=$1 AND ($2='' OR id::text>$2) ORDER BY id LIMIT 101", p.ID, cursor)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, state, title string
		var revision int64
		var hash *string
		if e = rows.Scan(&id, &state, &revision, &title, &hash); e != nil {
			return e
		}
		items = append(items, map[string]any{"id": id, "state": state, "revision": revision, "title": title, "manifestHash": hash})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	next := ""
	if len(items) > 100 {
		items = items[:100]
		next = items[99]["id"].(string)
	}
	return reply(w, 200, map[string]any{"items": items, "nextCursor": next})
}
func (a *App) transferDetail(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	id := r.PathValue("id")
	var state, title string
	var revision int64
	var hash *string
	e := a.DB.QueryRow(r.Context(), "SELECT state,title,revision,manifest_hash FROM transfers WHERE id::text=$1 AND owner_id=$2", id, p.ID).Scan(&state, &title, &revision, &hash)
	if e != nil {
		return e
	}
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,path,size,upload_offset,state,sha256 FROM files WHERE transfer_id::text=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted') ORDER BY path", id)
	if e != nil {
		return e
	}
	items := []files.File{}
	for rows.Next() {
		var f files.File
		if e = rows.Scan(&f.ID, &f.Path, &f.Size, &f.Offset, &f.State, &f.SHA256); e != nil {
			rows.Close()
			return e
		}
		items = append(items, f)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	links, e := a.ownerLinks(r, id, p.ID, c)
	if e != nil {
		return e
	}
	return reply(w, 200, map[string]any{"id": id, "state": state, "title": title, "revision": revision, "manifestHash": hash, "files": items, "links": links})
}
func (a *App) createUpload(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	w.Header().Set("Tus-Resumable", "1.0.0")
	if r.Header.Get("Tus-Resumable") != "1.0.0" {
		w.Header().Set("Tus-Version", "1.0.0")
		w.WriteHeader(412)
		return nil
	}
	size, e := strconv.ParseInt(r.Header.Get("Upload-Length"), 10, 64)
	if e != nil {
		return auth.ErrInvalid
	}
	path := ""
	seen := map[string]bool{}
	for _, part := range strings.Split(r.Header.Get("Upload-Metadata"), ",") {
		pair := strings.SplitN(strings.TrimSpace(part), " ", 2)
		key := pair[0]
		if key == "" || seen[key] || strings.ContainsAny(key, "\r\n\t") {
			return auth.ErrInvalid
		}
		seen[key] = true
		var data []byte
		if len(pair) == 2 {
			data, e = base64.StdEncoding.Strict().DecodeString(pair[1])
			if e != nil {
				return auth.ErrInvalid
			}
		}
		if key == "filename" {
			path = string(data)
		}
	}
	u, e := a.Files.Create(r.Context(), p.ID, r.PathValue("id"), path, size, c)
	if e != nil {
		return e
	}
	w.Header().Set("Location", "/api/v1/uploads/"+u.ID)
	w.Header().Set("Upload-Offset", "0")
	return reply(w, 201, map[string]any{"id": u.ID, "revision": u.Revision})
}
func (a *App) patchUpload(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	w.Header().Set("Tus-Resumable", "1.0.0")
	if r.Header.Get("Tus-Resumable") != "1.0.0" {
		w.Header().Set("Tus-Version", "1.0.0")
		w.WriteHeader(412)
		return nil
	}
	if r.Header.Get("Content-Type") != "application/offset+octet-stream" {
		w.WriteHeader(415)
		return nil
	}
	offset, e := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
	if e != nil || offset < 0 {
		return auth.ErrInvalid
	}
	r.Body = http.MaxBytesReader(w, r.Body, files.ChunkSize)
	b, e := io.ReadAll(r.Body)
	if e != nil {
		w.WriteHeader(413)
		return nil
	}
	next, e := a.Files.Append(r.Context(), p.ID, r.PathValue("id"), offset, b)
	if e != nil {
		return e
	}
	w.Header().Set("Upload-Offset", strconv.FormatInt(next, 10))
	w.WriteHeader(204)
	return nil
}
func (a *App) cancelUpload(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	if e := a.Files.Cancel(r.Context(), p.ID, r.PathValue("id")); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) renameFile(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Path string `json:"path"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	f, e := a.Files.Get(r.Context(), p.ID, r.PathValue("id"))
	if e != nil {
		return e
	}
	r.SetPathValue("id", f.TransferID)
	return a.repath(w, r, p, c, []pathMove{{ID: f.ID, Path: v.Path}}, "", false)
}

func (a *App) publish(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v struct {
		Expected   int64 `json:"expectedRevision"`
		TTL        int64 `json:"ttlSeconds"`
		Recipients []struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"recipients"`
		Common         bool   `json:"commonLink"`
		CommonPassword string `json:"commonPassword"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if len(v.Recipients) > 100 || (!v.Common && len(v.Recipients) == 0) {
		return auth.ErrInvalid
	}
	kid, key, e := a.Keys.Active(r.Context(), "configuration")
	if e != nil {
		return e
	}
	type prepared struct {
		id, email, hash, token string
	}
	preparedLinks := []prepared{}
	seen := map[string]bool{}
	for _, rcpt := range v.Recipients {
		email, e := auth.Email(rcpt.Email)
		if e != nil || seen[email] {
			return auth.ErrInvalid
		}
		seen[email] = true
		hash := ""
		if rcpt.Password != "" {
			hash, e = auth.LinkSecret(rcpt.Password)
			if e != nil {
				return e
			}
		}
		preparedLinks = append(preparedLinks, prepared{id: domain.NewID(), email: email, hash: hash, token: auth.Token()})
	}
	if v.Common {
		hash := ""
		if v.CommonPassword != "" {
			hash, e = auth.LinkSecret(v.CommonPassword)
			if e != nil {
				return e
			}
		}
		preparedLinks = append(preparedLinks, prepared{id: domain.NewID(), token: auth.Token(), hash: hash})
	}
	id := r.PathValue("id")
	input, e := json.Marshal(v)
	if e != nil {
		return e
	}
	mac := hmac.New(sha256.New, a.Key)
	mac.Write(input)
	fingerprint := hex.EncodeToString(mac.Sum(nil))
	if len(preparedLinks) > 0 {
		if e = a.Keys.Reserve(r.Context(), kid, len(preparedLinks)); e != nil {
			return e
		}
	}
	published, e := transfer.New(a.DB, c).PublishWithRequest(r.Context(), p.ID, id, v.Expected, v.TTL, fingerprint, func(tx pgx.Tx) error {
		if e := keyring.RequireActive(r.Context(), tx, kid); e != nil {
			return e
		}
		for _, link := range preparedLinks {
			envelope, e := cryptobox.Seal(key, "filex/link/v1/"+link.id, []byte(link.token))
			if e != nil {
				return e
			}
			_, e = tx.Exec(r.Context(), "INSERT INTO recipient_links(id,transfer_id,recipient_email,token_hash,token_envelope,password_hash,key_id) VALUES($1,$2,NULLIF($3,''),$4,$5,NULLIF($6,''),$7)", link.id, id, link.email, auth.Digest(link.token), envelope, link.hash, kid)
			if e != nil {
				return e
			}
		}
		return audit.Append(r.Context(), tx, a.Key, p.ID, "transfer.published", id)
	})
	if e != nil {
		return e
	}
	links, e := a.ownerLinks(r, id, p.ID, c)
	if e != nil {
		return e
	}
	return reply(w, 200, map[string]any{"transfer": published, "links": links})
}
func (a *App) ownerLinks(r *http.Request, id, owner string, c config.Settings) ([]map[string]any, error) {
	rows, e := a.DB.Query(r.Context(), "SELECT l.id::text,l.recipient_email,l.token_envelope,l.key_id::text FROM recipient_links l JOIN transfers t ON t.id=l.transfer_id WHERE t.id::text=$1 AND t.owner_id::text=$2 AND l.revoked_at IS NULL ORDER BY l.id", id, owner)
	if e != nil {
		return nil, e
	}
	type stored struct {
		id, keyid string
		email     *string
		envelope  []byte
	}
	all := []stored{}
	for rows.Next() {
		var v stored
		if e = rows.Scan(&v.id, &v.email, &v.envelope, &v.keyid); e != nil {
			rows.Close()
			return nil, e
		}
		all = append(all, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for _, v := range all {
		key, e := a.Keys.Get(r.Context(), v.keyid)
		if e != nil {
			return nil, e
		}
		token, e := cryptobox.Open(key, "filex/link/v1/"+v.id, v.envelope)
		if e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"id": v.id, "email": v.email, "url": c.PublicOrigin + "/receive#token=" + string(token)})
	}

	return out, nil
}
func (a *App) revoke(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	var id string
	if e = tx.QueryRow(r.Context(), "UPDATE transfers SET state='revoked',access_epoch=access_epoch+1 WHERE id::text=$1 AND owner_id=$2 RETURNING id::text", r.PathValue("id"), p.ID).Scan(&id); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "UPDATE files SET state='deleting',upload_epoch=upload_epoch+1 WHERE transfer_id=$1 AND state NOT IN ('deleted','cancelled')", id); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO deletion_jobs(provider_id,object_key,file_id) SELECT f.provider_id,ch.object_key,f.id FROM files f JOIN file_chunks ch ON ch.file_id=f.id WHERE f.transfer_id=$1 ON CONFLICT(object_key) DO NOTHING", id); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) VALUES($1,'transfer.revoked',$2,'{}') ON CONFLICT(deduplication_key) DO NOTHING", id, "revoked:"+id); e != nil {
		return e
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "transfer.revoked", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}

// Method override applies only to tus resources, before authentication and CSRF checks.
func tusProtocol(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resource := strings.HasPrefix(r.URL.Path, "/api/v1/uploads/") && !strings.HasSuffix(r.URL.Path, "/finalize")
		creation := strings.HasPrefix(r.URL.Path, "/api/v1/transfers/") && strings.HasSuffix(r.URL.Path, "/uploads")
		if resource || creation {
			w.Header().Set("Tus-Resumable", "1.0.0")
			if r.Method == "OPTIONS" {
				w.Header().Set("Tus-Version", "1.0.0")
				w.Header().Set("Tus-Extension", "creation,termination")
				w.Header().Set("Tus-Max-Size", "1099511627776")
				w.WriteHeader(204)
				return
			}
			if r.Header.Get("Tus-Resumable") != "1.0.0" {
				w.Header().Set("Tus-Version", "1.0.0")
				w.WriteHeader(412)
				return
			}
			if override := r.Header.Get("X-HTTP-Method-Override"); override != "" {
				if !resource || r.Method != "POST" || (override != "PATCH" && override != "DELETE") {
					failure(w, auth.ErrInvalid)
					return
				}
				r.Method = override
			}
		}
		next.ServeHTTP(w, r)
	})
}

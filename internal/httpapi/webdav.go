package httpapi

import (
	"context"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/transfer"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const davPrefix = "/api/v1/webdav/"
const davAllow = "OPTIONS, PROPFIND, GET, HEAD, PUT, MKCOL, MOVE, DELETE, PROPPATCH, LOCK, UNLOCK"

type davError int

func (e davError) Error() string { return http.StatusText(int(e)) }

type davAccess struct {
	app                    *App
	owner, transfer, token string
	credential             string
	paid                   bool
	tokens                 map[string]bool
	settings               config.Settings
}

func (a *App) webDAVRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/transfers/{id}/webdav-credentials", a.protect("transfers:write", false, a.createWebDAVCredential))
}

func (a *App) createWebDAVCredential(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	if p.Bearer || time.Since(p.Reauth) > 5*time.Minute {
		return auth.ErrForbidden
	}
	var v struct {
		TTL int64 `json:"ttlSeconds"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.TTL < 60 || v.TTL > 2592000 {
		return auth.ErrInvalid
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	var id string
	if e = tx.QueryRow(r.Context(), "SELECT id::text FROM transfers WHERE id::text=$1 AND owner_id=$2 AND state='draft' AND NOT frozen FOR UPDATE", r.PathValue("id"), p.ID).Scan(&id); e != nil {
		return e
	}
	if _, e = licensing.Guard(r.Context(), tx, "webdav"); e != nil {
		return e
	}
	token := auth.Token()
	var tokenID string
	var expires time.Time
	e = tx.QueryRow(r.Context(), "INSERT INTO api_tokens(id,user_id,token_hash,scopes,auth_epoch,expires_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,clock_timestamp()+make_interval(secs=>$5)) RETURNING id::text,expires_at", p.ID, auth.Digest(token), []string{"webdav:" + id}, p.Epoch, v.TTL).Scan(&tokenID, &expires)
	if e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 201, map[string]any{"id": tokenID, "url": c.PublicOrigin + davPrefix + id + "/", "username": "filex", "password": token, "expiresAt": expires})
}

// Dispatch before ServeMux path cleaning: invalid DAV names must not redirect.
func (a *App) protocols(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, davPrefix) {
			a.webDAV(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func davPath(u *url.URL, maxBytes int) (string, string, error) {
	if u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.User != nil {
		return "", "", davError(400)
	}
	escaped := strings.ToLower(u.EscapedPath())
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		return "", "", davError(400)
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, davPrefix), "/", 2)
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if !strings.HasPrefix(u.Path, davPrefix) || !domain.ValidID(parts[0]) {
		return "", "", davError(400)
	}
	path := strings.TrimSuffix(parts[1], "/")
	if path != "" {
		if _, e := domain.PathKey(path, maxBytes); e != nil {
			return "", "", davError(400)
		}
	}
	if parts[1] == "/" {
		return "", "", davError(400)
	}
	return parts[0], path, nil
}

func davTransport(r *http.Request, c config.Settings) bool {
	u, e := url.Parse(c.PublicOrigin)
	if e != nil {
		return false
	}
	if !strings.EqualFold(r.Host, u.Host) {
		return false
	}
	if u.Scheme == "https" {
		return r.TLS != nil || (c.NetworkPolicy.RequireProxy && c.NetworkPolicy.RequireHTTPS)
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
		return false
	}
	// The development Compose ingress binds loopback. Its internal proxy peer
	// is a container address, so never infer the client from untrusted XFF.
	return strings.EqualFold(r.Host, u.Host)
}

func (a *App) webDAV(w http.ResponseWriter, r *http.Request) {
	w = davResponse{w}
	w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV", charset="UTF-8"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Allow", davAllow)
	e := a.serveWebDAV(w, r)
	if e == nil {
		return
	}
	var status davError
	switch {
	case errors.Is(e, transfer.ErrLocked):
		http.Error(w, "Locked", http.StatusLocked)
	case errors.As(e, &status):
		http.Error(w, http.StatusText(int(status)), int(status))
	case errors.Is(e, auth.ErrCredentials):
		w.Header().Set("WWW-Authenticate", `Basic realm="WebDAV", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	default:
		failure(w, e)
	}
}

func (a *App) serveWebDAV(w http.ResponseWriter, r *http.Request) error {
	username, token, ok := r.BasicAuth()
	if !ok || username != "filex" || len(token) > 512 {
		return auth.ErrCredentials
	}
	id, path, e := davPath(r.URL, 4096)
	if e != nil {
		return e
	}
	r.SetPathValue("id", id)
	// Reuse the regular authentication, network and operational boundary.
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+token)
	var result error
	a.protect("webdav:"+id, false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		result = func() error {
			if !davTransport(r, c) || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != c.PublicOrigin) {
				return auth.ErrForbidden
			}
			if _, _, e := davPath(r.URL, c.MaxPathBytes); e != nil {
				return e
			}
			d := davAccess{app: a, owner: p.ID, transfer: id, token: token, settings: c, paid: r.Method == "PUT" || r.Method == "MOVE" || r.Method == "MKCOL" || r.Method == "PROPPATCH" || r.Method == "LOCK"}
			if e := d.check(r.Context()); e != nil {
				return e
			}
			w.Header().Set("DAV", "1, 2")
			w.Header().Set("MS-Author-Via", "DAV")
			d.tokens = map[string]bool{}
			if e := a.DB.QueryRow(r.Context(), "SELECT id::text FROM api_tokens WHERE token_hash=$1", auth.Digest(token)).Scan(&d.credential); e != nil {
				return e
			}
			r = r.WithContext(transfer.WithDAVGrant(r.Context(), d.credential, d.tokens))
			if e := d.conditions(r.Context(), a.DB, r, path); e != nil {
				if errors.Is(e, davError(304)) {
					w.WriteHeader(304)
					return nil
				}
				return e
			}
			switch r.Method {
			case "OPTIONS":
				w.WriteHeader(200)
				return nil
			case "PROPFIND":
				return d.propfind(w, r, path)
			case "PROPPATCH":
				return d.proppatch(w, r, path)
			case "GET", "HEAD":
				return d.read(w, r, path)
			case "LOCK":
				return d.lockResource(w, r, path)
			case "UNLOCK":
				return d.unlock(w, r, path)
			case "PUT":
				return d.put(w, r, path)
			case "MKCOL":
				return d.mkdir(w, r, path)
			case "DELETE", "MOVE":
				return d.change(w, r, path, p)
			case "COPY":
				return davError(403)
			default:
				return davError(405)
			}
		}()
		return nil
	})(w, r)
	// Authentication failures are written by protect; protocol failures by webDAV.
	return result
}

type davResponse struct{ http.ResponseWriter }

func (w davResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w davResponse) WriteHeader(status int) {
	if status != 401 {
		w.Header().Del("WWW-Authenticate")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (d davAccess) check(ctx context.Context) error {
	state, e := d.app.Operations.State(ctx)
	if e != nil {
		return e
	}
	if state.Mode != "open" {
		return maintenance.ErrClosed
	}
	var stateName string
	var frozen bool
	e = d.app.DB.QueryRow(ctx, "SELECT state,frozen OR EXISTS(SELECT 1 FROM files f WHERE f.transfer_id=t.id AND f.frozen) FROM transfers t WHERE id=$1 AND owner_id=$2", d.transfer, d.owner).Scan(&stateName, &frozen)
	if e != nil {
		return e
	}
	if stateName != "draft" || frozen {
		return auth.ErrForbidden
	}
	return nil
}

func (d davAccess) guard(ctx context.Context, tx pgx.Tx) error {
	if d.paid {
		if _, e := licensing.Guard(ctx, tx, "webdav"); e != nil {
			return e
		}
	}
	var id string
	e := tx.QueryRow(ctx, `SELECT t.id::text FROM api_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=$1 AND t.user_id=$2 AND t.revoked_at IS NULL AND t.expires_at>clock_timestamp() AND t.auth_epoch=u.auth_epoch AND NOT u.disabled AND $3=ANY(t.scopes) FOR SHARE OF t,u`, auth.Digest(d.token), d.owner, "webdav:"+d.transfer).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return auth.ErrCredentials
	}
	if e != nil {
		return e
	}
	var allowed bool
	e = tx.QueryRow(ctx, "SELECT state='draft' AND NOT frozen AND NOT EXISTS(SELECT 1 FROM files f WHERE f.transfer_id=t.id AND f.frozen) FROM transfers t WHERE id=$1 AND owner_id=$2", d.transfer, d.owner).Scan(&allowed)
	if e != nil {
		return e
	}
	if !allowed {
		return auth.ErrForbidden
	}
	return nil
}

func davFileETag(id, state string, offset int64, hash *string) string {
	if hash != nil && state == "ready" {
		return `"` + id + ":" + *hash + `"`
	}
	return fmt.Sprintf(`W/"%s:%s:%d"`, id, state, offset)
}

func davHref(id, path string, directory bool) string {
	value := davPrefix + id + "/" + path
	if directory && !strings.HasSuffix(value, "/") {
		value += "/"
	}
	return (&url.URL{Path: value}).EscapedPath()
}

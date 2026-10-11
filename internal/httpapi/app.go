package httpapi

import (
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/settings"
	"filex.local/filex/internal/storage"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type App struct {
	DB          *pgxpool.Pool
	Auth        *auth.Store
	Key         []byte
	requests    atomic.Uint64
	streamSlots chan struct{}
	Files       files.Service
	Keys        keyring.Ring
}

func New(db *pgxpool.Pool, key []byte) (*App, error) {
	auditKey, e := hkdf.Key(sha256.New, key, nil, "filex/audit/key/v1", 32)
	if e != nil {
		return nil, e
	}
	a, e := auth.New(db, auditKey)
	if e != nil {
		return nil, e
	}
	ring := keyring.Ring{DB: db, KEK: key}
	return &App{streamSlots: make(chan struct{}, 8), DB: db, Auth: a, Key: auditKey, Keys: ring, Files: files.Service{DB: db, Keys: ring, Storage: storage.Registry{DB: db, Keys: ring}}}, nil
}

type action func(http.ResponseWriter, *http.Request, auth.Principal, config.Settings) error

func reply(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}
func body(w http.ResponseWriter, r *http.Request, v any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return auth.ErrInvalid
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return auth.ErrInvalid
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return auth.ErrInvalid
	}
	return nil
}
func failure(w http.ResponseWriter, e error) {
	code, name := 500, "internal_error"
	switch {
	case errors.Is(e, auth.ErrCredentials):
		code, name = 401, "unauthenticated"
	case errors.Is(e, auth.ErrForbidden):
		code, name = 403, "forbidden"
	case errors.Is(e, auth.ErrRate):
		code, name = 429, "rate_limited"
	case errors.Is(e, transfer.ErrNotFound):
		code, name = 404, "not_found"
	case errors.Is(e, transfer.ErrConflict) || errors.Is(e, files.ErrOffset) || errors.Is(e, keyring.ErrChanged):
		code, name = 409, "conflict"
	case errors.Is(e, domain.ErrInvalid):
		code, name = 400, "invalid_input"
	case errors.Is(e, auth.ErrInvalid):
		code, name = 400, "invalid_input"
	case errors.Is(e, pgx.ErrNoRows):
		code, name = 404, "not_found"
	}
	if code == 429 {
		w.Header().Set("Retry-After", "60")
	}
	_ = reply(w, code, map[string]string{"code": name, "message": name})
}
func (a *App) protect(scope string, admin bool, fn action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.requests.Add(1)
		c, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			failure(w, e)
			return
		}
		token := ""
		bearer := false
		header := r.Header.Get("Authorization")
		if header != "" {
			if !strings.HasPrefix(header, "Bearer ") {
				failure(w, auth.ErrCredentials)
				return
			}
			token = strings.TrimPrefix(header, "Bearer ")
			bearer = true
		} else if cookie, e := r.Cookie("filexSession"); e == nil {
			token = cookie.Value
		}
		p, e := a.Auth.Authenticate(r.Context(), token, bearer)
		if e != nil {
			failure(w, e)
			return
		}
		if !p.Allows(scope) || (admin && (p.Role != "admin" || p.Bearer || time.Since(p.Reauth) > 5*time.Minute)) {
			failure(w, auth.ErrForbidden)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && !bearer {
			if r.Header.Get("Origin") != c.PublicOrigin || !p.CheckCSRF(r.Header.Get("X-CSRF-Token")) {
				failure(w, auth.ErrForbidden)
				return
			}
		}
		if e = fn(w, r, p, c); e != nil {
			failure(w, e)
		}
	}
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/status", Public(a.DB))
	mux.HandleFunc("GET /api/v1/banner", func(w http.ResponseWriter, r *http.Request) {
		c, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			failure(w, e)
			return
		}
		_ = reply(w, 200, map[string]string{"ru": c.BannerRU, "en": c.BannerEN})
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		c, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			failure(w, e)
			return
		}
		if r.Header.Get("Origin") != c.PublicOrigin {
			failure(w, auth.ErrForbidden)
			return
		}
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if e = body(w, r, &input); e != nil {
			failure(w, e)
			return
		}
		remote, _, e := net.SplitHostPort(r.RemoteAddr)
		if e != nil {
			remote = r.RemoteAddr
		}
		session, e := a.Auth.Login(r.Context(), input.Email, input.Password, remote, c.SessionTTLSeconds)
		if e != nil {
			failure(w, e)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "filexSession", Value: session.Token, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteStrictMode, Expires: session.Expires, MaxAge: int(c.SessionTTLSeconds)})
		_ = reply(w, 200, session)
	})
	mux.HandleFunc("GET /api/v1/auth/me", a.protect("account:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		return reply(w, 200, p)
	}))
	mux.HandleFunc("POST /api/v1/auth/logout", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if e := a.Auth.Logout(r.Context(), p, p.SessionID); e != nil {
			return e
		}
		http.SetCookie(w, &http.Cookie{Name: "filexSession", Path: "/", MaxAge: -1, HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteStrictMode})
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("POST /api/v1/auth/reauth", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Password string `json:"password"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Auth.Reauthenticate(r.Context(), p, v.Password); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("PATCH /api/v1/auth/password", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Current string `json:"currentPassword"`
			New     string `json:"newPassword"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		if e := a.Auth.ChangePassword(r.Context(), p, v.Current, v.New); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("GET /api/v1/auth/sessions", a.protect("account:read", false, a.sessions))
	mux.HandleFunc("DELETE /api/v1/auth/sessions/{id}", a.protect("account:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if e := a.Auth.Logout(r.Context(), p, r.PathValue("id")); e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("GET /api/v1/auth/history", a.protect("account:read", false, a.loginHistory))
	mux.HandleFunc("GET /api/v1/auth/tokens", a.protect("account:read", false, a.listTokens))
	mux.HandleFunc("POST /api/v1/auth/tokens", a.protect("account:write", false, a.createToken))
	mux.HandleFunc("DELETE /api/v1/auth/tokens/{id}", a.protect("account:write", false, a.revokeToken))
	mux.HandleFunc("POST /api/v1/admin/users", a.protect("admin:users", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		id, e := a.Auth.CreateUser(r.Context(), p.ID, v.Email, v.Password, v.Role)
		if e != nil {
			return e
		}
		return reply(w, 201, map[string]string{"id": id})
	}))
	mux.HandleFunc("GET /api/v1/admin/settings", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		items, e := (settings.Store{DB: a.DB, Key: a.Key}).List(r.Context())
		if e != nil {
			return e
		}
		return reply(w, 200, items)
	}))
	mux.HandleFunc("POST /api/v1/admin/settings", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var raw json.RawMessage
		if e := body(w, r, &raw); e != nil {
			return e
		}
		id, e := (settings.Store{DB: a.DB, Key: a.Key}).Draft(r.Context(), p.ID, raw)
		if e != nil {
			return auth.ErrInvalid
		}
		return reply(w, 201, map[string]int64{"revision": id})
	}))
	mux.HandleFunc("POST /api/v1/admin/settings/{revision}/validate", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		id, e := strconv.ParseInt(r.PathValue("revision"), 10, 64)
		if e != nil {
			return auth.ErrInvalid
		}
		if e = (settings.Store{DB: a.DB, Key: a.Key}).Validate(r.Context(), id); e != nil {
			return auth.ErrInvalid
		}
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("POST /api/v1/admin/settings/{revision}/apply", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var v struct {
			Expected int64 `json:"expectedRevision"`
		}
		if e := body(w, r, &v); e != nil {
			return e
		}
		id, e := strconv.ParseInt(r.PathValue("revision"), 10, 64)
		if e != nil {
			return auth.ErrInvalid
		}
		if e = (settings.Store{DB: a.DB, Key: a.Key}).Apply(r.Context(), p.ID, id, v.Expected); e != nil {
			return auth.ErrInvalid
		}
		w.WriteHeader(204)
		return nil
	}))
	mux.HandleFunc("POST /api/v1/admin/audit/verify", a.protect("admin:audit", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if e := audit.Verify(r.Context(), a.DB, a.Key); e != nil {
			return e
		}
		return reply(w, 200, map[string]string{"status": "verified"})
	}))
	a.treeRoutes(mux)
	a.organizationRoutes(mux)
	a.exportRoutes(mux)
	a.reportRoutes(mux)
	a.communicationsRoutes(mux)
	a.maintenanceRoutes(mux)
	a.adminFileRoutes(mux)
	a.transferRoutes(mux)
	a.recipientRoutes(mux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = reply(w, 404, map[string]string{"code": "not_found", "message": "not_found"})
	})
	return headers(a.limitStreams(tusProtocol(mux)))
}
func (a *App) Metrics() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", Internal(a.DB))
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = io.WriteString(w, "# TYPE filex_authenticated_requests_total counter\nfilex_authenticated_requests_total "+strconv.FormatUint(a.requests.Load(), 10)+"\n")
	})
	return mux
}
func (a *App) sessions(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,created_at,expires_at,revoked_at FROM web_sessions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 100", p.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	type item struct {
		ID      string     `json:"id"`
		Created time.Time  `json:"createdAt"`
		Expires time.Time  `json:"expiresAt"`
		Revoked *time.Time `json:"revokedAt"`
	}
	items := []item{}
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.ID, &v.Created, &v.Expires, &v.Revoked); e != nil {
			return e
		}
		items = append(items, v)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, items)
}
func (a *App) loginHistory(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT occurred_at,success FROM login_history WHERE user_id=$1 ORDER BY id DESC LIMIT 100", p.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	type item struct {
		At      time.Time `json:"occurredAt"`
		Success bool      `json:"success"`
	}
	items := []item{}
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.At, &v.Success); e != nil {
			return e
		}
		items = append(items, v)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, items)
}
func (a *App) createToken(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	if p.Bearer || time.Since(p.Reauth) > 5*time.Minute {
		return auth.ErrForbidden
	}
	var v struct {
		Scopes []string `json:"scopes"`
		TTL    int64    `json:"ttlSeconds"`
	}
	if e := body(w, r, &v); e != nil {
		return e
	}
	allowed := map[string]bool{"account:read": true, "transfers:read": true, "transfers:write": true, "reports:read": true, "reports:write": true, "exports:write": true, "notifications:read": true, "notifications:write": true, "webhooks:write": true}
	if v.TTL < 60 || v.TTL > 2592000 || len(v.Scopes) == 0 {
		return auth.ErrInvalid
	}
	for _, scope := range v.Scopes {
		if !allowed[scope] {
			return auth.ErrInvalid
		}
	}
	token := auth.Token()
	var id string
	e := a.DB.QueryRow(r.Context(), "INSERT INTO api_tokens(id,user_id,token_hash,scopes,auth_epoch,expires_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,clock_timestamp()+make_interval(secs=>$5)) RETURNING id::text", p.ID, auth.Digest(token), v.Scopes, p.Epoch, v.TTL).Scan(&id)
	if e != nil {
		return e
	}
	return reply(w, 201, map[string]string{"id": id, "token": token})
}
func (a *App) revokeToken(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	if p.Bearer {
		return auth.ErrForbidden
	}
	_, e := a.DB.Exec(r.Context(), "UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE id::text=$1 AND user_id=$2", r.PathValue("id"), p.ID)
	if e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}

func (a *App) listTokens(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,scopes,expires_at,revoked_at FROM api_tokens WHERE user_id=$1 ORDER BY expires_at DESC LIMIT 100", p.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id string
		var scopes []string
		var expires time.Time
		var revoked *time.Time
		if e = rows.Scan(&id, &scopes, &expires, &revoked); e != nil {
			return e
		}
		out = append(out, map[string]any{"id": id, "scopes": scopes, "expiresAt": expires, "revokedAt": revoked})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, out)
}

func (a *App) limitStreams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		heavy := strings.HasSuffix(r.URL.Path, "/content") || strings.Contains(r.URL.Path, "/forensic") || strings.HasSuffix(r.URL.Path, "/finalize") || ((r.Method == "PATCH" || r.Method == "POST") && strings.HasPrefix(r.URL.Path, "/api/v1/uploads/"))
		if heavy {
			select {
			case a.streamSlots <- struct{}{}:
				defer func() { <-a.streamSlots }()
			default:
				failure(w, auth.ErrRate)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

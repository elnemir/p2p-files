package issuer

import (
	"encoding/json"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"filex.local/filex/internal/setup"
	"io"
	"net/http"
)

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /issuer/api/setup", func(w http.ResponseWriter, r *http.Request) { setup.Reply(w, 200, "complete") })
	m.HandleFunc("POST /issuer/api/setup", func(w http.ResponseWriter, r *http.Request) { setup.Reply(w, 403, "disabled") })
	m.HandleFunc("GET /issuer/auth/login", s.login)
	m.HandleFunc("GET /issuer/auth/callback", s.callback)
	m.HandleFunc("GET /issuer/api/session", func(w http.ResponseWriter, r *http.Request) {
		p, e := s.operator(r, false)
		if e != nil {
			reject(w, 401)
			return
		}
		_ = json.NewEncoder(w).Encode(p)
	})
	m.HandleFunc("GET /issuer/api/issuances", func(w http.ResponseWriter, r *http.Request) {
		if _, e := s.operator(r, false); e != nil {
			reject(w, 401)
			return
		}
		v, e := List(r.Context(), s.DB)
		if e != nil {
			reject(w, 500)
			return
		}
		_ = json.NewEncoder(w).Encode(v)
	})
	m.HandleFunc("POST /issuer/api/issuances", func(w http.ResponseWriter, r *http.Request) {
		p, e := s.operator(r, true)
		if e != nil || p.Role == "viewer" {
			reject(w, 403)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, licensing.MaxBytes)
		raw, e := io.ReadAll(r.Body)
		var in Input
		if e != nil || licensing.Decode(raw, &in) != nil {
			reject(w, 400)
			return
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || len(fields) != 2 {
			reject(w, 400)
			return
		}
		if _, e = licensing.ParseClaims(fields["claims"]); e != nil {
			reject(w, 400)
			return
		}
		v, e := Issue(r.Context(), s.DB, p.Subject, in)
		if errors.Is(e, licensing.ErrConflict) {
			reject(w, 409)
			return
		}
		if errors.Is(e, licensing.ErrInvalid) {
			reject(w, 400)
			return
		}
		if e != nil {
			reject(w, 500)
			return
		}
		_ = json.NewEncoder(w).Encode(v)
	})
	m.HandleFunc("GET /issuer/api/issuances/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, e := s.operator(r, false); e != nil {
			reject(w, 401)
			return
		}
		id := r.PathValue("id")
		if !domain.ValidID(id) {
			reject(w, 400)
			return
		}
		var envelope string
		if e := s.DB.QueryRow(r.Context(), "SELECT envelope FROM issuer_issuances WHERE id=$1 AND state='issued'", id).Scan(&envelope); e != nil {
			reject(w, 409)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="filex-`+id+`.jws"`)
		_, _ = io.WriteString(w, envelope)
	})
	m.HandleFunc("POST /issuer/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if p, e := s.operator(r, false); e != nil || r.Header.Get("Origin") != s.config.Origin || r.Header.Get("X-CSRF-Token") != p.CSRF {
			reject(w, 403)
			return
		}
		c, e := r.Cookie("issuerSession")
		if e != nil {
			reject(w, 401)
			return
		}
		if _, e = s.DB.Exec(r.Context(), "DELETE FROM issuer_sessions WHERE token_hash=$1", auth.Digest(c.Value)); e != nil {
			reject(w, 500)
			return
		}
		cookie(w, "issuerSession", "", -1)
		_ = json.NewEncoder(w).Encode(map[string]bool{"loggedOut": true})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		m.ServeHTTP(w, r)
	})
}

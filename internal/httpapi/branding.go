package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/branding"
	"filex.local/filex/internal/config"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (a *App) publicBranding(w http.ResponseWriter, r *http.Request) {
	rev, e := branding.Current(r.Context(), a.DB)
	if e != nil {
		failure(w, e)
		return
	}
	raw, e := json.Marshal(rev)
	if e != nil {
		failure(w, e)
		return
	}
	tag := fmt.Sprintf(`"%x"`, sha256.Sum256(raw))
	w.Header().Set("ETag", tag)
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("If-None-Match") == tag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}
func (a *App) brandingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/branding", a.publicBranding)
	mux.HandleFunc("GET /api/v1/admin/branding", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rev, e := branding.Current(r.Context(), a.DB)
		if e != nil {
			return e
		}
		return reply(w, 200, rev)
	}))
	mux.HandleFunc("PUT /api/v1/admin/branding", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			return auth.ErrInvalid
		}
		var input struct {
			Expected *int64           `json:"expectedRevision"`
			Value    *branding.Config `json:"value"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, branding.MaxRequestBytes)
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if e := d.Decode(&input); e != nil {
			return auth.ErrInvalid
		}
		if e := d.Decode(new(any)); e != io.EOF || input.Expected == nil || input.Value == nil {
			return auth.ErrInvalid
		}
		rev, e := (branding.Store{DB: a.DB, Key: a.Key}).Apply(r.Context(), p, *input.Expected, *input.Value)
		if e != nil {
			return e
		}
		return reply(w, 200, rev)
	}))
}

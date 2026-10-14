package httpapi

import (
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/licensing"
	"net/http"
)

func (a *App) licenseRoutes(m *http.ServeMux) {
	store := licensing.Store{DB: a.DB, AuditKey: a.Key}
	m.HandleFunc("GET /api/v1/admin/license", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		v, e := store.Status(r.Context())
		if e != nil {
			return e
		}
		return reply(w, 200, v)
	}))
	m.HandleFunc("POST /api/v1/admin/license/preview", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var in struct {
			Envelope string `json:"envelope"`
		}
		if e := body(w, r, &in); e != nil {
			return e
		}
		v, e := store.Preview(r.Context(), in.Envelope)
		if e != nil {
			return e
		}
		return reply(w, 200, v)
	}))
	m.HandleFunc("PUT /api/v1/admin/license", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var in struct {
			Envelope string `json:"envelope"`
			Epoch    *int64 `json:"expectedEpoch"`
		}
		if e := body(w, r, &in); e != nil {
			return e
		}
		if in.Epoch == nil {
			return auth.ErrInvalid
		}
		v, e := store.Apply(r.Context(), in.Envelope, *in.Epoch, p.ID, p.SessionID, p.Epoch)
		if e != nil {
			return e
		}
		return reply(w, 200, v)
	}))
}

package httpapi

import (
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/setup"
	"net/http"
	"os"
)

func (a *App) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/setup", func(w http.ResponseWriter, r *http.Request) {
		var complete bool
		if e := a.DB.QueryRow(r.Context(), "SELECT completed OR EXISTS(SELECT 1 FROM users) FROM ops_setup WHERE singleton").Scan(&complete); e != nil {
			setup.Reply(w, 503, "unavailable")
			return
		}
		state := "complete"
		if !complete {
			c, e := database.Settings(r.Context(), a.DB)
			if e != nil {
				setup.Reply(w, 503, "unavailable")
				return
			}
			state = "disabled"
			if setup.FromEnv("FILEX_", os.Getenv, c.PublicOrigin).Enabled() {
				state = "pending"
			}
		}
		setup.Reply(w, 200, state)
	})
	mux.HandleFunc("POST /api/v1/setup", func(w http.ResponseWriter, r *http.Request) {
		c, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			setup.Reply(w, 503, "unavailable")
			return
		}
		if !setup.FromEnv("FILEX_", os.Getenv, c.PublicOrigin).Allowed(r) {
			setup.Reply(w, 403, "disabled")
			return
		}
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if setup.Decode(w, r, &in) != nil {
			setup.Reply(w, 400, "invalid_input")
			return
		}
		unlock, e := maintenance.StartupShared(r.Context(), a.DB)
		if e != nil {
			setup.Reply(w, 503, "unavailable")
			return
		}
		defer unlock()
		if _, e = a.Auth.SetupAdmin(r.Context(), in.Email, in.Password); e != nil {
			code := 503
			if errors.Is(e, auth.ErrInvalid) {
				code = 400
			}
			if errors.Is(e, auth.ErrConflict) {
				code = 409
			}
			setup.Reply(w, code, "setup_failed")
			return
		}
		setup.Reply(w, 200, "complete")
	})
}

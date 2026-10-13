package httpapi

import (
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/identity"
	"filex.local/filex/internal/maintenance"
	"net/http"
	"strings"
	"time"
)

func (a *App) identities() identity.Store {
	return identity.Store{DB: a.DB, Keys: a.Keys, AuditKey: a.Key}
}
func (a *App) externalRequest(r *http.Request, originRequired bool) (config.Settings, error) {
	c, e := database.Settings(r.Context(), a.DB)
	if e != nil {
		return c, e
	}
	if originRequired && r.Header.Get("Origin") != c.PublicOrigin {
		return c, auth.ErrForbidden
	}
	remote, ok := r.Context().Value(clientAddressKey{}).(string)
	if !ok || remote == "" {
		return c, auth.ErrForbidden
	}
	if e = a.Auth.ThrottleExternal(r.Context(), remote, r.PathValue("id")+":"+remote); e != nil {
		return c, e
	}
	return c, nil
}
func (a *App) externalActor(r *http.Request, purpose string) (auth.Principal, error) {
	if purpose == "login" {
		return auth.Principal{}, nil
	}
	if purpose != "link" && purpose != "reauth" {
		return auth.Principal{}, auth.ErrInvalid
	}
	if r.Header.Get("Authorization") != "" {
		return auth.Principal{}, auth.ErrForbidden
	}
	cookie, e := r.Cookie("filexSession")
	if e != nil {
		return auth.Principal{}, auth.ErrCredentials
	}
	p, e := a.Auth.Authenticate(r.Context(), cookie.Value, false)
	if e != nil {
		return p, e
	}
	if !p.CheckCSRF(r.Header.Get("X-CSRF-Token")) || (purpose == "link" && time.Since(p.Reauth) > 5*time.Minute) {
		return p, auth.ErrForbidden
	}
	return p, nil
}
func (a *App) externalSession(w http.ResponseWriter, r *http.Request, c config.Settings, session auth.Session) error {
	state, e := a.Operations.State(r.Context())
	if e != nil {
		return e
	}
	if state.Mode != "open" && session.User.Role != "admin" {
		if e = a.Auth.Logout(r.Context(), session.User, session.User.SessionID); e != nil {
			return e
		}
		return maintenance.ErrClosed
	}
	http.SetCookie(w, &http.Cookie{Name: "filexSession", Value: session.Token, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteStrictMode, Expires: session.Expires, MaxAge: int(c.SessionTTLSeconds)})
	return reply(w, 200, session)
}
func (a *App) identityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/providers", func(w http.ResponseWriter, r *http.Request) {
		list, e := a.identities().List(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		result := []identity.Public{}
		for _, p := range list {
			if p.Config.Enabled {
				result = append(result, identity.Public{ID: p.ID, Name: p.Config.Name, Protocol: p.Config.Protocol, AllowRegistration: p.Config.AllowRegistration})
			}
		}
		_ = reply(w, 200, result)
	})
	mux.HandleFunc("GET /api/v1/admin/identity-providers", a.protect("admin:settings", true, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		list, e := a.identities().List(r.Context())
		if e != nil {
			return e
		}
		return reply(w, 200, list)
	}))
	apply := func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		var in identity.Input
		if e := body(w, r, &in); e != nil {
			return e
		}
		out, e := a.identities().Apply(r.Context(), p, r.PathValue("id"), in)
		if e != nil {
			return e
		}
		code := 200
		if r.Method == "POST" {
			code = 201
		}
		return reply(w, code, out)
	}
	mux.HandleFunc("POST /api/v1/admin/identity-providers", a.protect("admin:settings", true, apply))
	mux.HandleFunc("PUT /api/v1/admin/identity-providers/{id}", a.protect("admin:settings", true, apply))
	mux.HandleFunc("POST /api/v1/auth/providers/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		c, e := a.externalRequest(r, true)
		if e != nil {
			failure(w, e)
			return
		}
		var in struct {
			Purpose string `json:"purpose"`
		}
		if e = body(w, r, &in); e != nil {
			failure(w, e)
			return
		}
		actor, e := a.externalActor(r, in.Purpose)
		if e != nil {
			failure(w, e)
			return
		}
		store := a.identities()
		p, e := store.Get(r.Context(), r.PathValue("id"))
		if e != nil || !p.Config.Enabled || p.Config.Protocol == "ldap" {
			failure(w, auth.ErrCredentials)
			return
		}
		f, e := identity.NewFlow(p, c.PublicOrigin, in.Purpose, actor)
		if e != nil {
			failure(w, e)
			return
		}
		state, binding := auth.Token(), auth.Token()
		target := ""
		if p.Config.Protocol == "oidc" {
			target, e = p.OIDCStart(r.Context(), state, f)
		} else {
			target, e = p.SAMLStart(state, &f)
		}
		if e != nil {
			failure(w, e)
			return
		}
		if e = store.SaveFlow(r.Context(), state, binding, f); e != nil {
			failure(w, e)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "filexLogin", Value: binding, Path: "/api/v1/auth", HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 300})
		_ = reply(w, 200, map[string]string{"url": target})
	})
	mux.HandleFunc("GET /api/v1/auth/providers/{id}/callback", func(w http.ResponseWriter, r *http.Request) { a.externalCallback(w, r, false) })
	mux.HandleFunc("POST /api/v1/auth/providers/{id}/saml", func(w http.ResponseWriter, r *http.Request) { a.externalCallback(w, r, true) })
	mux.HandleFunc("GET /api/v1/auth/providers/{id}/metadata", func(w http.ResponseWriter, r *http.Request) {
		c, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			failure(w, e)
			return
		}
		p, e := a.identities().Get(r.Context(), r.PathValue("id"))
		if e != nil || p.Config.Protocol != "saml" {
			failure(w, auth.ErrInvalid)
			return
		}
		raw, e := p.SAMLMetadata(c.PublicOrigin)
		if e != nil {
			failure(w, e)
			return
		}
		w.Header().Set("Content-Type", "application/samlmetadata+xml")
		_, _ = w.Write(raw)
	})
	mux.HandleFunc("POST /api/v1/auth/complete", func(w http.ResponseWriter, r *http.Request) {
		c, e := a.externalRequest(r, true)
		if e != nil {
			failure(w, e)
			return
		}
		var in struct {
			State string `json:"state"`
		}
		if e = body(w, r, &in); e != nil {
			failure(w, e)
			return
		}
		cookie, e := r.Cookie("filexLogin")
		if e != nil {
			failure(w, auth.ErrCredentials)
			return
		}
		session, e := a.identities().Complete(r.Context(), in.State, cookie.Value, c.PublicOrigin, c.SessionTTLSeconds)
		if e != nil {
			failure(w, e)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "filexLogin", Path: "/api/v1/auth", HttpOnly: true, Secure: strings.HasPrefix(c.PublicOrigin, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: -1})
		if e = a.externalSession(w, r, c, session); e != nil {
			failure(w, e)
		}
	})
	mux.HandleFunc("POST /api/v1/auth/providers/{id}/ldap", func(w http.ResponseWriter, r *http.Request) {
		c, e := a.externalRequest(r, true)
		if e != nil {
			failure(w, e)
			return
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Purpose  string `json:"purpose"`
		}
		if e = body(w, r, &in); e != nil {
			failure(w, e)
			return
		}
		if e = a.Auth.ThrottleExternalSubject(r.Context(), r.PathValue("id")+":"+strings.ToLower(in.Username)); e != nil {
			failure(w, e)
			return
		}
		actor, e := a.externalActor(r, in.Purpose)
		if e != nil {
			failure(w, e)
			return
		}
		p, e := a.identities().Get(r.Context(), r.PathValue("id"))
		if e != nil || !p.Config.Enabled || p.Config.Protocol != "ldap" {
			failure(w, auth.ErrCredentials)
			return
		}
		f, e := identity.NewFlow(p, c.PublicOrigin, in.Purpose, actor)
		if e != nil {
			failure(w, e)
			return
		}
		f.Proof, e = p.LDAPVerify(r.Context(), in.Username, in.Password)
		if e != nil {
			failure(w, e)
			return
		}
		session, e := a.identities().LDAPSession(r.Context(), f, c.SessionTTLSeconds)
		if e != nil {
			failure(w, e)
			return
		}
		if e = a.externalSession(w, r, c, session); e != nil {
			failure(w, e)
		}
	})
	mux.HandleFunc("GET /api/v1/auth/identities", a.protect("account:read", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		rows, e := a.DB.Query(r.Context(), `SELECT p.id::text,p.config->>'name',p.config->>'protocol',(p.config->>'enabled')::boolean FROM external_identities i JOIN identity_providers p ON p.id=i.provider_id WHERE i.user_id=$1 ORDER BY p.created_at`, p.ID)
		if e != nil {
			return e
		}
		defer rows.Close()
		type item struct {
			ProviderID string `json:"providerId"`
			Name       string `json:"name"`
			Protocol   string `json:"protocol"`
			Enabled    bool   `json:"enabled"`
		}
		out := []item{}
		for rows.Next() {
			var v item
			if e = rows.Scan(&v.ProviderID, &v.Name, &v.Protocol, &v.Enabled); e != nil {
				return e
			}
			out = append(out, v)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return reply(w, 200, out)
	}))
}
func (a *App) externalCallback(w http.ResponseWriter, r *http.Request, isSAML bool) {
	c, e := a.externalRequest(r, false)
	if e != nil {
		failure(w, e)
		return
	}
	state, code := "", ""
	if isSAML {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			failure(w, auth.ErrInvalid)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 3<<20)
		if e = r.ParseForm(); e != nil || len(r.PostForm["RelayState"]) != 1 {
			failure(w, auth.ErrInvalid)
			return
		}
		state = r.PostForm.Get("RelayState")
	} else {
		state, code, e = identity.CallbackQuery(r.URL.Query())
		if e != nil {
			failure(w, e)
			return
		}
	}
	store := a.identities()
	f, e := store.Flow(r.Context(), state)
	if e != nil || f.ProviderID != r.PathValue("id") || f.Origin != c.PublicOrigin {
		failure(w, auth.ErrCredentials)
		return
	}
	p, e := store.Get(r.Context(), f.ProviderID)
	if e != nil || !p.Config.Enabled || p.Revision != f.Revision || (isSAML && p.Config.Protocol != "saml") || (!isSAML && p.Config.Protocol != "oidc") {
		failure(w, auth.ErrCredentials)
		return
	}
	if !isSAML && (len(r.URL.Query()["iss"]) > 1 || (r.URL.Query().Get("iss") != "" && r.URL.Query().Get("iss") != p.Config.Issuer)) {
		failure(w, auth.ErrCredentials)
		return
	}
	if isSAML {
		f.Proof, e = p.SAMLVerify(r, f)
	} else {
		f.Proof, e = p.OIDCVerify(r.Context(), code, f)
	}
	if e != nil {
		failure(w, e)
		return
	}
	if e = store.VerifyFlow(r.Context(), state, f); e != nil {
		failure(w, e)
		return
	}
	http.Redirect(w, r, c.PublicOrigin+"/?authFlow="+state, http.StatusSeeOther)
}

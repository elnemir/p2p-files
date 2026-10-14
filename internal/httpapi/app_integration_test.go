package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func integrationApp(t *testing.T) (*App, *pgxpool.Pool) {
	return integrationAppWithKEK(t, bytes.Repeat([]byte{1}, 32))
}
func integrationAppWithKEK(t *testing.T, kek []byte) (*App, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("FILEX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires FILEX_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := "api_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		p.Close()
		if _, e := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
		if e := admin.Close(ctx); e != nil {
			t.Error(e)
		}
	})
	if e = database.Migrate(ctx, p); e != nil {
		t.Fatal(e)
	}
	installTestLicense(t, p)
	a, e := New(p, kek)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(a.Close)
	return a, p
}
func call(h http.Handler, method, path, body string, cookie *http.Cookie, csrf, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:1234"
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	r.Header.Set("Origin", origin)
	r.Header.Set("X-CSRF-Token", csrf)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestIntegrationIdentityAndSettings(t *testing.T) {
	a, p := integrationApp(t)
	ctx := context.Background()
	origin := "http://localhost:8090"
	h := a.Handler()
	if _, e := a.Auth.CreateUser(ctx, "host_cli", "admin@example.test", "strong-password-0123", "admin"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Auth.CreateUser(ctx, "host_cli", "user@example.test", "strong-password-0123", "user"); e != nil {
		t.Fatal(e)
	}
	for _, email := range []string{"unknown@example.test", "admin@example.test"} {
		w := call(h, "POST", "/api/v1/auth/login", `{"email":"`+email+`","password":"incorrect-password"}`, nil, "", origin)
		if w.Code != 401 || strings.Contains(w.Body.String(), email) {
			t.Fatal("enumeration or login bypass", w.Code, w.Body.String())
		}
	}
	w := call(h, "POST", "/api/v1/auth/login", `{"email":"admin@example.test","password":"strong-password-0123"}`, nil, "", "https://evil.test")
	if w.Code != 403 {
		t.Fatal("login CSRF bypass")
	}
	w = call(h, "POST", "/api/v1/auth/login", `{"email":"admin@example.test","password":"strong-password-0123"}`, nil, "", origin)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var login struct {
		CSRF string `json:"csrfToken"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &login); e != nil {
		t.Fatal(e)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	w = call(h, "POST", "/api/v1/auth/logout", "", cookie, "", origin)
	if w.Code != 403 {
		t.Fatal("CSRF bypass")
	}
	w = call(h, "GET", "/api/v1/auth/me", "", cookie, "", origin)
	if w.Code != 200 {
		t.Fatal("session invalid")
	}
	raw := `{"version":1,"transferMaxTtlSeconds":60,"maxFiles":20,"maxPathBytes":1024,"publicOrigin":"http://localhost:8090","sessionTtlSeconds":600}`
	w = call(h, "POST", "/api/v1/admin/settings", raw, cookie, login.CSRF, origin)
	if w.Code != 201 {
		t.Fatal("draft settings", w.Code, w.Body.String())
	}
	w = call(h, "POST", "/api/v1/admin/settings/2/apply", `{"expectedRevision":1}`, cookie, login.CSRF, origin)
	if w.Code != 400 {
		t.Fatal("unvalidated settings applied")
	}
	w = call(h, "POST", "/api/v1/admin/settings/2/validate", "", cookie, login.CSRF, origin)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = call(h, "POST", "/api/v1/admin/settings/2/apply", `{"expectedRevision":1}`, cookie, login.CSRF, origin)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = call(h, "POST", "/api/v1/admin/settings/1/apply", `{"expectedRevision":2}`, cookie, login.CSRF, origin)
	if w.Code != 204 {
		t.Fatal("rollback failed", w.Code, w.Body.String())
	}
	w = call(h, "POST", "/api/v1/auth/login", `{"email":"user@example.test","password":"strong-password-0123"}`, nil, "", origin)
	userCookie := w.Result().Cookies()[0]
	w = call(h, "GET", "/api/v1/admin/settings", "", userCookie, "", origin)
	if w.Code != 403 {
		t.Fatal("role bypass")
	}
	w = call(h, "PATCH", "/api/v1/auth/password", `{"currentPassword":"strong-password-0123","newPassword":"strong-password-0123"}`, cookie, login.CSRF, origin)
	if w.Code != 400 {
		t.Fatal("password reuse allowed", w.Code)
	}
	w = call(h, "PATCH", "/api/v1/auth/password", `{"currentPassword":"strong-password-0123","newPassword":"different-strong-password-567"}`, cookie, login.CSRF, origin)
	if w.Code != 204 {
		t.Fatal("password change", w.Code, w.Body.String())
	}
	w = call(h, "GET", "/api/v1/auth/me", "", cookie, "", origin)
	if w.Code != 401 {
		t.Fatal("old session survives password change")
	}
	if e := audit.Verify(ctx, p, a.Key); e != nil {
		t.Fatal(e)
	}
	if _, e := p.Exec(ctx, "UPDATE audit_events SET action='tampered' WHERE id=(SELECT min(id) FROM audit_events)"); e != nil {
		t.Fatal(e)
	}
	if e := audit.Verify(ctx, p, a.Key); e == nil {
		t.Fatal("audit corruption undetected")
	}
}

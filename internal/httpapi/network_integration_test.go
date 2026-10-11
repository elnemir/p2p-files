package httpapi

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/settings"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationNetworkPolicyAndHostRecovery(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	store := settings.Store{DB: c.a.DB, Key: c.a.Key}
	active, e := store.Active(ctx)
	if e != nil {
		t.Fatal(e)
	}
	value, e := config.ParseSettings(active.Value)
	if e != nil {
		t.Fatal(e)
	}
	value.PublicOrigin = "https://files.example.test"
	value.NetworkPolicy = config.NetworkPolicy{TrustedProxyCIDRs: []string{"10.0.0.2/32"}, AllowedClientCIDRs: []string{"203.0.113.0/24"}, RequireProxy: true, RequireHTTPS: true}
	raw, _ := json.Marshal(value)
	id, e := store.Draft(ctx, "host_cli", raw)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Validate(ctx, id); e != nil {
		t.Fatal(e)
	}
	stream, release, e := c.a.Gate.Acquire(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	if e = store.Apply(ctx, "host_cli", id, active.Revision+1); e == nil {
		t.Fatal("stale settings revision accepted")
	}
	if state, e := c.a.Operations.State(ctx); e != nil || state.Mode != "open" {
		t.Fatal("failed apply changed gate", state, e)
	}
	if e = store.Apply(ctx, "host_cli", id, active.Revision); e != nil {
		t.Fatal(e)
	}
	select {
	case <-stream.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("network update did not cancel privileged request")
	}
	release()
	if state, e := c.a.Operations.State(ctx); e != nil || state.Mode != "maintenance" {
		t.Fatal("network update did not close gate", state, e)
	}
	if w := c.req("GET", "/api/v1/admin/operations", "", nil); w.Code != 503 || !strings.Contains(w.Body.String(), "restart_required") {
		t.Fatal("old runtime survived network policy change", w.Code, w.Body.String())
	}
	restarted, e := New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(restarted.Close)
	c.a = restarted
	login := func(peer, forwarded, proto, host string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://"+host+"/api/v1/auth/login", strings.NewReader(`{"email":"sender@example.test","password":"strong-password-0123"}`))
		r.RemoteAddr = peer
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", value.PublicOrigin)
		r.Header.Set("X-Forwarded-For", forwarded)
		r.Header.Set("X-Forwarded-Proto", proto)
		w := httptest.NewRecorder()
		c.a.Handler().ServeHTTP(w, r)
		return w
	}
	for _, peer := range []string{"198.51.100.1:80", "10.0.0.3:80"} {
		if w := login(peer, "203.0.113.20", "https", "files.example.test"); w.Code != 403 {
			t.Fatal("untrusted peer bypass", w.Code)
		}
	}
	if w := login("10.0.0.2:80", "198.51.100.1", "https", "files.example.test"); w.Code != 403 {
		t.Fatal("allowlist bypass", w.Code)
	}
	w := login("10.0.0.2:80", "203.0.113.20", "https", "files.example.test")
	if w.Code != 200 || len(w.Result().Cookies()) == 0 || !w.Result().Cookies()[0].Secure {
		t.Fatal("trusted HTTPS ADMIN recovery login", w.Code, w.Body.String())
	}
	var count int
	if e = c.a.DB.QueryRow(ctx, "SELECT count FROM auth_limits WHERE key=$1", auth.Digest("ip:203.0.113.20")).Scan(&count); e != nil || count != 1 {
		t.Fatal("login did not use effective IP", count, e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM auth_limits WHERE key=$1", auth.Digest("ip:10.0.0.2")).Scan(&count); e != nil || count != 0 {
		t.Fatal("proxy IP used for login bucket", count, e)
	}
	if _, e = store.ResetNetworkAllowlist(ctx, " "); e == nil {
		t.Fatal("unaudited reset accepted")
	}
	reset, e := store.ResetNetworkAllowlist(ctx, "Repair operator access after network change")
	if e != nil || reset <= id {
		t.Fatal(reset, e)
	}
	current, e := database.Settings(ctx, c.a.DB)
	if e != nil || len(current.NetworkPolicy.AllowedClientCIDRs) != 0 || !current.NetworkPolicy.RequireProxy || !current.NetworkPolicy.RequireHTTPS || len(current.NetworkPolicy.TrustedProxyCIDRs) != 1 || current.PublicOrigin != value.PublicOrigin {
		t.Fatal("reset weakened transport trust", current, e)
	}
	if w := login("10.0.0.2:80", "198.51.100.1", "https", "files.example.test"); w.Code != 503 {
		t.Fatal("reset did not fence previous runtime", w.Code)
	}
	afterReset, e := New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(afterReset.Close)
	c.a = afterReset
	if w := login("10.0.0.2:80", "198.51.100.1", "https", "files.example.test"); w.Code != 200 {
		t.Fatal("reset did not repair allowed clients", w.Code)
	}
	if w := login("10.0.0.2:80", "198.51.100.1", "http", "files.example.test"); w.Code != 403 {
		t.Fatal("reset disabled HTTPS", w.Code)
	}
	if state, e := c.a.Operations.State(ctx); e != nil || state.Mode != "maintenance" || state.ReconciliationID != nil {
		t.Fatal("reset reopened gate", state, e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE action='network.allowlist_reset'").Scan(&count); e != nil || count != 1 {
		t.Fatal("reset missing audit", count, e)
	}
	if e = audit.Verify(ctx, c.a.DB, c.a.Key); e != nil {
		t.Fatal(e)
	}
}

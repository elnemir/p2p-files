package httpapi

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/identity"
	"strings"
	"sync"
	"testing"
	"time"
)

func identityFixture(t *testing.T, c apiClient) (identity.Store, identity.Provider, auth.Principal) {
	t.Helper()
	ctx := context.Background()
	actor, e := c.a.Auth.Authenticate(ctx, c.cookie.Value, false)
	if e != nil {
		t.Fatal(e)
	}
	store := c.a.identities()
	secret := "private-client-secret-fixture"
	config := identity.Config{Name: "Corporate", Protocol: "oidc", Issuer: "https://idp.example.test", ClientID: "client", EmailAttribute: "email", Enabled: true, AllowRegistration: true, PrivateCIDRs: []string{}, AllowedEmailDomains: []string{}, ExtraOrigins: []string{}}
	p, e := store.Apply(ctx, actor, "", identity.Input{Config: config, Secret: &secret})
	if e != nil {
		t.Fatal(e)
	}
	return store, p, actor
}
func verifiedIdentityFlow(t *testing.T, s identity.Store, p identity.Provider, actor auth.Principal, purpose, subject, email string) (string, string) {
	t.Helper()
	ctx := context.Background()
	f, e := identity.NewFlow(p, "http://localhost:8090", purpose, actor)
	if e != nil {
		t.Fatal(e)
	}
	state, binding := auth.Token(), auth.Token()
	if e = s.SaveFlow(ctx, state, binding, f); e != nil {
		t.Fatal(e)
	}
	f.Proof = identity.Proof{Subject: subject, Email: email, AuthenticatedAt: time.Now().Unix()}
	if e = s.VerifyFlow(ctx, state, f); e != nil {
		t.Fatal(e)
	}
	return state, binding
}
func TestIntegrationExternalIdentityRegistrationReplayAndCollision(t *testing.T) {
	c, _ := testClient(t)
	s, p, _ := identityFixture(t, c)
	ctx := context.Background()
	state, binding := verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "first", "external@example.test")
	if _, e := s.Complete(ctx, state, auth.Token(), "http://localhost:8090", 600); e == nil {
		t.Fatal("wrong browser binding")
	}
	if _, e := s.Complete(ctx, state, binding, "https://evil.test", 600); e == nil {
		t.Fatal("wrong origin")
	}
	var wg sync.WaitGroup
	results := make(chan auth.Session, 16)
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600)
			results <- session
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	success := 0
	for e := range errs {
		if e == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("replay consumed multiple times", success)
	}
	var logged auth.Session
	for result := range results {
		if result.Token != "" {
			logged = result
		}
	}
	if logged.User.Role != "user" {
		t.Fatal("external role escalation")
	}
	var hasPassword bool
	if e := c.a.DB.QueryRow(ctx, "SELECT password_hash IS NOT NULL FROM users WHERE id=$1", logged.User.ID).Scan(&hasPassword); e != nil || hasPassword {
		t.Fatal("JIT local credential", e)
	}
	for _, subject := range []string{"another-subject", "third-subject"} {
		state, binding = verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", subject, "external@example.test")
		if _, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
			t.Fatal("auto linked email")
		}
	}
	// Stable provider subject keeps the account even when an IdP changes the email.
	state, binding = verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "first", "changed@example.test")
	again, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600)
	if e != nil || again.User.ID != logged.User.ID || again.User.Email != "external@example.test" {
		t.Fatal("stable subject", again, e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE users SET disabled=true WHERE id=$1", logged.User.ID); e != nil {
		t.Fatal(e)
	}
	state, binding = verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "first", "external@example.test")
	if _, e = s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("disabled account")
	}
}
func TestIntegrationExternalIdentityLinkReauthAndProviderRevocation(t *testing.T) {
	c, _ := testClient(t)
	s, p, actor := identityFixture(t, c)
	ctx := context.Background()
	state, binding := verifiedIdentityFlow(t, s, p, actor, "link", "admin-external", "unrelated@example.test")
	linked, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600)
	if e != nil || linked.User.ID != actor.ID {
		t.Fatal(linked, e)
	}
	if _, e = c.a.Auth.Authenticate(ctx, c.cookie.Value, false); e == nil {
		t.Fatal("prior session not revoked")
	}
	principal, e := c.a.Auth.Authenticate(ctx, linked.Token, false)
	if e != nil {
		t.Fatal(e)
	}
	state, binding = verifiedIdentityFlow(t, s, p, principal, "reauth", "wrong-subject", "admin@example.test")
	if _, e = s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("wrong reauth identity")
	}
	state, binding = verifiedIdentityFlow(t, s, p, principal, "reauth", "admin-external", "unrelated@example.test")
	reauth, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600)
	if e != nil {
		t.Fatal(e)
	}
	fresh, e := c.a.Auth.Authenticate(ctx, reauth.Token, false)
	if e != nil || time.Since(fresh.Reauth) > time.Minute {
		t.Fatal("external reauth", e)
	}
	pending, nonce := verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "pending", "pending@example.test")
	config := p.Config
	config.Enabled = false
	updated, e := s.Apply(ctx, fresh, p.ID, identity.Input{ExpectedRevision: p.Revision, Config: config})
	if e != nil || updated.Revision != 2 {
		t.Fatal(updated, e)
	}
	if _, e = c.a.Auth.Authenticate(ctx, reauth.Token, false); e == nil {
		t.Fatal("provider session survived disable")
	}
	if _, e = s.Complete(ctx, pending, nonce, "http://localhost:8090", 600); e == nil {
		t.Fatal("pending flow survived disable")
	}
	// Local emergency login remains usable after the linked provider is disabled.
	if w := c.req("POST", "/api/v1/auth/login", `{"email":"sender@example.test","password":"strong-password-0123"}`, nil); w.Code != 200 {
		t.Fatal("local fallback", w.Code)
	}
}
func TestIntegrationExternalIdentityProviderGuardsAndSecretRedaction(t *testing.T) {
	c, _ := testClient(t)
	s, p, actor := identityFixture(t, c)
	ctx := context.Background()
	raw, e := json.Marshal(p)
	if e != nil || strings.Contains(string(raw), "private-client-secret-fixture") {
		t.Fatal("secret leaked", e)
	}
	w := c.req("GET", "/api/v1/auth/providers", "", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), "issuer") || strings.Contains(w.Body.String(), "clientId") {
		t.Fatal("public disclosure", w.Code)
	}
	w = c.req("GET", "/api/v1/admin/identity-providers", "", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), "private-client-secret-fixture") {
		t.Fatal("secret readback", w.Code)
	}
	bad := p.Config
	bad.Issuer = "https://other.test"
	if _, e = s.Apply(ctx, actor, p.ID, identity.Input{ExpectedRevision: p.Revision, Config: bad}); e == nil {
		t.Fatal("authority rebound")
	}
	if _, e = s.Apply(ctx, actor, p.ID, identity.Input{ExpectedRevision: 0, Config: p.Config}); e == nil {
		t.Fatal("stale CAS")
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp()-interval '6 minutes' WHERE id=$1", actor.SessionID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Apply(ctx, actor, p.ID, identity.Input{ExpectedRevision: 1, Config: p.Config}); e == nil {
		t.Fatal("stale ADMIN reauth")
	}
	for _, path := range []string{"/api/v1/auth/providers/" + p.ID + "/start", "/api/v1/auth/complete"} {
		if w = c.req("POST", path, `{"purpose":"login"}`, map[string]string{"Origin": "https://evil.test"}); w.Code != 403 {
			t.Fatal("Origin bypass", w.Code)
		}
	}
	var secretBox []byte
	if e = c.a.DB.QueryRow(ctx, "SELECT secret_box FROM identity_providers WHERE id=$1", p.ID).Scan(&secretBox); e != nil || strings.Contains(string(secretBox), "private-client-secret-fixture") {
		t.Fatal("plaintext secret", e)
	}
}
func TestIntegrationExternalIdentityAuditFaultRollsBackRegistration(t *testing.T) {
	c, _ := testClient(t)
	s, p, _ := identityFixture(t, c)
	ctx := context.Background()
	state, binding := verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "fault-subject", "fault@example.test")
	_, e := c.a.DB.Exec(ctx, `CREATE FUNCTION reject_identity_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='identity.registered' THEN RAISE EXCEPTION 'injected'; END IF; RETURN NEW; END $$; CREATE TRIGGER identity_fault BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_identity_audit()`)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("audit failure ignored")
	}
	var n int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM users WHERE email='fault@example.test'").Scan(&n); e != nil || n != 0 {
		t.Fatal("partial registration", e)
	}
	if _, e = c.a.DB.Exec(ctx, "DROP TRIGGER identity_fault ON audit_events"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Complete(ctx, state, binding, "http://localhost:8090", 600); e != nil {
		t.Fatal("atomic flow rollback", e)
	}
}

func TestIntegrationExternalIdentityExpiryAndAuthorityGuards(t *testing.T) {
	c, _ := testClient(t)
	s, p, actor := identityFixture(t, c)
	ctx := context.Background()
	state, binding := verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "expired", "expired@example.test")
	if _, e := c.a.DB.Exec(ctx, "UPDATE identity_flows SET expires_at=clock_timestamp()-interval '1 second' WHERE state_hash=$1", auth.Digest(state)); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("expired flow accepted")
	}
	state, binding = verifiedIdentityFlow(t, s, p, actor, "link", "revoked", "revoked@example.test")
	if _, e := c.a.DB.Exec(ctx, "UPDATE web_sessions SET revoked_at=clock_timestamp() WHERE id=$1", actor.SessionID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("revoked actor linked")
	}
	login, e := c.a.Auth.Login(ctx, "sender@example.test", "strong-password-0123", "identity-test", 600)
	if e != nil {
		t.Fatal(e)
	}
	actor, e = c.a.Auth.Authenticate(ctx, login.Token, false)
	if e != nil {
		t.Fatal(e)
	}
	p.Config.AllowRegistration = false
	p, e = s.Apply(ctx, actor, p.ID, identity.Input{ExpectedRevision: p.Revision, Config: p.Config})
	if e != nil {
		t.Fatal(e)
	}
	state, binding = verifiedIdentityFlow(t, s, p, auth.Principal{}, "login", "no-jit", "no-jit@example.test")
	if _, e = s.Complete(ctx, state, binding, "http://localhost:8090", 600); e == nil {
		t.Fatal("JIT disabled ignored")
	}
	var count int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM users WHERE email IN ('expired@example.test','revoked@example.test','no-jit@example.test')").Scan(&count); e != nil || count != 0 {
		t.Fatal("guard created account", count, e)
	}
}

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/branding"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/reports"
	"fmt"
	"strings"
	"testing"
)

func brandingBody(t *testing.T, rev int64, b branding.Config) string {
	t.Helper()
	raw, e := json.Marshal(map[string]any{"expectedRevision": rev, "value": b})
	if e != nil {
		t.Fatal(e)
	}
	return string(raw)
}
func applyBrand(t *testing.T, c apiClient, rev int64, name string) branding.Revision {
	t.Helper()
	b := branding.Default()
	b.ProductName = name
	w := c.req("PUT", "/api/v1/admin/branding", brandingBody(t, rev, b), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result branding.Revision
	decode(t, w, &result)
	return result
}
func TestIntegrationBrandingAuthorizationAndValidation(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	public := c
	public.cookie = nil
	public.csrf = ""
	w := public.req("GET", "/api/v1/branding", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var initial branding.Revision
	decode(t, w, &initial)
	if initial.Revision != 0 || initial.Value.ProductName != "FileX" {
		t.Fatal("default branding")
	}
	if cached := public.req("GET", "/api/v1/branding", "", map[string]string{"If-None-Match": w.Header().Get("ETag")}); cached.Code != 304 || cached.Body.Len() != 0 {
		t.Fatal("conditional response", cached.Code)
	}
	payload := brandingBody(t, 0, branding.Default())
	if w = public.req("PUT", "/api/v1/admin/branding", payload, nil); w.Code != 401 {
		t.Fatal("anonymous publish", w.Code)
	}
	for name, headers := range map[string]map[string]string{"csrf": {"X-CSRF-Token": "wrong"}, "origin": {"Origin": "https://evil.test"}} {
		if w = c.req("PUT", "/api/v1/admin/branding", payload, headers); w.Code != 403 {
			t.Fatal(name, w.Code)
		}
	}
	for _, invalid := range []string{strings.Replace(payload, `"productName":"FileX"`, `"productName":null`, 1), strings.Replace(payload, `,"favicon":""`, "", 1), strings.Replace(payload, `"#196342"`, `"url(https://evil.test)"`, 1), `{"expectedRevision":0,"value":{},"ignored":true}`, strings.Repeat(" ", branding.MaxRequestBytes) + payload} {
		if w = c.req("PUT", "/api/v1/admin/branding", invalid, nil); w.Code != 400 {
			t.Fatal("invalid accepted", w.Code)
		}
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp()-interval '6 minutes'"); e != nil {
		t.Fatal(e)
	}
	if w = c.req("PUT", "/api/v1/admin/branding", payload, nil); w.Code != 403 {
		t.Fatal("expired reauth", w.Code)
	}
	if w = c.req("POST", "/api/v1/auth/reauth", `{"password":"strong-password-0123"}`, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	token := c.req("POST", "/api/v1/auth/tokens", `{"scopes":["account:read"],"ttlSeconds":600}`, nil)
	if token.Code != 201 {
		t.Fatal(token.Code)
	}
	var credential struct{ Token string }
	decode(t, token, &credential)
	if w = c.req("PUT", "/api/v1/admin/branding", payload, map[string]string{"Authorization": "Bearer " + credential.Token}); w.Code != 403 {
		t.Fatal("bearer accepted", w.Code)
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE users SET role='user' WHERE email='sender@example.test'"); e != nil {
		t.Fatal(e)
	}
	if w = c.req("PUT", "/api/v1/admin/branding", payload, nil); w.Code != 403 {
		t.Fatal("USER publish", w.Code)
	}
	if w = c.req("GET", "/api/v1/admin/branding", "", nil); w.Code != 403 {
		t.Fatal("USER admin read", w.Code)
	}
	var count int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM branding_revisions").Scan(&count); e != nil || count != 0 {
		t.Fatal("rejected input mutated state", e, count)
	}
}
func TestIntegrationBrandingCASAndAuditRollback(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	bodies := []string{}
	for _, name := range []string{"Brand A", "Brand B"} {
		b := branding.Default()
		b.ProductName = name
		bodies = append(bodies, brandingBody(t, 0, b))
	}
	start := make(chan struct{})
	results := make(chan int, 2)
	for _, body := range bodies {
		go func() { <-start; results <- c.req("PUT", "/api/v1/admin/branding", body, nil).Code }()
	}
	close(start)
	got := map[int]int{}
	for range 2 {
		got[<-results]++
	}
	if got[200] != 1 || got[409] != 1 {
		t.Fatal("lost update", got)
	}
	current, e := branding.Current(ctx, c.a.DB)
	if e != nil || current.Revision != 1 {
		t.Fatal(current.Revision, e)
	}
	var events int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE action='branding.applied'").Scan(&events); e != nil || events != 1 {
		t.Fatal("audit mismatch", e, events)
	}
	if _, e = c.a.DB.Exec(ctx, `CREATE FUNCTION fail_branding_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='branding.applied' THEN RAISE EXCEPTION 'injected audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER branding_audit_fault BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_branding_audit()`); e != nil {
		t.Fatal(e)
	}
	b := branding.Default()
	b.ProductName = "Never published"
	if w := c.req("PUT", "/api/v1/admin/branding", brandingBody(t, current.Revision, b), nil); w.Code != 500 {
		t.Fatal("fault injection", w.Code)
	}
	after, e := branding.Current(ctx, c.a.DB)
	if e != nil || after != current {
		t.Fatal("partial apply on audit error", e)
	}
	if _, e = c.a.DB.Exec(ctx, `DROP TRIGGER branding_audit_fault ON audit_events`); e != nil {
		t.Fatal(e)
	}
	if e = audit.Verify(ctx, c.a.DB, c.a.Key); e != nil {
		t.Fatal(e)
	}
	// A stale authenticated principal must fail after demotion, even inside the store.
	p, e := c.a.Auth.Authenticate(ctx, c.cookie.Value, false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE users SET role='user' WHERE id=$1", p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = (branding.Store{DB: c.a.DB, Key: c.a.Key}).Apply(ctx, p, current.Revision, b); e == nil {
		t.Fatal("stale ADMIN accepted")
	}
}
func TestIntegrationBrandingReportsAndMail(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	artifactProvider(t, c)
	first := applyBrand(t, c, 0, "Acme Secure")
	id := c.draft()
	c.upload(id, "document.txt", "report fixture")
	c.publish(id)
	w := c.req("POST", "/api/v1/transfers/"+id+"/reports", `{"kind":"user","locale":"en","timezone":"UTC"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var report struct{ ID string }
	decode(t, w, &report)
	raw, hash, e := c.a.reportService().Snapshot(ctx, report.ID)
	if e != nil {
		t.Fatal(e)
	}
	applyBrand(t, c, first.Revision, "New Company")
	if e = c.a.reportService().RenderPending(ctx); e != nil {
		t.Fatal(e)
	}
	var presentation []byte
	if e = c.a.DB.QueryRow(ctx, "SELECT branding FROM reports WHERE id=$1", report.ID).Scan(&presentation); e != nil {
		t.Fatal(e)
	}
	stored, e := branding.DecodeSnapshot(presentation)
	if e != nil || stored.ProductName != "Acme Secure" {
		t.Fatal("mutable presentation", e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE reports SET branding='{}' WHERE id=$1", report.ID); e == nil {
		t.Fatal("immutable branding bypass")
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE branding_revisions SET value='{}' WHERE revision=$1", first.Revision); e == nil {
		t.Fatal("immutable revision bypass")
	}
	var expected bytes.Buffer
	if e = reports.RenderBranded(&expected, raw, hash, "pdf", &stored); e != nil {
		t.Fatal(e)
	}
	var actual string
	if e = c.a.DB.QueryRow(ctx, "SELECT a.sha256 FROM report_artifacts r JOIN artifacts a ON a.id=r.artifact_id WHERE r.report_id=$1 AND r.format='pdf'", report.ID).Scan(&actual); e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(expected.Bytes())
	if actual != hex.EncodeToString(sum[:]) {
		t.Fatal("renderer used current branding instead of saved snapshot")
	}
	raw2, hash2, e := c.a.reportService().Snapshot(ctx, report.ID)
	if e != nil || hash != hash2 || !bytes.Equal(raw, raw2) {
		t.Fatal("canonical report changed", e)
	}
	if bytes.Contains(raw, []byte("Acme Secure")) || bytes.Contains(raw, []byte("New Company")) {
		t.Fatal("strict report JSON changed")
	}
	if w = c.req("PUT", "/api/v1/admin/smtp", `{"host":"mail.example.test","port":587,"tls":"starttls","from":"sender@example.test"}`, nil); w.Code != 200 {
		t.Fatal(w.Code)
	}
	service := communications.Service{DB: c.a.DB, Keys: c.a.Keys}
	if e = service.Fanout(ctx); e != nil {
		t.Fatal(e)
	}
	if e = service.Schedule(ctx); e != nil {
		t.Fatal(e)
	}
	sent := 0
	service.Mail = func(_ context.Context, cfg communications.SMTPConfig, _ string, _ string, subject, body string) error {
		sent++
		if cfg.DisplayName != "New Company" || !strings.Contains(body, "New Company") || strings.Contains(subject+body, "FileX") {
			t.Error("wrong mail branding")
			return fmt.Errorf("wrong mail branding")
		}
		return nil
	}
	config, e := database.Settings(ctx, c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	for range 4 {
		if e = service.Deliver(ctx, config); e != nil {
			t.Fatal(e)
		}
	}
	if sent != 1 {
		t.Fatal("no branded SMTP attempt", sent)
	}
	// A failed branded transport must remain retryable, never become delivered.
	if _, e = c.a.DB.Exec(ctx, "UPDATE deliveries SET state='pending',available_at=clock_timestamp(),delivered_at=NULL WHERE kind='smtp'"); e != nil {
		t.Fatal(e)
	}
	service.Mail = func(context.Context, communications.SMTPConfig, string, string, string, string) error {
		return errors.New("injected SMTP failure")
	}
	if e = service.Deliver(ctx, config); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM deliveries WHERE kind='smtp'").Scan(&state); e != nil || state != "pending" {
		t.Fatal("SMTP failure lost", state, e)
	}
}

package httpapi

import (
	"context"
	"errors"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/database"
	"strings"
	"testing"
	"time"
)

func TestIntegrationOutboxRetryDLQAndPrivacy(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	w := c.req("PUT", "/api/v1/admin/smtp", `{"host":"mail.example.test","port":587,"tls":"starttls","username":"test","password":"smtp-secret","from":"sender@example.test"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("POST", "/api/v1/webhooks", `{"url":"https://hooks.example.test/events","secret":"01234567890123456789012345678901"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	id := c.draft()
	c.upload(id, "a", "body")
	c.publish(id)
	s := communications.Service{DB: c.a.DB, Keys: c.a.Keys}
	if e := s.Fanout(ctx); e != nil {
		t.Fatal(e)
	}
	if e := s.Fanout(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM notifications").Scan(&count); e != nil || count != 1 {
		t.Fatal("outbox duplicate", e, count)
	}
	if e := s.Schedule(ctx); e != nil {
		t.Fatal(e)
	}
	conf, e := database.Settings(ctx, c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	sent := 0
	s.Mail = func(ctx context.Context, config communications.SMTPConfig, id, to, title, body string) error {
		sent++
		if strings.Contains(body, "recipient@example") {
			t.Fatal("recipient correlation")
		}
		return nil
	}
	s.Hook = func(context.Context, communications.WebhookConfig, string, []byte) error {
		return errors.New("simulated connection failure")
	}
	for i := 0; i < 10; i++ {
		if _, e = c.a.DB.Exec(ctx, "UPDATE deliveries SET available_at=clock_timestamp() WHERE state='pending'"); e != nil {
			t.Fatal(e)
		}
		if e = s.Deliver(ctx, conf); e != nil {
			t.Fatal(e)
		}
	}
	if sent != 1 {
		t.Fatal("SMTP resend", sent)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM deliveries WHERE state='dead'").Scan(&count); e != nil || count != 1 {
		t.Fatal("DLQ", e, count)
	}
	var delivery string
	if e = c.a.DB.QueryRow(ctx, "SELECT id::text FROM deliveries WHERE state='dead'").Scan(&delivery); e != nil {
		t.Fatal(e)
	}
	w = c.req("POST", "/api/v1/deliveries/"+delivery+"/retry", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	s.Hook = func(context.Context, communications.WebhookConfig, string, []byte) error { return nil }
	if e = s.Deliver(ctx, conf); e != nil {
		t.Fatal(e)
	}
	w = c.req("GET", "/api/v1/notifications", "", nil)
	for _, secret := range []string{"smtp-secret", "0123456789", "recipient_email", "token"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("private data leaked")
		}
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE deliveries SET state='sending',lease_until=clock_timestamp()-interval '1 second' WHERE id=$1", delivery); e != nil {
		t.Fatal(e)
	}
	if e = s.Deliver(ctx, conf); e != nil {
		t.Fatal("crash recovery", e)
	}
	_ = time.Second
}

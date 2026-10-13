package communications

import (
	"context"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestWebhookGuards(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.2.3.4", "100.64.1.2", "169.254.169.254", "192.0.0.1", "198.18.0.1", "::1", "::ffff:8.8.8.8"} {
		if PublicIPv4(netip.MustParseAddr(s)) {
			t.Fatal("SSRF bypass", s)
		}
	}
	if !PublicIPv4(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("public IPv4 rejected")
	}
	for _, u := range []string{"http://example.com", "https://127.0.0.1", "https://[::1]", "https://user:pass@example.com", "https://example.com:8443", "https://example.com/#fragment"} {
		if (WebhookConfig{URL: u, Secret: "12345678901234567890123456789012"}).Validate() == nil {
			t.Fatal("URL accepted", u)
		}
	}
	b := []byte(`{"eventId":"one"}`)
	now := time.Now()
	sig := Signature("secret", now.Unix(), b)
	if !VerifySignature("secret", now.Unix(), b, sig, now) || VerifySignature("secret", now.Unix(), []byte("tampered"), sig, now) || VerifySignature("secret", now.Unix(), b, sig, now.Add(301*time.Second)) {
		t.Fatal("HMAC or replay window")
	}
}
func TestSMTPRequiresSTARTTLS(t *testing.T) {
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		t.Skip("local sockets unavailable")
	}
	defer l.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, e := l.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(time.Second))
		conn.Write([]byte("220 test ESMTP\r\n"))
		b := make([]byte, 512)
		conn.Read(b)
		conn.Write([]byte("250 test\r\n"))
		conn.Read(b)
	}()
	port := l.Addr().(*net.TCPAddr).Port
	e = SendMail(context.Background(), SMTPConfig{Host: "127.0.0.1", Port: port, TLS: "starttls", From: "a@example.test"}, "test", "b@example.test", "subject", "body")
	if e == nil {
		t.Fatal("unencrypted SMTP accepted")
	}
	<-done
}
func TestQuietHoursDSTAndTemplates(t *testing.T) {
	start, end := 22, 8
	if !Quiet(time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC), "Europe/Moscow", &start, &end) {
		t.Fatal("timezone ignored")
	}
	if Quiet(time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC), "Europe/Moscow", &start, &end) {
		t.Fatal("daytime quiet")
	}
	for _, locale := range []string{"ru", "en"} {
		title, body := Template("transfer.published", locale, "https://test/#token=private", 1)
		if title == "" || body == "" {
			t.Fatal("missing template")
		}
	}
}

func TestBrandedTemplatesPreserveCapabilities(t *testing.T) {
	link := "https://example.test/receive#token=FileX-fixture"
	for _, locale := range []string{"ru", "en"} {
		for _, kind := range []string{"transfer.published", "transfer.expiring", "transfer.expired", "download.completed", "digest", "other"} {
			subject, body := BrandedTemplate(kind, locale, link, 3, "North Share")
			if kind == "transfer.published" {
				if !strings.Contains(body, link) {
					t.Fatal("brand substitution modified capability")
				}
			} else if strings.Contains(subject+body, "FileX") {
				t.Fatal("vendor copy remains", locale, kind)
			}
			if kind == "digest" && (!strings.Contains(subject+body, "North Share") || !strings.Contains(body, "3")) {
				t.Fatal("branded digest missing", locale)
			}
		}
	}
}

package communications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/netip"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      string `json:"tls"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

func (c SMTPConfig) Validate() error {
	a, e := mail.ParseAddress(c.From)
	if e != nil || a.Address != c.From || strings.ContainsAny(c.Host, "\r\n /:@") || c.Host == "" || c.Port < 1 || c.Port > 65535 || (c.TLS != "starttls" && c.TLS != "smtps") {
		return errors.New("invalid SMTP configuration")
	}
	return nil
}
func SendMail(ctx context.Context, c SMTPConfig, id, to, subject, body string) error {
	return sendMailTLS(ctx, c, id, to, subject, body, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: c.Host})
}
func sendMailTLS(ctx context.Context, c SMTPConfig, id, to, subject, body string, tlsConfig *tls.Config) error {
	if e := c.Validate(); e != nil {
		return e
	}
	a, e := mail.ParseAddress(to)
	if e != nil || a.Address != to || strings.ContainsAny(subject, "\r\n") {
		return errors.New("invalid mail envelope")
	}
	address := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	if c.TLS == "smtps" {
		conn, e = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp4", address)
	} else {
		conn, e = dialer.DialContext(ctx, "tcp4", address)
	}
	if e != nil {
		return e
	}
	defer conn.Close()
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if e = conn.SetDeadline(deadline); e != nil {
		return e
	}
	client, e := smtp.NewClient(conn, c.Host)
	if e != nil {
		return e
	}
	defer client.Close()
	if c.TLS == "starttls" {
		if e = client.StartTLS(tlsConfig); e != nil {
			return e
		}
	}
	if c.Username != "" {
		if e = client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); e != nil {
			return e
		}
	}
	if e = client.Mail(c.From); e != nil {
		return e
	}
	if e = client.Rcpt(to); e != nil {
		return e
	}
	writer, e := client.Data()
	if e != nil {
		return e
	}
	var message bytes.Buffer
	fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: <%s@filex.local>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", c.From, to, mime.QEncoding.Encode("utf-8", subject), id)
	q := quotedprintable.NewWriter(&message)
	if _, e = io.WriteString(q, body); e != nil {
		return e
	}
	if e = q.Close(); e != nil {
		return e
	}
	if _, e = writer.Write(message.Bytes()); e != nil {
		return e
	}
	if e = writer.Close(); e != nil {
		return e
	}
	return client.Quit()
}

type WebhookConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

var denied = []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4"}

func PublicIPv4(ip netip.Addr) bool {
	if !ip.Is4() || !ip.IsGlobalUnicast() {
		return false
	}
	for _, s := range denied {
		if netip.MustParsePrefix(s).Contains(ip) {
			return false
		}
	}
	return true
}
func (c WebhookConfig) Validate() error {
	u, e := url.Parse(c.URL)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || len(c.Secret) < 32 || len(c.Secret) > 1024 {
		return errors.New("invalid webhook configuration")
	}
	if ip, e := netip.ParseAddr(u.Hostname()); e == nil && !PublicIPv4(ip) {
		return errors.New("webhook destination rejected")
	}
	return nil
}
func Signature(secret string, at int64, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(h, "%d.", at)
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
func VerifySignature(secret string, at int64, payload []byte, signature string, now time.Time) bool {
	if at < now.Unix()-300 || at > now.Unix()+300 {
		return false
	}
	got, e := hex.DecodeString(signature)
	if e != nil {
		return false
	}
	want, _ := hex.DecodeString(Signature(secret, at, payload))
	return hmac.Equal(got, want)
}
func WebhookClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil || port != "443" {
			return nil, errors.New("webhook port rejected")
		}
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
		if e != nil || len(ips) == 0 {
			return nil, errors.New("webhook DNS unavailable")
		}
		for _, ip := range ips {
			if !PublicIPv4(ip) {
				return nil, errors.New("webhook destination rejected")
			}
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", net.JoinHostPort(ips[0].String(), port))
	}}}
}
func SendWebhook(ctx context.Context, c WebhookConfig, id string, payload []byte) error {
	if e := c.Validate(); e != nil {
		return e
	}
	at := time.Now().Unix()
	req, e := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(payload))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FileX-Event-ID", id)
	req.Header.Set("X-FileX-Timestamp", strconv.FormatInt(at, 10))
	req.Header.Set("X-FileX-Signature", Signature(c.Secret, at, payload))
	response, e := WebhookClient().Do(req)
	if e != nil {
		return e
	}
	defer response.Body.Close()
	if _, e = io.Copy(io.Discard, io.LimitReader(response.Body, 65536)); e != nil {
		return e
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return errors.New("webhook non-success response")
	}
	return nil
}

package identity

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"filex.local/filex/internal/auth"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponse = 2 << 20

func (c Config) allowedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || !ip.IsGlobalUnicast() {
		return false
	}
	if ip.IsPrivate() {
		for _, raw := range c.PrivateCIDRs {
			_, n, e := net.ParseCIDR(raw)
			if e == nil && n.Contains(ip) {
				return true
			}
		}
		return false
	}
	// Never permit carrier NAT or cloud instance metadata via public fallback.
	for _, raw := range []string{"100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15"} {
		_, n, _ := net.ParseCIDR(raw)
		if n.Contains(ip) {
			return false
		}
	}
	return true
}
func (c Config) dial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, e := net.SplitHostPort(address)
	if e != nil {
		return nil, auth.ErrInvalid
	}
	ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
	if e != nil || len(ips) == 0 {
		return nil, auth.ErrCredentials
	}
	for _, ip := range ips {
		if !c.allowedIP(ip.IP) {
			return nil, auth.ErrForbidden
		}
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	var last error
	for _, ip := range ips {
		conn, e := d.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
		if e == nil {
			return conn, nil
		}
		last = e
	}
	return nil, last
}
func (c Config) tls(host string) (*tls.Config, error) {
	roots, e := x509.SystemCertPool()
	if e != nil {
		roots = x509.NewCertPool()
	}
	if c.CAPEM != "" && !roots.AppendCertsFromPEM([]byte(c.CAPEM)) {
		return nil, auth.ErrInvalid
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: host}, nil
}

type boundedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.remaining == 0 {
		var probe [1]byte
		n, e := b.ReadCloser.Read(probe[:])
		if n > 0 {
			return 0, errors.New("identity response too large")
		}
		return 0, e
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, e := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	return n, e
}

type identityTransport struct {
	inner   *http.Transport
	origins map[string]bool
}

func (t identityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" || r.URL.User != nil || !t.origins[r.URL.Scheme+"://"+r.URL.Host] {
		return nil, auth.ErrForbidden
	}
	resp, e := t.inner.RoundTrip(r)
	if e != nil {
		return nil, e
	}
	if resp.ContentLength > maxResponse {
		resp.Body.Close()
		return nil, auth.ErrInvalid
	}
	resp.Body = &boundedBody{resp.Body, maxResponse}
	return resp, nil
}
func (c Config) client() (*http.Client, error) {
	cfg, e := c.tls("")
	if e != nil {
		return nil, e
	}
	origins := map[string]bool{}
	u, e := url.Parse(c.Issuer)
	if e != nil {
		return nil, e
	}
	origins[u.Scheme+"://"+u.Host] = true
	for _, v := range c.ExtraOrigins {
		origins[v] = true
	}
	tr := &http.Transport{Proxy: nil, DialContext: c.dial, TLSClientConfig: cfg, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 32768, DisableKeepAlives: true}
	return &http.Client{Transport: identityTransport{tr, origins}, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return auth.ErrForbidden }}, nil
}
func (c Config) allowedEndpoint(raw string) bool {
	u, e := httpsURL(raw)
	if e != nil {
		return false
	}
	issuer, _ := url.Parse(c.Issuer)
	origin := u.Scheme + "://" + u.Host
	if origin == issuer.Scheme+"://"+issuer.Host {
		return true
	}
	for _, v := range c.ExtraOrigins {
		if v == origin {
			return true
		}
	}
	return false
}
func normalizedUsername(v string) bool {
	return len(v) > 0 && len(v) <= 254 && strings.TrimSpace(v) == v && !strings.ContainsRune(v, 0)
}

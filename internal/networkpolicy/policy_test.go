package networkpolicy

import (
	"filex.local/filex/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func strictSettings() config.Settings {
	return config.Settings{PublicOrigin: "https://files.example.test", NetworkPolicy: config.NetworkPolicy{TrustedProxyCIDRs: []string{"10.0.0.2/32"}, AllowedClientCIDRs: []string{"203.0.113.0/24"}, RequireProxy: true, RequireHTTPS: true}}
}

func proxyRequest() *http.Request {
	r := httptest.NewRequest("GET", "http://files.example.test/api/v1/banner", nil)
	r.RemoteAddr = "10.0.0.2:2345"
	r.Header.Set("X-Forwarded-For", "203.0.113.20")
	r.Header.Set("X-Forwarded-Proto", "https")
	return r
}

func TestSingleTrustedProxyRejectsSpoofing(t *testing.T) {
	s := strictSettings()
	ip, e := Resolve(proxyRequest(), s)
	if e != nil || ip.String() != "203.0.113.20" {
		t.Fatal(ip, e)
	}
	for name, mutate := range map[string]func(*http.Request){
		"untrusted peer": func(r *http.Request) { r.RemoteAddr = "10.0.0.3:1234" },
		"unknown peer":   func(r *http.Request) { r.RemoteAddr = "invalid" },
		"outside CIDR":   func(r *http.Request) { r.Header.Set("X-Forwarded-For", "198.51.100.9") },
		"header chain":   func(r *http.Request) { r.Header.Set("X-Forwarded-For", "203.0.113.20, 198.51.100.9") },
		"duplicate IP":   func(r *http.Request) { r.Header.Add("X-Forwarded-For", "203.0.113.21") },
		"duplicate TLS":  func(r *http.Request) { r.Header.Add("X-Forwarded-Proto", "https") },
		"unknown IP":     func(r *http.Request) { r.Header.Set("X-Forwarded-For", "unknown") },
		"IP port":        func(r *http.Request) { r.Header.Set("X-Forwarded-For", "203.0.113.20:80") },
		"missing IP":     func(r *http.Request) { r.Header.Del("X-Forwarded-For") },
		"missing TLS":    func(r *http.Request) { r.Header.Del("X-Forwarded-Proto") },
		"HTTP":           func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "http") },
		"TLS chain":      func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https,http") },
		"host poisoning": func(r *http.Request) { r.Host = "attacker.example.test" },
		"zero address":   func(r *http.Request) { r.Header.Set("X-Forwarded-For", "0.0.0.0") },
		"mapped IPv4":    func(r *http.Request) { r.Header.Set("X-Forwarded-For", "::ffff:203.0.113.20") },
	} {
		t.Run(name, func(t *testing.T) {
			r := proxyRequest()
			mutate(r)
			if _, e := Resolve(r, s); e == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
	// Untrusted caller headers never affect direct-mode identity or its allowlist.
	r := proxyRequest()
	r.RemoteAddr = "198.51.100.7:1234"
	r.Header.Set("Forwarded", "for=203.0.113.20;proto=https")
	r.Header.Set("X-Real-IP", "203.0.113.20")
	ip, e = Resolve(r, config.Settings{})
	if e != nil || ip.String() != "198.51.100.7" {
		t.Fatal("untrusted metadata affected peer", ip, e)
	}
	if _, e = Resolve(r, config.Settings{NetworkPolicy: config.NetworkPolicy{AllowedClientCIDRs: []string{"203.0.113.0/24"}}}); e == nil {
		t.Fatal("forwarded headers bypassed allowlist")
	}
}

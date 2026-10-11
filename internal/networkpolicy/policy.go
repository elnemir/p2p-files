// Package networkpolicy resolves the single trusted Traefik hop without trusting client headers.
package networkpolicy

import (
	"errors"
	"filex.local/filex/internal/config"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

var ErrDenied = errors.New("network policy denied")
var ErrMalformed = errors.New("invalid proxy metadata")

func contains(cidrs []string, addr netip.Addr) bool {
	for _, raw := range cidrs {
		p, e := netip.ParsePrefix(raw)
		if e == nil && p.Contains(addr) {
			return true
		}
	}
	return false
}

func Peer(raw string) (netip.Addr, error) {
	host, _, e := net.SplitHostPort(raw)
	if e != nil {
		host = raw
	}
	ip, e := netip.ParseAddr(host)
	if e != nil || ip.Zone() != "" || ip.IsUnspecified() || ip.IsMulticast() {
		return netip.Addr{}, ErrMalformed
	}
	return ip.Unmap(), nil
}

func Resolve(r *http.Request, s config.Settings) (netip.Addr, error) {
	if e := config.ValidateNetworkPolicy(s); e != nil {
		return netip.Addr{}, ErrDenied
	}
	client, e := Peer(r.RemoteAddr)
	if e != nil {
		return client, e
	}
	p := s.NetworkPolicy
	trusted := contains(p.TrustedProxyCIDRs, client)
	if p.RequireProxy && !trusted {
		return netip.Addr{}, ErrDenied
	}
	secure := r.TLS != nil
	if trusted {
		forwarded, proto := r.Header.Values("X-Forwarded-For"), r.Header.Values("X-Forwarded-Proto")
		if len(forwarded) != 1 || len(proto) != 1 || (proto[0] != "http" && proto[0] != "https") {
			return netip.Addr{}, ErrMalformed
		}
		client, e = netip.ParseAddr(strings.TrimSpace(forwarded[0]))
		if e != nil || client.Zone() != "" || !client.Is4() || client.IsUnspecified() || client.IsMulticast() {
			return netip.Addr{}, ErrMalformed
		}
		secure = proto[0] == "https"
	}
	if p.RequireHTTPS {
		origin, e := url.Parse(s.PublicOrigin)
		if e != nil || !secure || !strings.EqualFold(r.Host, origin.Host) {
			return netip.Addr{}, ErrDenied
		}
	}
	if len(p.AllowedClientCIDRs) > 0 && !contains(p.AllowedClientCIDRs, client) {
		return netip.Addr{}, ErrDenied
	}
	return client, nil
}

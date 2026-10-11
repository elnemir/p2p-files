package config

import (
	"errors"
	"net/netip"
	"slices"
	"strings"
)

type NetworkPolicy struct {
	TrustedProxyCIDRs  []string `json:"trustedProxyCidrs,omitempty"`
	AllowedClientCIDRs []string `json:"allowedClientCidrs,omitempty"`
	RequireProxy       bool     `json:"requireProxy,omitempty"`
	RequireHTTPS       bool     `json:"requireHttps,omitempty"`
}

func (p NetworkPolicy) Equal(other NetworkPolicy) bool {
	return p.RequireProxy == other.RequireProxy && p.RequireHTTPS == other.RequireHTTPS && slices.Equal(p.TrustedProxyCIDRs, other.TrustedProxyCIDRs) && slices.Equal(p.AllowedClientCIDRs, other.AllowedClientCIDRs)
}

func ValidateNetworkPolicy(s Settings) error {
	p := s.NetworkPolicy
	invalid := errors.New("invalid network policy")
	if len(p.TrustedProxyCIDRs) > 16 || len(p.AllowedClientCIDRs) > 128 || (p.RequireProxy && len(p.TrustedProxyCIDRs) == 0) || (p.RequireHTTPS && (!p.RequireProxy || !strings.HasPrefix(s.PublicOrigin, "https://"))) {
		return invalid
	}
	for group, values := range [][]string{p.TrustedProxyCIDRs, p.AllowedClientCIDRs} {
		seen := map[string]bool{}
		for _, raw := range values {
			prefix, e := netip.ParsePrefix(raw)
			if e != nil || !prefix.Addr().Is4() || prefix.Masked() != prefix || prefix.String() != raw || seen[raw] {
				return invalid
			}
			// Only the designated proxy host may assert a client address.
			if group == 0 && (prefix.Bits() != 32 || prefix.Addr().IsUnspecified() || prefix.Addr().IsMulticast()) {
				return invalid
			}
			seen[raw] = true
		}
	}
	return nil
}

// IngressReady checks settings only; it does not authorize a production release.
func IngressReady(s Settings) error {
	if e := ValidateNetworkPolicy(s); e != nil {
		return e
	}
	if !s.NetworkPolicy.RequireProxy || !s.NetworkPolicy.RequireHTTPS || !strings.HasPrefix(s.PublicOrigin, "https://") {
		return errors.New("HTTPS and a required trusted proxy must be configured")
	}
	return nil
}

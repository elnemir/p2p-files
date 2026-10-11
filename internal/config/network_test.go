package config

import (
	"encoding/json"
	"testing"
)

func TestNetworkPolicyValidation(t *testing.T) {
	valid := Settings{Version: 1, PublicOrigin: "https://files.example.test", TransferMaxTTLSeconds: 600, MaxFiles: 10, MaxPathBytes: 1024, NetworkPolicy: NetworkPolicy{TrustedProxyCIDRs: []string{"10.0.0.2/32"}, AllowedClientCIDRs: []string{"203.0.113.0/24"}, RequireProxy: true, RequireHTTPS: true}}
	raw, _ := json.Marshal(valid)
	if s, e := ParseSettings(raw); e != nil || IngressReady(s) != nil {
		t.Fatal("valid ingress settings rejected", e)
	}
	for _, p := range []NetworkPolicy{
		{TrustedProxyCIDRs: []string{"0.0.0.0/0"}},
		{TrustedProxyCIDRs: []string{"10.0.0.0/24"}},
		{TrustedProxyCIDRs: []string{"10.0.0.2"}},
		{TrustedProxyCIDRs: []string{"10.0.0.2/32", "10.0.0.2/32"}},
		{TrustedProxyCIDRs: []string{"0.0.0.0/32"}},
		{AllowedClientCIDRs: []string{"203.0.113.5/24"}},
		{AllowedClientCIDRs: []string{"::/0"}},
		{AllowedClientCIDRs: []string{"invalid"}},
		{RequireProxy: true}, {RequireHTTPS: true},
	} {
		s := valid
		s.NetworkPolicy = p
		raw, _ := json.Marshal(s)
		if _, e := ParseSettings(raw); e == nil {
			t.Fatalf("invalid policy accepted: %+v", p)
		}
	}
	valid.PublicOrigin = "http://localhost:8090"
	if e := ValidateNetworkPolicy(valid); e == nil {
		t.Fatal("HTTPS policy accepted HTTP origin")
	}
	if e := IngressReady(Settings{}); e == nil {
		t.Fatal("unconfigured ingress accepted")
	}
}

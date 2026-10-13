package identity

import (
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func oidcConfig() Config {
	return Config{Name: "Corporate", Protocol: "oidc", Enabled: true, AllowRegistration: true, Issuer: "https://idp.test", ClientID: "client", EmailAttribute: "email", AllowedEmailDomains: []string{}, ExtraOrigins: []string{}, PrivateCIDRs: []string{}}
}
func TestConfigurationAndAuthority(t *testing.T) {
	c := oidcConfig()
	if e := c.Validate(""); e != nil {
		t.Fatal(e)
	}
	for _, issuer := range []string{"http://idp.test", "https://user:pass@idp.test", "https://idp.test/#x", "https://idp.test?x=1"} {
		bad := c
		bad.Issuer = issuer
		if bad.Validate("") == nil {
			t.Fatal("unsafe issuer", issuer)
		}
	}
	bad := c
	bad.TrustEmail = true
	if bad.Validate("") == nil {
		t.Fatal("directory trust without domains")
	}
	bad.AllowedEmailDomains = []string{"example.com"}
	if e := bad.Validate(""); e != nil {
		t.Fatal(e)
	}
	if c.SameAuthority(Config{Protocol: "oidc", Issuer: "https://evil.test", ClientID: "client"}) {
		t.Fatal("authority changed")
	}
	for _, cidr := range []string{"0.0.0.0/0", "127.0.0.1/32", "169.254.0.0/16", "10.0.0.1/8", "10.0.0.0/7"} {
		bad := c
		bad.PrivateCIDRs = []string{cidr}
		if bad.Validate("") == nil {
			t.Fatal("CIDR", cidr)
		}
	}
	c.AllowedEmailDomains = []string{"example.test"}
	for _, proof := range []Proof{{Subject: "", Email: "user@example.test"}, {Subject: "sub", Email: "user@evil.test"}, {Subject: strings.Repeat("x", 513), Email: "user@example.test"}} {
		if _, e := c.CheckProof(proof); e == nil {
			t.Fatal("invalid proof")
		}
	}
	proof, e := c.CheckProof(Proof{Subject: "stable", Email: "USER@example.test", AuthenticatedAt: time.Now().Unix()})
	if e != nil || proof.Email != "user@example.test" {
		t.Fatal(proof, e)
	}
}
func TestOutboundAddressPolicy(t *testing.T) {
	c := oidcConfig()
	c.PrivateCIDRs = []string{"10.12.0.0/16"}
	for _, address := range []string{"127.0.0.1", "::1", "0.0.0.0", "169.254.169.254", "100.100.100.200", "10.13.0.1", "192.168.1.1", "224.0.0.1"} {
		if c.allowedIP(net.ParseIP(address)) {
			t.Fatal("allowed forbidden address", address)
		}
	}
	for _, address := range []string{"10.12.3.4", "8.8.8.8"} {
		if !c.allowedIP(net.ParseIP(address)) {
			t.Fatal("denied configured address", address)
		}
	}
}

func TestConfigurationStrictJSONAndResponseBound(t *testing.T) {
	raw, err := json.Marshal(oidcConfig())
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err = json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(raw), `"enabled":true,`, "", 1), strings.Replace(string(raw), `"enabled":true`, `"enabled":null`, 1), strings.Replace(string(raw), `"enabled":true`, `"unknown":true,"enabled":true`, 1)} {
		if json.Unmarshal([]byte(bad), &cfg) == nil {
			t.Fatal("accepted invalid configuration")
		}
	}
	for _, size := range []int{maxResponse, maxResponse + 1} {
		b := &boundedBody{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", size))), remaining: maxResponse}
		raw, e := io.ReadAll(b)
		if len(raw) != maxResponse || (size == maxResponse && e != nil) || (size > maxResponse && e == nil) {
			t.Fatal("response bound", len(raw), e)
		}
	}
}

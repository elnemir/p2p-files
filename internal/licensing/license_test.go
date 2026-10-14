package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sample() Claims {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	end := now.AddDate(1, 0, 0)
	n := int64(10)
	return Claims{1, "filex", "filex-vendor", "b5582e28-44f0-43ea-8653-e35c9231e437", "fe675ab3-d9f2-4ea9-85c9-8865eae589e5", 1, "customer-42", "49ca5e21-4f82-4a02-93d9-a2f939c711f5", "subscription", now, now, &end, end, end, Limits{&n, &n}, []string{"webdav"}}
}
func TestSignatureTampering(t *testing.T) {
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	c := sample()
	s, e := Sign(c, "test", key)
	if e != nil {
		t.Fatal(e)
	}
	keys := map[string]ed25519.PublicKey{"test": pub}
	got, e := Verify(s, keys)
	if e != nil || got.CustomerID != c.CustomerID {
		t.Fatal(got, e)
	}
	for i := range len(s) {
		b := []byte(s)
		if b[i] == 'A' {
			b[i] = 'B'
		} else {
			b[i] = 'A'
		}
		if _, e = Verify(string(b), keys); e == nil {
			t.Fatalf("accepted tamper %d", i)
		}
	}
	if _, e = Verify(s, map[string]ed25519.PublicKey{}); e == nil {
		t.Fatal("unknown key")
	}
}
func TestSignedMalformedClaims(t *testing.T) {
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(sample())
	if e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{strings.Replace(string(raw), `"revision":`, `"Revision":`, 1), strings.Replace(string(raw), `"maxEnabledUsers":`, `"MaxEnabledUsers":`, 1), strings.Replace(string(raw), `"revision":1`, `"revision":1,"revision":99`, 1), strings.Replace(string(raw), `"maxEnabledUsers":10,`, "", 1), strings.Replace(string(raw), `"expiresAt":`, `"unknown":true,"expiresAt":`, 1), strings.Replace(string(raw), `"revision":1`, `"revision":9007199254740992`, 1)} {
		h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA","kid":"test","typ":"filex-license+jws"}`))
		input := h + "." + base64.RawURLEncoding.EncodeToString([]byte(bad))
		signed := input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(input)))
		if _, e = Verify(signed, map[string]ed25519.PublicKey{"test": pub}); e == nil {
			t.Fatal("malformed accepted")
		}
	}
}
func TestExpiryAndPerpetual(t *testing.T) {
	c := sample()
	if c.State(*c.ExpiresAt) != "expired" || c.State(c.NotBefore) != "active" || c.State(c.NotBefore.Add(-time.Second)) != "not_yet_valid" {
		t.Fatal("boundary")
	}
	c.Kind = "perpetual"
	c.ExpiresAt = nil
	if c.Validate() != nil || c.State(c.NotBefore.AddDate(100, 0, 0)) != "active" {
		t.Fatal("perpetual")
	}
}
func FuzzVerify(f *testing.F) {
	f.Add("a.b.c")
	f.Fuzz(func(t *testing.T, s string) { _, _ = Verify(s, map[string]ed25519.PublicKey{}) })
}

func TestReleaseEntitlementAndDomainSeparation(t *testing.T) {
	oldTrust, oldManifest, oldSource := ReleaseMetadataTrustJSON, ReleaseMetadataJWS, ReleaseSource
	defer func() { ReleaseMetadataTrustJSON, ReleaseMetadataJWS, ReleaseSource = oldTrust, oldManifest, oldSource }()
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := json.Marshal(map[string]string{"release": base64.StdEncoding.EncodeToString(pub)})
	if e != nil {
		t.Fatal(e)
	}
	ReleaseMetadataTrustJSON = string(raw)
	ReleaseSource = strings.Repeat("a", 64)
	c := sample()
	ReleaseMetadataJWS, e = SignRelease(Release{Version: "v1", Source: ReleaseSource, PublishedAt: c.UpdatesUntil}, "release", key)
	if e != nil || !UpdateEntitled(c) {
		t.Fatal(e)
	}
	if _, e = Verify(ReleaseMetadataJWS, map[string]ed25519.PublicKey{"release": pub}); e == nil {
		t.Fatal("release accepted as license")
	}
	c.UpdatesUntil = c.UpdatesUntil.Add(-time.Second)
	if UpdateEntitled(c) {
		t.Fatal("new release allowed")
	}
	ReleaseSource = strings.Repeat("b", 64)
	if _, e = BuildRelease(); e == nil {
		t.Fatal("wrong build source accepted")
	}
}

func FuzzClaims(f *testing.F) {
	raw, e := json.Marshal(sample())
	if e != nil {
		f.Fatal(e)
	}
	f.Add(string(raw))
	f.Add(`{"limits":null}`)
	f.Fuzz(func(t *testing.T, s string) { _, _ = ParseClaims([]byte(s)) })
}

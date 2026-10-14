package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCapabilityOriginExpiryAndDisabled(t *testing.T) {
	token := strings.Repeat("a", 43)
	sum := sha256.Sum256([]byte(token))
	a := Access{hex.EncodeToString(sum[:]), "https://files.test", time.Now().Add(time.Minute)}
	r := httptest.NewRequest("POST", "https://files.test/api/v1/setup", nil)
	r.Header.Set("Origin", a.Origin)
	r.Header.Set("X-Setup-Token", token)
	if !a.Allowed(r) {
		t.Fatal("valid capability rejected")
	}
	for _, origin := range []string{"", "https://evil.test", "https://files.test.evil.test", "null"} {
		r.Header.Set("Origin", origin)
		if a.Allowed(r) {
			t.Fatal("origin accepted")
		}
	}
	r.Header.Set("Origin", a.Origin)
	r.Header.Set("X-Setup-Token", token+"b")
	if a.Allowed(r) {
		t.Fatal("bad token accepted")
	}
	r.Header.Set("X-Setup-Token", token)
	a.Expires = time.Now().Add(-time.Second)
	if a.Allowed(r) {
		t.Fatal("expired accepted")
	}
	a.Expires = time.Now().Add(time.Minute)
	a.Hash = ""
	if a.Enabled() {
		t.Fatal("unconfigured enabled")
	}
}
func TestBoundedStrictSetupJSON(t *testing.T) {
	for _, raw := range []string{`{"unknown":1}`, `{"email":"x"} {}`, `{"email":"` + strings.Repeat("x", 16384) + `"}`} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		var in struct {
			Email string `json:"email"`
		}
		if Decode(httptest.NewRecorder(), r, &in) == nil {
			t.Fatal("malformed accepted")
		}
	}
}

package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func env() map[string]string {
	return map[string]string{"APP_ENV": "test", "DATABASE_URL": "postgres://user:secret@localhost:5432/filex", "MASTER_KEK": base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))}
}
func TestBootstrapFailClosed(t *testing.T) {
	valid := env()
	if _, err := Load(func(k string) string { return valid[k] }); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ k, v string }{{"APP_ENV", "production"}, {"APP_ENV", ""}, {"DATABASE_URL", "invalid-secret"}, {"MASTER_KEK", "bad-secret"}, {"HTTP_ADDR", "localhost:0"}, {"HTTP_ADDR", "127.0.0.1:9090"}} {
		e := env()
		e[tc.k] = tc.v
		if _, err := Load(func(k string) string { return e[k] }); err == nil {
			t.Fatalf("accepted invalid %s", tc.k)
		} else if strings.Contains(err.Error(), "secret") {
			t.Fatal("error leaks supplied credentials")
		}
	}
}
func TestTypedSettings(t *testing.T) {
	valid := `{"version":1,"transferMaxTtlSeconds":604800,"maxFiles":10000,"maxPathBytes":1024}`
	if _, err := ParseSettings([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{valid + " garbage", valid + " {}", strings.Replace(valid, `"version":1`, `"version":2`, 1), strings.Replace(valid, `"maxFiles":10000`, `"maxFiles":0`, 1), strings.Replace(valid, `"version":1`, `"unknown":1,"version":1`, 1), `{}`} {
		if _, err := ParseSettings([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid settings %s", raw)
		}
	}
}

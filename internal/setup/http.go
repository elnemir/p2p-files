// Package setup defines the opt-in, expiring host capability for first-run setup.
package setup

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Access struct {
	Hash, Origin string
	Expires      time.Time
}

func FromEnv(prefix string, getenv func(string) string, origin string) Access {
	expiry, _ := time.Parse(time.RFC3339, getenv(prefix+"SETUP_EXPIRES_AT"))
	return Access{getenv(prefix + "SETUP_TOKEN_HASH"), origin, expiry}
}
func (a Access) Enabled() bool {
	b, e := hex.DecodeString(a.Hash)
	u, ue := url.Parse(a.Origin)
	return e == nil && len(b) == 32 && time.Now().Before(a.Expires) && ue == nil && u.Host != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && (u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")))
}
func (a Access) Allowed(r *http.Request) bool {
	token := r.Header.Get("X-Setup-Token")
	sum := sha256.Sum256([]byte(token))
	expected, _ := hex.DecodeString(a.Hash)
	return a.Enabled() && len(token) >= 32 && len(token) <= 256 && r.Header.Get("Origin") == a.Origin && subtle.ConstantTimeCompare(sum[:], expected) == 1
}
func Decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return io.ErrUnexpectedEOF
	}
	if e := d.Decode(v); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func Reply(w http.ResponseWriter, code int, state string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"state": state})
}

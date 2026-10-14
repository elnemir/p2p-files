// Package licensing implements the vendor-only offline license signature profile.
package licensing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxBytes = 16384
const MaxInteger int64 = 9007199254740991

var ErrInvalid = errors.New("invalid license")
var ErrDenied = errors.New("license restricted")
var ErrConflict = errors.New("license revision conflict")

// ReleaseTrustJSON is injected at build time, never read from runtime settings.
// An unconfigured build trusts no issuer. Test fixtures replace it only in tests.
var ReleaseTrustJSON = "{}"

type Limits struct {
	Users *int64 `json:"maxEnabledUsers"`
	Bytes *int64 `json:"maxTransferBytes"`
}
type Claims struct {
	SchemaVersion  int        `json:"schemaVersion"`
	Product        string     `json:"product"`
	Issuer         string     `json:"issuer"`
	LicenseID      string     `json:"licenseId"`
	EntitlementID  string     `json:"entitlementId"`
	Revision       int64      `json:"revision"`
	CustomerID     string     `json:"customerId"`
	InstallationID string     `json:"installationId"`
	Kind           string     `json:"kind"`
	IssuedAt       time.Time  `json:"issuedAt"`
	NotBefore      time.Time  `json:"notBefore"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	UpdatesUntil   time.Time  `json:"updatesUntil"`
	SupportUntil   time.Time  `json:"supportUntil"`
	Limits         Limits     `json:"limits"`
	Features       []string   `json:"features"`
}
type header struct {
	Alg   string `json:"alg"`
	KeyID string `json:"kid"`
	Type  string `json:"typ"`
}

var identifier = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var label = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Decode rejects duplicate members recursively before typed decoding. A signed
// but ambiguous JSON object must not produce different rights in two parsers.
func Decode(raw []byte, dst any) error {
	if len(raw) > MaxBytes || !utf8.Valid(raw) {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 12 {
			return ErrInvalid
		}
		t, e := d.Token()
		if e != nil {
			return e
		}
		if delim, ok := t.(json.Delim); ok {
			switch delim {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return ErrInvalid
					}
					seen[key] = true
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
				t, e = d.Token()
				if e != nil || t != json.Delim('}') {
					return ErrInvalid
				}
			case '[':
				for d.More() {
					if e = walk(depth + 1); e != nil {
						return e
					}
				}
				t, e = d.Token()
				if e != nil || t != json.Delim(']') {
					return ErrInvalid
				}
			default:
				return ErrInvalid
			}
		}
		return nil
	}
	if e := walk(0); e != nil {
		return ErrInvalid
	}
	if _, e := d.Token(); e != io.EOF {
		return ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(dst); e != nil {
		return ErrInvalid
	}
	return nil
}
func (c Claims) Validate() error {
	if c.SchemaVersion != 1 || c.Product != "filex" || c.Issuer != "filex-vendor" || !identifier.MatchString(c.LicenseID) || !identifier.MatchString(c.EntitlementID) || !identifier.MatchString(c.InstallationID) || !label.MatchString(c.CustomerID) || c.Revision < 1 || c.Revision > MaxInteger {
		return ErrInvalid
	}
	for _, t := range []time.Time{c.IssuedAt, c.NotBefore, c.UpdatesUntil, c.SupportUntil} {
		if t.IsZero() || t.Location() != time.UTC || t.Year() > 9999 {
			return ErrInvalid
		}
	}
	if c.IssuedAt.After(c.NotBefore) || c.UpdatesUntil.Before(c.NotBefore) || c.SupportUntil.Before(c.IssuedAt) {
		return ErrInvalid
	}
	if c.Kind == "subscription" {
		if c.ExpiresAt == nil || c.ExpiresAt.Location() != time.UTC || !c.ExpiresAt.After(c.NotBefore) {
			return ErrInvalid
		}
	} else if c.Kind != "perpetual" || c.ExpiresAt != nil {
		return ErrInvalid
	}
	for _, limit := range []*int64{c.Limits.Users, c.Limits.Bytes} {
		if limit != nil && (*limit < 1 || *limit > MaxInteger) {
			return ErrInvalid
		}
	}
	if c.Features == nil || len(c.Features) > 3 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, f := range c.Features {
		if !slices.Contains([]string{"webdav", "federated_auth", "white_label"}, f) || seen[f] {
			return ErrInvalid
		}
		seen[f] = true
	}
	return nil
}
func ParseClaims(raw []byte) (Claims, error) {
	var c Claims
	if Decode(raw, &c) != nil {
		return c, ErrInvalid
	}
	// Explicit null and missing members have different meanings for limits.
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || !exactFields(fields, "schemaVersion", "product", "issuer", "licenseId", "entitlementId", "revision", "customerId", "installationId", "kind", "issuedAt", "notBefore", "expiresAt", "updatesUntil", "supportUntil", "limits", "features") {
		return c, ErrInvalid
	}
	var limits map[string]json.RawMessage
	if json.Unmarshal(fields["limits"], &limits) != nil || !exactFields(limits, "maxEnabledUsers", "maxTransferBytes") {
		return c, ErrInvalid
	}
	return c, c.Validate()
}

// encoding/json accepts case-insensitive struct fields; the wire profile does not.
func exactFields(fields map[string]json.RawMessage, names ...string) bool {
	if len(fields) != len(names) {
		return false
	}
	for _, name := range names {
		if _, ok := fields[name]; !ok {
			return false
		}
	}
	return true
}
func Sign(c Claims, kid string, key ed25519.PrivateKey) (string, error) {
	if c.Validate() != nil || !label.MatchString(kid) || len(key) != ed25519.PrivateKeySize {
		return "", ErrInvalid
	}
	h, e := json.Marshal(header{"EdDSA", kid, "filex-license+jws"})
	if e != nil {
		return "", e
	}
	p, e := json.Marshal(c)
	if e != nil {
		return "", e
	}
	input := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(input))), nil
}
func verifyEnvelope(envelope string, keys map[string]ed25519.PublicKey, typ string) ([]byte, error) {
	if len(envelope) > MaxBytes {
		return nil, ErrInvalid
	}
	parts := strings.Split(envelope, ".")
	if len(parts) != 3 {
		return nil, ErrInvalid
	}
	decoded := make([][]byte, 3)
	for i, p := range parts {
		b, e := base64.RawURLEncoding.Strict().DecodeString(p)
		if e != nil || len(b) == 0 || base64.RawURLEncoding.EncodeToString(b) != p {
			return nil, ErrInvalid
		}
		decoded[i] = b
	}
	var h header
	if Decode(decoded[0], &h) != nil || h.Alg != "EdDSA" || h.Type != typ || !label.MatchString(h.KeyID) {
		return nil, ErrInvalid
	}
	var headerFields map[string]json.RawMessage
	if json.Unmarshal(decoded[0], &headerFields) != nil || !exactFields(headerFields, "alg", "kid", "typ") {
		return nil, ErrInvalid
	}
	key := keys[h.KeyID]
	if len(key) != ed25519.PublicKeySize || len(decoded[2]) != ed25519.SignatureSize || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), decoded[2]) {
		return nil, ErrInvalid
	}
	return decoded[1], nil
}
func Verify(envelope string, keys map[string]ed25519.PublicKey) (Claims, error) {
	raw, e := verifyEnvelope(envelope, keys, "filex-license+jws")
	if e != nil {
		return Claims{}, e
	}
	return ParseClaims(raw)
}
func ReleaseKeys() (map[string]ed25519.PublicKey, error) {
	var encoded map[string]string
	if Decode([]byte(ReleaseTrustJSON), &encoded) != nil || encoded == nil {
		return nil, ErrInvalid
	}
	keys := map[string]ed25519.PublicKey{}
	for id, v := range encoded {
		b, e := base64.StdEncoding.Strict().DecodeString(v)
		if e != nil || len(b) != ed25519.PublicKeySize || !label.MatchString(id) {
			return nil, ErrInvalid
		}
		keys[id] = b
	}
	return keys, nil
}
func (c Claims) State(now time.Time) string {
	if now.Before(c.NotBefore) {
		return "not_yet_valid"
	}
	if c.ExpiresAt != nil && !now.Before(*c.ExpiresAt) {
		return "expired"
	}
	return "active"
}
func (c Claims) Allows(feature string) bool {
	return feature == "" || slices.Contains(c.Features, feature)
}

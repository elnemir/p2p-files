package licensing

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"time"
)

// Release metadata is signed separately from licenses and bound to this build's
// source digest. Runtime environment variables cannot override these values.
var ReleaseMetadataTrustJSON = "{}"
var ReleaseMetadataJWS = ""
var ReleaseSource = ""

type Release struct {
	Version     string    `json:"version"`
	Source      string    `json:"sourceSha256"`
	PublishedAt time.Time `json:"publishedAt"`
}

var sourcePattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (r Release) validate() error {
	if !label.MatchString(r.Version) || !sourcePattern.MatchString(r.Source) || r.PublishedAt.IsZero() || r.PublishedAt.Location() != time.UTC {
		return ErrInvalid
	}
	return nil
}
func SignRelease(r Release, kid string, key ed25519.PrivateKey) (string, error) {
	if r.validate() != nil || !label.MatchString(kid) || len(key) != ed25519.PrivateKeySize {
		return "", ErrInvalid
	}
	h, e := json.Marshal(header{"EdDSA", kid, "filex-release+jws"})
	if e != nil {
		return "", e
	}
	raw, e := json.Marshal(r)
	if e != nil {
		return "", e
	}
	input := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(raw)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(input))), nil
}
func BuildRelease() (Release, error) {
	var encoded map[string]string
	if Decode([]byte(ReleaseMetadataTrustJSON), &encoded) != nil {
		return Release{}, ErrInvalid
	}
	keys := map[string]ed25519.PublicKey{}
	for k, v := range encoded {
		b, e := base64.StdEncoding.Strict().DecodeString(v)
		if e != nil || len(b) != 32 {
			return Release{}, ErrInvalid
		}
		keys[k] = b
	}
	raw, e := verifyEnvelope(ReleaseMetadataJWS, keys, "filex-release+jws")
	if e != nil {
		return Release{}, e
	}
	var r Release
	if Decode(raw, &r) != nil || r.validate() != nil || r.Source != ReleaseSource {
		return r, ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || !exactFields(fields, "version", "sourceSha256", "publishedAt") {
		return r, ErrInvalid
	}
	return r, nil
}
func UpdateEntitled(c Claims) bool {
	r, e := BuildRelease()
	return e == nil && !r.PublishedAt.After(c.UpdatesUntil)
}

// CheckUpgrade runs before migration. An ineligible candidate cannot change the
// database and then require an unsafe binary/database downgrade to recover.
func CheckUpgrade(ctx context.Context, db *pgxpool.Pool) error {
	keys, e := ReleaseKeys()
	if e != nil {
		return e
	}
	if len(keys) == 0 {
		return nil
	}
	r, e := BuildRelease()
	if e != nil {
		return e
	}
	var exists bool
	if e = db.QueryRow(ctx, "SELECT to_regclass(quote_ident(current_schema())||'.ops_license') IS NOT NULL").Scan(&exists); e != nil {
		return e
	}
	if !exists {
		return nil
	}
	var raw, pending string
	if e = db.QueryRow(ctx, "SELECT envelope,pending_envelope FROM ops_license WHERE singleton").Scan(&raw, &pending); e != nil {
		return e
	}
	if raw == "" {
		return nil
	}
	c, e := Verify(raw, keys)
	if e != nil {
		return e
	}
	if !r.PublishedAt.After(c.UpdatesUntil) {
		return nil
	}
	if pending != "" {
		c, e = Verify(pending, keys)
		if e == nil && !time.Now().UTC().Before(c.NotBefore) && !r.PublishedAt.After(c.UpdatesUntil) {
			return nil
		}
	}
	return ErrDenied
}

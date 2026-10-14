package httpapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"testing"
	"time"
)

var licenseTestKey ed25519.PrivateKey

func init() {
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		panic(e)
	}
	licenseTestKey = key
	raw, e := json.Marshal(map[string]string{"integration-only": base64.StdEncoding.EncodeToString(pub)})
	if e != nil {
		panic(e)
	}
	licensing.ReleaseTrustJSON = string(raw)
	licensing.ReleaseMetadataTrustJSON = string(raw)
	licensing.ReleaseSource = strings.Repeat("a", 64)
	metadata, e := licensing.SignRelease(licensing.Release{Version: "test", Source: licensing.ReleaseSource, PublishedAt: time.Now().UTC().Add(-time.Hour)}, "integration-only", key)
	if e != nil {
		panic(e)
	}
	licensing.ReleaseMetadataJWS = metadata
}
func installTestLicense(t *testing.T, db *pgxpool.Pool) licensing.Claims {
	t.Helper()
	ctx := context.Background()
	var id string
	if e := db.QueryRow(ctx, "SELECT installation_id::text FROM ops_license").Scan(&id); e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC().Add(-time.Minute)
	end := now.AddDate(1, 0, 0)
	c := licensing.Claims{SchemaVersion: 1, Product: "filex", Issuer: "filex-vendor", LicenseID: domain.NewID(), EntitlementID: domain.NewID(), Revision: 1, CustomerID: "integration-only", InstallationID: id, Kind: "subscription", IssuedAt: now, NotBefore: now, ExpiresAt: &end, UpdatesUntil: end, SupportUntil: end, Features: []string{"webdav", "federated_auth", "white_label"}}
	raw, e := licensing.Sign(c, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "UPDATE ops_license SET envelope=$1,entitlement_id=$2,high_revision=1,epoch=1", raw, c.EntitlementID); e != nil {
		t.Fatal(e)
	}
	return c
}

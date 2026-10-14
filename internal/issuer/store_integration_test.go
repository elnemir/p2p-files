package issuer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("FILEX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires FILEX_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	schema := "issuer_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		db.Close()
		if _, e := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
		if e := admin.Close(ctx); e != nil {
			t.Error(e)
		}
	})
	if e = Migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	return db
}
func input() Input {
	now := time.Now().UTC().Add(-time.Minute)
	end := now.AddDate(1, 0, 0)
	return Input{domain.NewID(), licensing.Claims{SchemaVersion: 1, Product: "filex", Issuer: "filex-vendor", LicenseID: domain.NewID(), EntitlementID: domain.NewID(), Revision: 1, CustomerID: "fixture", InstallationID: domain.NewID(), Kind: "subscription", IssuedAt: now, NotBefore: now, ExpiresAt: &end, UpdatesUntil: end, SupportUntil: end, Features: []string{}}}
}
func TestIntegrationIssuerConcurrentIssueAndSignerFault(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	in := input()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, e := Issue(ctx, db, "operator", in); results <- e })
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	changed := in
	changed.Claims.CustomerID = "other"
	if _, e := Issue(ctx, db, "operator", changed); !errors.Is(e, licensing.ErrConflict) {
		t.Fatal(e)
	}
	_, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	// Fail the final audit insert: the queue claim and signature must roll back.
	if _, e = db.Exec(ctx, `CREATE FUNCTION fail_sign_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='issued' THEN RAISE EXCEPTION 'injected'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_sign BEFORE INSERT ON issuer_audit FOR EACH ROW EXECUTE FUNCTION fail_sign_audit()`); e != nil {
		t.Fatal(e)
	}
	if _, e = SignNext(ctx, db, "fixture", key); e == nil {
		t.Fatal("fault not propagated")
	}
	var state, envelope string
	if e = db.QueryRow(ctx, "SELECT state,envelope FROM issuer_issuances").Scan(&state, &envelope); e != nil || state != "approved" || envelope != "" {
		t.Fatal(state, e)
	}
	if _, e = db.Exec(ctx, "DROP TRIGGER fail_sign ON issuer_audit"); e != nil {
		t.Fatal(e)
	}
	if worked, e := SignNext(ctx, db, "fixture", key); e != nil || !worked {
		t.Fatal(worked, e)
	}
	if worked, e := SignNext(ctx, db, "fixture", key); e != nil || worked {
		t.Fatal("duplicate sign", worked, e)
	}
	if e = db.QueryRow(ctx, "SELECT envelope FROM issuer_issuances").Scan(&envelope); e != nil {
		t.Fatal(e)
	}
	if _, e = licensing.Verify(envelope, map[string]ed25519.PublicKey{"fixture": key.Public().(ed25519.PublicKey)}); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "UPDATE issuer_issuances SET claims='{}'"); e == nil {
		t.Fatal("immutable issuance changed")
	}
	if _, e = Issue(ctx, db, "operator", in); e != nil {
		t.Fatal("retry failed", e)
	}
}

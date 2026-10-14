package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"filex.local/filex/internal/transfer"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"
)

func TestIntegrationLicenseLifecycleAndQuotas(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	claims := installTestLicense(t, db)
	store := licensing.Store{DB: db, AuditKey: a.Key}
	owner, e := a.Auth.CreateUser(ctx, "host_cli", "license-owner@example.test", "strong-password-0123", "admin")
	if e != nil {
		t.Fatal(e)
	}
	one := int64(1)
	ten := int64(10)
	claims.Limits = licensing.Limits{Users: &one, Bytes: &ten}
	claims.Revision = 2
	claims.LicenseID = domain.NewID()
	signed, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	v, e := store.Apply(ctx, signed, 1, "host_cli", "", 0)
	if e != nil || v.Epoch != 2 {
		t.Fatal(v, e)
	}
	if _, e = a.Auth.CreateUser(ctx, "host_cli", "extra@example.test", "strong-password-0123", "user"); !errors.Is(e, licensing.ErrDenied) {
		t.Fatalf("seat bypass: %v", e)
	}
	settings, e := database.Settings(ctx, db)
	if e != nil {
		t.Fatal(e)
	}
	transfers := transfer.New(db, settings)
	draft, e := transfers.CreateDraft(ctx, owner)
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, path := range []string{"one.txt", "two.txt"} {
		wg.Go(func() { _, err := transfers.ReserveUpload(ctx, owner, draft.ID, path, 7); results <- err })
	}
	wg.Wait()
	close(results)
	won, denied := 0, 0
	for err := range results {
		if err == nil {
			won++
		} else if errors.Is(err, licensing.ErrDenied) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if won != 1 || denied != 1 {
		t.Fatal(won, denied)
	}
	if _, e = store.Apply(ctx, signed+"a", v.Epoch, "host_cli", "", 0); !errors.Is(e, licensing.ErrInvalid) {
		t.Fatal("tamper accepted", e)
	}
	current, e := store.Status(ctx)
	if e != nil || current.Epoch != v.Epoch || current.State != "active" {
		t.Fatal(current, e)
	}
	claims.Revision = 1
	signed, e = licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.Apply(ctx, signed, v.Epoch, "host_cli", "", 0); !errors.Is(e, licensing.ErrConflict) {
		t.Fatal("rollback accepted", e)
	}
	// Expiry is evaluated on the write boundary, not by a periodic scheduler.
	past := time.Now().UTC().Add(-time.Second)
	claims.Revision = 3
	claims.ExpiresAt = &past
	signed, e = licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "UPDATE ops_license SET envelope=$1", signed); e != nil {
		t.Fatal(e)
	}
	if _, e = transfers.CreateDraft(ctx, owner); !errors.Is(e, licensing.ErrDenied) {
		t.Fatal("expired draft accepted", e)
	}
	if _, e = db.Exec(ctx, "UPDATE ops_license SET max_observed_at=clock_timestamp()+interval '1 hour'"); e != nil {
		t.Fatal(e)
	}
	current, e = store.Status(ctx)
	if e != nil || current.State != "clock_suspect" {
		t.Fatal(current, e)
	}
}
func TestIntegrationLicenseConcurrentLastUser(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	claims := installTestLicense(t, db)
	one := int64(1)
	claims.Limits.Users = &one
	raw, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(ctx, "UPDATE ops_license SET envelope=$1", raw); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, email := range []string{"last-a@example.test", "last-b@example.test"} {
		wg.Go(func() {
			_, err := a.Auth.CreateUser(ctx, "host_cli", email, "strong-password-0123", "user")
			results <- err
		})
	}
	wg.Wait()
	close(results)
	won, denied := 0, 0
	for err := range results {
		if err == nil {
			won++
		} else if errors.Is(err, licensing.ErrDenied) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if won != 1 || denied != 1 {
		t.Fatal(won, denied)
	}
}
func TestIntegrationLicenseBootstrapAndFutureRenewal(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	claims := installTestLicense(t, db)
	if _, e := db.Exec(ctx, "UPDATE ops_license SET envelope='',entitlement_id='',high_revision=0,epoch=0"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Auth.CreateUser(ctx, "host_cli", "blocked@example.test", "strong-password-0123", "user"); !errors.Is(e, licensing.ErrDenied) {
		t.Fatal(e)
	}
	if _, e := a.Auth.CreateUser(ctx, "host_cli", "bootstrap@example.test", "strong-password-0123", "admin"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Auth.CreateUser(ctx, "host_cli", "extra-admin@example.test", "strong-password-0123", "admin"); !errors.Is(e, licensing.ErrDenied) {
		t.Fatal(e)
	}
	store := licensing.Store{DB: db, AuditKey: a.Key}
	raw, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	v, e := store.Apply(ctx, raw, 0, "host_cli", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	claims.Revision = 2
	claims.LicenseID = domain.NewID()
	claims.NotBefore = time.Now().UTC().Add(time.Hour)
	raw, e = licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	v, e = store.Apply(ctx, raw, v.Epoch, "host_cli", "", 0)
	if e != nil || v.State != "active" || v.Pending == nil || v.Claims.Revision != 1 {
		t.Fatal(v, e)
	}
}

func TestIntegrationLicenseExpiryPreservesDelivery(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	c.upload(id, "keep.txt", "licensed delivery")
	token := c.publish(id)
	claims := installTestLicense(t, c.a.DB)
	past := time.Now().UTC().Add(-time.Second)
	claims.ExpiresAt = &past
	raw, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE ops_license SET envelope=$1", raw); e != nil {
		t.Fatal(e)
	}
	if w := c.req("POST", "/api/v1/transfers", "", nil); w.Code != 403 {
		t.Fatal("expired draft", w.Code, w.Body.String())
	}
	recipient := c.recipient(token)
	refs := recipient.refs()
	download := recipient.lease(`{"fileRef":"` + refs[0]["ref"].(string) + `"}`)
	w := recipient.req("GET", download, "", nil)
	if w.Code != 200 || w.Body.String() != "licensed delivery" {
		t.Fatal(w.Code, w.Body.String())
	}
	var state string
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM files WHERE transfer_id=$1", id).Scan(&state); e != nil || state != "deleting" {
		t.Fatal("one-time consume changed", state, e)
	}
	if w = c.req("GET", "/api/v1/admin/license", "", nil); w.Code != 200 {
		t.Fatal("recovery UI unavailable", w.Code)
	}
}
func TestIntegrationLicenseDAVReplacementAndFeature(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	cred := davCredential(t, c, id)
	claims := installTestLicense(t, c.a.DB)
	limit := int64(10)
	claims.Limits.Bytes = &limit
	apply := func() {
		t.Helper()
		raw, e := licensing.Sign(claims, "integration-only", licenseTestKey)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = c.a.DB.Exec(ctx, "UPDATE ops_license SET envelope=$1", raw); e != nil {
			t.Fatal(e)
		}
	}
	apply()
	davStatus(t, davRequest(c, cred, "PUT", "one.txt", "1234567", nil), 201)
	davStatus(t, davRequest(c, cred, "PUT", "one.txt", "abcdefgh", nil), 204)
	// A renamed path retains its quota; creating the old name cannot reuse it.
	davStatus(t, davRequest(c, cred, "MOVE", "one.txt", "", map[string]string{"Destination": cred.URL + "two.txt"}), 201)
	w := davRequest(c, cred, "PUT", "one.txt", "1234", nil)
	if w.Code != 403 {
		t.Fatal("renamed quota bypass", w.Code, w.Body.String())
	}
	claims.Features = []string{}
	apply()
	w = davRequest(c, cred, "PUT", "two.txt", "x", nil)
	if w.Code != 403 {
		t.Fatal("feature bypass", w.Code, w.Body.String())
	}
	davStatus(t, davRequest(c, cred, "GET", "two.txt", "", nil), 200)
	davStatus(t, davRequest(c, cred, "DELETE", "two.txt", "", nil), 204)
}

func TestIntegrationLicenseCorruptionRecoveryBeforeMigration(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	claims := installTestLicense(t, db)
	u, e := url.Parse(os.Getenv("FILEX_TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", db.Config().ConnConfig.RuntimeParams["search_path"])
	u.RawQuery = q.Encode()
	t.Setenv("DATABASE_URL", u.String())
	t.Setenv("APP_ENV", "test")
	t.Setenv("MASTER_KEK", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	if _, e = db.Exec(ctx, "UPDATE ops_license SET envelope='tampered'"); e != nil {
		t.Fatal(e)
	}
	_, recovered, e := bootstrap.Open(ctx)
	if e != nil {
		t.Fatal("matching schema must permit license recovery", e)
	}
	defer recovered.Close()
	store := licensing.Store{DB: recovered, AuditKey: a.Key}
	v, e := store.Status(ctx)
	if e != nil || v.State != "invalid" {
		t.Fatal(v, e)
	}
	claims.Revision = 2
	claims.LicenseID = domain.NewID()
	signed, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	v, e = store.Apply(ctx, signed, v.Epoch, "host_cli", "", 0)
	if e != nil || v.State != "active" {
		t.Fatal(v, e)
	}
	// Simulate a pending migration: a corrupt license must not reach DDL.
	if _, e = db.Exec(ctx, "UPDATE ops_license SET envelope='tampered'; DELETE FROM schema_migrations WHERE name='0021_first_run_setup.sql'"); e != nil {
		t.Fatal(e)
	}
	if _, p, e := bootstrap.Open(ctx); e == nil {
		p.Close()
		t.Fatal("unlicensed migration accepted")
	} else if !errors.Is(e, licensing.ErrInvalid) {
		t.Fatal("expected license verifier rejection before DDL", e)
	}
	var count int
	if e = db.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); e != nil || count != 20 {
		t.Fatal(count, e)
	}
}

func TestIntegrationLicenseHighWaterSurvivesRestore(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	claims := installTestLicense(t, c.a.DB)
	store := licensing.Store{DB: c.a.DB, AuditKey: c.a.Key}
	previous, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	provider, e := c.a.Files.Storage.Provision(ctx, t.TempDir(), "backups")
	if e != nil {
		t.Fatal(e)
	}
	backup := operationsJob(t, c, "backup", "", provider)
	if e = c.a.Operations.Execute(ctx, backup, nil); e != nil {
		t.Fatal(e)
	}
	claims.Revision = 2
	claims.LicenseID = domain.NewID()
	next, e := licensing.Sign(claims, "integration-only", licenseTestKey)
	if e != nil {
		t.Fatal(e)
	}
	before, e := store.Apply(ctx, next, 1, "host_cli", "", 0)
	if e != nil {
		t.Fatal(e)
	}
	var observed time.Time
	if e = c.a.DB.QueryRow(ctx, "SELECT max_observed_at FROM ops_license").Scan(&observed); e != nil {
		t.Fatal(e)
	}
	restored := operationsJob(t, c, "restore", backup, provider)
	if e = c.a.Operations.Execute(ctx, restored, nil); e != nil {
		t.Fatal(e)
	}
	after, e := store.Status(ctx)
	if e != nil || after.InstallationID != before.InstallationID || after.Epoch != before.Epoch || after.Claims == nil || after.Claims.Revision != 2 {
		t.Fatal("restore rolled back license", after, e)
	}
	var high int64
	var now time.Time
	if e = c.a.DB.QueryRow(ctx, "SELECT high_revision,max_observed_at FROM ops_license").Scan(&high, &now); e != nil || high != 2 || now.Before(observed) {
		t.Fatal("restore rolled back high water", high, e)
	}
	if _, e = store.Apply(ctx, previous, after.Epoch, "host_cli", "", 0); !errors.Is(e, licensing.ErrConflict) {
		t.Fatal("old license accepted after restore", e)
	}
}

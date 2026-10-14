package issuer

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strings"
	"testing"
)

func TestIntegrationIssuerDatabaseRoleIsolation(t *testing.T) {
	url := os.Getenv("FILEX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires FILEX_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, e := pgx.Connect(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(ctx)
	name := "issuer_roles_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE DATABASE "+name); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); e != nil {
			t.Error(e)
		}
	}()
	cfg, e := pgxpool.ParseConfig(url)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.Database = name
	owner, e := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close()
	if e = Migrate(ctx, owner); e != nil {
		t.Fatal(e)
	}
	appPassword, signerPassword := domain.NewID(), domain.NewID()
	if e = ProvisionRoles(ctx, owner, appPassword, signerPassword); e != nil {
		t.Fatal(e)
	}
	connect := func(role, password string) *pgxpool.Pool {
		c := cfg.Copy()
		c.ConnConfig.User = role
		c.ConnConfig.Password = password
		p, e := pgxpool.NewWithConfig(ctx, c)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(p.Close)
		return p
	}
	app, signer := connect("filex_issuer_app", appPassword), connect("filex_issuer_signer", signerPassword)
	if e = saveSetup(ctx, app, bytes.Repeat([]byte{5}, 32), Config{Origin: "https://issuer.test"}); e != nil {
		t.Fatal("app cannot persist setup", e)
	}
	if _, e = signer.Exec(ctx, "SELECT config FROM issuer_setup"); e == nil {
		t.Fatal("signer can read identity configuration")
	}
	if _, e = app.Exec(ctx, "UPDATE issuer_setup SET config='bad'"); e == nil {
		t.Fatal("app can replace bootstrap identity")
	}
	in := input()
	if _, e = Issue(ctx, app, "fixture", in); e != nil {
		t.Fatal(e)
	}
	if _, e = app.Exec(ctx, "UPDATE issuer_issuances SET state='issued'"); e == nil {
		t.Fatal("web role can forge signed state")
	}
	if _, e = Issue(ctx, signer, "fixture", input()); e == nil {
		t.Fatal("signer can approve issuance")
	}
	if _, e = app.Exec(ctx, "CREATE TABLE issuer_bypass(id int)"); e == nil {
		t.Fatal("web role has DDL")
	}
	_, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	if worked, e := SignNext(ctx, signer, "fixture", key); e != nil || !worked {
		t.Fatal(worked, e)
	}
	if _, e = app.Exec(ctx, "DELETE FROM issuer_audit"); e == nil {
		t.Fatal("web can erase audit")
	}
	if _, e = signer.Exec(ctx, "SELECT * FROM issuer_sessions"); e == nil {
		t.Fatal("signer can read sessions")
	}
}

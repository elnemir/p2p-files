package transfer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setup(t *testing.T) (*Store, *pgxpool.Pool, string, string) {
	t.Helper()
	url := os.Getenv("FILEX_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("integration requires FILEX_TEST_DATABASE_URL; make test-integration enforces it")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close(ctx)
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 12
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p.Close()
		_, e := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		if e != nil {
			t.Error(e)
		}
		if e = admin.Close(ctx); e != nil {
			t.Error(e)
		}
	})
	if err = database.Migrate(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, p); err != nil {
		t.Fatal("reapply:", err)
	}
	settings, err := database.Settings(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	installTestLicense(t, p)
	owner, other := domain.NewID(), domain.NewID()
	if _, err = p.Exec(ctx, "INSERT INTO users(id,email,role) VALUES($1,'a@example.test','user'),($2,'b@example.test','user')", owner, other); err != nil {
		t.Fatal(err)
	}
	return New(p, settings), p, owner, other
}
func ready(t *testing.T, s *Store, owner string) (Draft, Upload, int64) {
	t.Helper()
	ctx := context.Background()
	d, err := s.CreateDraft(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.ReserveUpload(ctx, owner, d.ID, "docs/report.txt", 12)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BeginFinalize(ctx, owner, d.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	rev, err := s.Finalize(ctx, owner, d.ID, u.ID, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return d, u, rev
}
func TestIntegrationPublishOwnershipAndImmutability(t *testing.T) {
	s, p, owner, other := setup(t)
	ctx := context.Background()
	d, u, rev := ready(t, s, owner)
	if _, err := s.Publish(ctx, other, d.ID, rev, 60); !errors.Is(err, ErrNotFound) {
		t.Fatal("ownership bypass", err)
	}
	if _, err := s.ReserveUpload(ctx, other, d.ID, "x", 0); !errors.Is(err, ErrNotFound) {
		t.Fatal("reserve ownership bypass", err)
	}
	if _, err := s.Publish(ctx, owner, d.ID, rev-1, 60); !errors.Is(err, ErrConflict) {
		t.Fatal("stale revision", err)
	}
	pub, err := s.Publish(ctx, owner, d.ID, rev, 60)
	if err != nil {
		t.Fatal(err)
	}
	if pub.ExpiresAt.Sub(pub.PublishedAt) != time.Minute || pub.ManifestHash == "" {
		t.Fatal("publication metadata")
	}
	if _, err = s.Publish(ctx, owner, d.ID, rev, 61); !errors.Is(err, ErrConflict) {
		t.Fatal("idempotency changed ttl", err)
	}
	if _, err = s.ReserveUpload(ctx, owner, d.ID, "other.txt", 0); !errors.Is(err, ErrConflict) {
		t.Fatal("mutation after publish", err)
	}
	if _, err = s.Finalize(ctx, owner, d.ID, u.ID, strings.Repeat("b", 64)); !errors.Is(err, ErrConflict) {
		t.Fatal("checksum mutation", err)
	}
	for _, query := range []string{"UPDATE files SET path='changed' WHERE transfer_id=$1", "DELETE FROM files WHERE transfer_id=$1", "UPDATE transfers SET manifest='bad'::bytea WHERE id=$1", "UPDATE transfers SET state='draft' WHERE id=$1"} {
		if _, err = p.Exec(ctx, query, d.ID); err == nil {
			t.Fatal("DB immutability bypass", query)
		}
	}
	var count int
	if err = p.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE aggregate_id=$1", d.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("missing atomic outbox", err, count)
	}
}
func TestIntegrationConcurrentPublishIsIdempotent(t *testing.T) {
	s, p, owner, _ := setup(t)
	d, _, rev := ready(t, s, owner)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan Published, 24)
	errs := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); pub, err := s.Publish(ctx, owner, d.ID, rev, 60); results <- pub; errs <- err }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first *Published
	for p := range results {
		if first == nil {
			first = &p
		} else if !bytes.Equal(first.Manifest, p.Manifest) || first.ManifestHash != p.ManifestHash || !first.PublishedAt.Equal(p.PublishedAt) {
			t.Fatal("concurrent publish differed")
		}
	}
	var count int
	if err := p.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE aggregate_id=$1", d.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate publication event", err, count)
	}
}
func TestIntegrationOutboxFailureRollsBackPublish(t *testing.T) {
	s, p, owner, _ := setup(t)
	d, _, rev := ready(t, s, owner)
	ctx := context.Background()
	_, err := p.Exec(ctx, `CREATE FUNCTION fail_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected storage failure'; END $$; CREATE TRIGGER fail_event BEFORE INSERT ON outbox FOR EACH ROW EXECUTE FUNCTION fail_outbox();`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Publish(ctx, owner, d.ID, rev, 60); err == nil {
		t.Fatal("fault did not abort")
	}
	var state string
	var manifest []byte
	if err = p.QueryRow(ctx, "SELECT state,manifest FROM transfers WHERE id=$1", d.ID).Scan(&state, &manifest); err != nil || state != "draft" || manifest != nil {
		t.Fatal("partial publish committed", err, state)
	}
	if _, err = p.Exec(ctx, "DROP TRIGGER fail_event ON outbox"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Publish(ctx, owner, d.ID, rev, 60); err != nil {
		t.Fatal("retry failed", err)
	}
}
func TestIntegrationDraftValidation(t *testing.T) {
	s, _, owner, _ := setup(t)
	ctx := context.Background()
	d, err := s.CreateDraft(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Publish(ctx, owner, d.ID, 1, 60); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("empty publish", err)
	}
	u, err := s.ReserveUpload(ctx, owner, d.ID, "A.txt", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Publish(ctx, owner, d.ID, u.Revision, 60); !errors.Is(err, ErrConflict) {
		t.Fatal("unfinalized publish", err)
	}
	for _, path := range []string{"a.txt", "A.txt/child"} {
		if _, err = s.ReserveUpload(ctx, owner, d.ID, path, 0); !errors.Is(err, ErrConflict) {
			t.Fatal("path collision", err)
		}
	}
	if _, err = s.Finalize(ctx, owner, d.ID, u.ID, "bad"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatal("invalid hash", err)
	}
	if _, err = s.Finalize(ctx, owner, d.ID, u.ID, strings.Repeat("b", 64)); !errors.Is(err, ErrConflict) {
		t.Fatal("skipped finalizing state", err)
	}
	if _, err = s.BeginFinalize(ctx, owner, d.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	rev, err := s.Finalize(ctx, owner, d.ID, u.ID, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.Finalize(ctx, owner, d.ID, u.ID, strings.Repeat("b", 64))
	if err != nil || rev != same {
		t.Fatal("finalize not idempotent", err)
	}
}
func TestIntegrationKeyAndMigrationFailClosed(t *testing.T) {
	_, p, _, _ := setup(t)
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	if err := database.CheckKEK(ctx, p, key); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckKEK(ctx, p, key); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckKEK(ctx, p, bytes.Repeat([]byte{8}, 32)); err == nil {
		t.Fatal("wrong KEK accepted")
	}
	if _, err := p.Exec(ctx, "UPDATE schema_migrations SET sha256='tampered'"); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, p); err == nil {
		t.Fatal("edited migration accepted")
	}
}

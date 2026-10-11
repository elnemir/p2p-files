package httpapi

import (
	"context"
	"errors"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/operations"
	"testing"
	"time"
)

func TestIntegrationUpgradePrepareDrainAndReopen(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	transfer := c.draft()
	c.upload(transfer, "retained.txt", "payload survives upgrade")
	state, e := c.a.Operations.State(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Operations.PrepareUpgrade(ctx, "host_cli", "bad"); !errors.Is(e, operations.ErrInvalid) {
		t.Fatal("reason validation", e)
	}
	job, e := c.a.Operations.Queue(ctx, "host_cli", "audit", "", "", "Pending work blocks preparation")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Operations.PrepareUpgrade(ctx, "host_cli", "Do not interrupt queued recovery"); !errors.Is(e, operations.ErrConflict) {
		t.Fatal("pending job admitted", e)
	}
	unchanged, _ := c.a.Operations.State(ctx)
	if unchanged.Generation != state.Generation || unchanged.Mode != "open" {
		t.Fatal("rejected prepare changed runtime")
	}
	if e = c.a.Operations.Execute(ctx, job, nil); e != nil {
		t.Fatal(e)
	}
	// Keep an external effect alive even after its context is cancelled. The
	// preparation must wait for release, not merely signal cancellation.
	active, release, e := c.a.Gate.Acquire(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	done := make(chan error, 1)
	go func() {
		out, e := c.a.Operations.PrepareUpgrade(ctx, "host_cli", "Controlled upgrade preparation")
		if e == nil && (!out.Drained || !out.RequiresRestart || out.Mode != "maintenance" || out.Generation == state.Generation) {
			e = errors.New("invalid preparation result")
		}
		done <- e
	}()
	select {
	case <-active.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("privileged request not cancelled")
	}
	select {
	case e := <-done:
		t.Fatal("prepare did not drain external effects", e)
	default:
	}
	release()
	if e = <-done; e != nil {
		t.Fatal("drain", e)
	}
	if w := c.req("GET", "/api/v1/auth/me", "", nil); w.Code != 503 {
		t.Fatal("old API resumed", w.Code)
	}
	if _, e = c.a.Operations.PrepareUpgrade(ctx, "host_cli", "Stale CLI must fail"); !errors.Is(e, maintenance.ErrRestart) {
		t.Fatal("stale CLI admitted", e)
	}
	a, e := New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(a.Close)
	c.a, c.h = a, a.Handler()
	if w := c.req("GET", "/api/v1/transfers", "", nil); w.Code != 503 {
		t.Fatal("restart reopened data plane", w.Code)
	}
	report, e := a.Operations.Reconcile(ctx, "host_cli", "Review post-upgrade state")
	if e != nil || !report.Passed || len(report.Findings) != 0 {
		t.Fatal("reconciliation", report, e)
	}
	if e = a.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Explicit post-upgrade reopening", true); e != nil {
		t.Fatal(e)
	}
	if w := c.req("GET", "/api/v1/transfers/"+transfer, "", nil); w.Code != 200 {
		t.Fatal("session or data lost", w.Code, w.Body.String())
	}
	if e = a.Operations.CheckAudit(ctx); e != nil {
		t.Fatal("audit", e)
	}
	var count int
	if e = a.DB.QueryRow(ctx, "SELECT count(*) FROM ops_events WHERE convert_from(payload,'UTF8')::jsonb->>'action' IN ('upgrade.preparing','upgrade.prepared')").Scan(&count); e != nil || count != 2 {
		t.Fatal("missing upgrade audit", count, e)
	}
}

func TestIntegrationUpgradePrepareTimeoutStaysClosed(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	active, release, e := a.Gate.Acquire(ctx, true)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	short, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	_, e = a.Operations.PrepareUpgrade(short, "host_cli", "Timeout while draining an external effect")
	cancel()
	if e == nil {
		t.Fatal("prepare bypassed drain")
	}
	if active.Err() == nil {
		t.Fatal("active context survived fence")
	}
	release()
	s, e := operations.New(ctx, db, a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	state, e := s.State(ctx)
	if e != nil || state.Mode == "open" || state.Generation == a.Gate.Generation {
		t.Fatal("timeout reopened or unfenced runtime", state, e)
	}
	if out, e := s.PrepareUpgrade(ctx, "host_cli", "Retry preparation after operator repair"); e != nil || !out.Drained {
		t.Fatal("retry", out, e)
	}
}

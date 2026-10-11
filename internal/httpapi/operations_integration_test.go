package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/maintenance"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func operationsJob(t *testing.T, c apiClient, kind, source, provider string) string {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"kind": kind, "reason": "Integration recovery drill"})
	var data map[string]any
	_ = json.Unmarshal(raw, &data)
	if source != "" {
		data["sourceId"] = source
	}
	if provider != "" {
		data["providerId"] = provider
	}
	raw, _ = json.Marshal(data)
	w := c.req("POST", "/api/v1/admin/operations/jobs", string(raw), nil)
	if w.Code != 202 {
		t.Fatal("queue", w.Code, w.Body.String())
	}
	var v struct{ ID string }
	decode(t, w, &v)
	return v.ID
}
func restartRecoveryClient(t *testing.T, c apiClient) apiClient {
	t.Helper()
	a, e := New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(a.Close)
	old := c
	c.a = a
	c.h = a.Handler()
	c.cookie = nil
	c.csrf = ""
	if w := old.req("GET", "/api/v1/auth/me", "", nil); w.Code != 503 {
		t.Fatal("old runtime resumed", w.Code)
	}
	old.h = c.h
	if w := old.req("GET", "/api/v1/auth/me", "", nil); w.Code != 401 {
		t.Fatal("old web session survived", w.Code)
	}
	w := c.req("POST", "/api/v1/auth/login", `{"email":"sender@example.test","password":"strong-password-0123"}`, nil)
	if w.Code != 200 {
		t.Fatal("recovery login", w.Code, w.Body.String())
	}
	var v struct {
		CSRF string `json:"csrfToken"`
	}
	decode(t, w, &v)
	c.cookie = w.Result().Cookies()[0]
	c.csrf = v.CSRF
	return c
}
func TestIntegrationRecoveryBackupFaultAndReopen(t *testing.T) {
	c, root := testClient(t)
	ctx := context.Background()
	backupRoot := t.TempDir()
	provider, e := c.a.Files.Storage.Provision(ctx, backupRoot, "backups")
	if e != nil {
		t.Fatal(e)
	}
	transfer := c.draft()
	file := c.upload(transfer, "retained.txt", "original private payload")
	capability := c.publish(transfer)
	recipient := c.recipient(capability)
	backup := operationsJob(t, c, "backup", "", provider)
	if e = c.a.Operations.Execute(ctx, backup, nil); e != nil {
		t.Fatal("backup", e)
	}
	entries, e := os.ReadDir(backupRoot)
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		b, e := os.ReadFile(filepath.Join(backupRoot, entry.Name()))
		if e != nil {
			t.Fatal(e)
		}
		if bytes.Contains(b, []byte("sender@example.test")) || bytes.Contains(b, []byte("original private payload")) {
			t.Fatal("plaintext backup leak")
		}
	}
	newer := c.draft()
	c.upload(newer, "after.txt", "newer orphan payload")
	var object string
	if e = c.a.DB.QueryRow(ctx, "SELECT object_key FROM file_chunks WHERE file_id=$1", file).Scan(&object); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(root, object)); e != nil {
		t.Fatal(e)
	}
	// A missing target cannot satisfy the mandatory pre-restore backup.
	unavailable := t.TempDir()
	bad, e := c.a.Files.Storage.Provision(ctx, unavailable, "backups")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(unavailable); e != nil {
		t.Fatal(e)
	}
	failed := operationsJob(t, c, "restore", backup, bad)
	if e = c.a.Operations.Execute(ctx, failed, nil); e == nil {
		t.Fatal("restore proceeded without pre-backup")
	}
	var exists bool
	if e = c.a.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM transfers WHERE id=$1)", newer).Scan(&exists); e != nil || !exists {
		t.Fatal("failed restore changed data", e)
	}
	if w := c.req("GET", "/api/v1/transfers", "", nil); w.Code != 503 {
		t.Fatal("gate open after failure", w.Code)
	}
	if w := c.req("POST", "/api/v1/admin/operations/reopen", fmt.Sprintf(`{"reconciliationId":%q,"hash":%q,"reason":"No reconciliation exists","acknowledgeCapabilities":true}`, domain.NewID(), string(bytes.Repeat([]byte{'0'}, 64))), nil); w.Code != 409 {
		t.Fatal("premature reopen", w.Code)
	}
	restored := operationsJob(t, c, "restore", backup, provider)
	if w := c.req("POST", "/api/v1/admin/operations/jobs", fmt.Sprintf(`{"kind":"restore","sourceId":%q,"providerId":%q,"reason":"Concurrent restore must be rejected"}`, backup, provider), nil); w.Code != 409 {
		t.Fatal("concurrent restore admitted", w.Code, w.Body.String())
	}
	if e = c.a.Operations.Execute(ctx, restored, nil); e != nil {
		t.Fatal("restore", e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM transfers WHERE id=$1)", newer).Scan(&exists); e != nil || exists {
		t.Fatal("new data survived replacement", e)
	}
	var pre string
	if e = c.a.DB.QueryRow(ctx, "SELECT pre_backup_id::text FROM ops_jobs WHERE id=$1", restored).Scan(&pre); e != nil || !domain.ValidID(pre) {
		t.Fatal("missing pre-backup", e)
	}
	c = restartRecoveryClient(t, c)
	recipient.h = c.h
	if w := recipient.req("GET", "/api/v1/recipient/files", "", nil); w.Code != 503 {
		t.Fatal("recipient gate bypass", w.Code)
	}
	report, e := c.a.Operations.Reconcile(ctx, "host_cli", "Verify restored storage and keys")
	if e != nil {
		t.Fatal("reconciliation", e)
	}
	if !report.Passed {
		t.Fatal("unexpected critical findings", report.Findings)
	}
	missing, orphan := false, false
	for _, f := range report.Findings {
		missing = missing || f.Kind == "storage_missing"
		orphan = orphan || f.Kind == "orphan_object"
	}
	if !missing || !orphan {
		t.Fatal("reconciliation missed storage drift", report.Findings)
	}
	if e = c.a.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Review capabilities and storage warnings", false); e == nil {
		t.Fatal("reopened without explicit acknowledgement")
	}
	if e = c.a.Operations.RevokeCapabilities(ctx, "host_cli", "Revoke restored capabilities after review"); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Reject stale findings after capability changes", true); e == nil {
		t.Fatal("stale reconciliation accepted")
	}
	report, e = c.a.Operations.Reconcile(ctx, "host_cli", "Verify revocation and refresh findings")
	if e != nil || !report.Passed || report.Links != 0 || report.Tokens != 0 {
		t.Fatal("capability reconciliation", e, report)
	}
	if e = c.a.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Review capabilities and storage warnings", true); e != nil {
		t.Fatal("reopen", e)
	}
	if w := recipient.req("GET", "/api/v1/recipient/files", "", nil); w.Code != 401 {
		t.Fatal("restored recipient session survived", w.Code, w.Body.String())
	}
	if e = audit.Verify(ctx, c.a.DB, c.a.Key); e != nil {
		t.Fatal("audit after restore", e)
	}
	if w := c.req("GET", "/api/v1/admin/operations", "", nil); w.Code != 200 {
		t.Fatal("dashboard", w.Code, w.Body.String())
	}
}
func TestIntegrationOperationsAuditCheckpoint(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	w := c.req("POST", "/api/v1/admin/operations/checkpoints", `{"kind":"regular","reason":"Retain independent audit anchor"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE audit_events SET action='tampered' WHERE id=(SELECT min(id) FROM audit_events)"); e != nil {
		t.Fatal(e)
	}
	if e := c.a.Operations.CheckAudit(ctx); e == nil {
		t.Fatal("audit tampering not detected")
	}
	state, e := c.a.Operations.State(ctx)
	if e != nil || state.Mode != "blocked" {
		t.Fatal("critical audit gate", e, state)
	}
	list, e := c.a.Operations.Incidents(ctx)
	if e != nil || len(list) != 1 {
		t.Fatal("incident dedup", e, list)
	}
	if e = c.a.Operations.CheckAudit(ctx); e == nil {
		t.Fatal("corruption disappeared")
	}
	again, e := c.a.Operations.Incidents(ctx)
	if e != nil || len(again) != 1 {
		t.Fatal("incident duplication", e)
	}
	raw, digest, e := c.a.Operations.IncidentReport(ctx, list[0].ID, 1)
	if e != nil || len(raw) == 0 || len(digest) != 64 {
		t.Fatal("immutable report", e)
	}
	if e = c.a.Operations.UpdateIncident(ctx, list[0].ID, "resolved", "host_cli", "Investigate the original chain first"); e == nil {
		t.Fatal("resolved uncontained audit corruption")
	}
	evidence, e := c.a.Operations.AuditEvidence(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Operations.Checkpoint(ctx, "host_cli", "recovery", "Evidence preserved outside the database", "bad hash"); e == nil {
		t.Fatal("stale checkpoint accepted")
	}
	if _, e = c.a.Operations.Checkpoint(ctx, "host_cli", "recovery", "Evidence preserved outside the database", evidence); e != nil {
		t.Fatal("recovery checkpoint", e)
	}
	if e = audit.Verify(ctx, c.a.DB, c.a.Key); e != nil {
		t.Fatal("new audit segment", e)
	}
	if e = audit.VerifyFull(ctx, c.a.DB, c.a.Key); e == nil {
		t.Fatal("damaged history was erased")
	}
	if e = c.a.Operations.UpdateIncident(ctx, list[0].ID, "resolved", "host_cli", "Reviewed preserved evidence and new checkpoint"); e != nil {
		t.Fatal(e)
	}
	retained, hash, e := c.a.Operations.IncidentReport(ctx, list[0].ID, 1)
	if e != nil || hash != digest || !bytes.Equal(retained, raw) {
		t.Fatal("old report was modified", e)
	}
	w = c.req("GET", "/api/v1/admin/operations/incidents/"+list[0].ID+"/report?format=pdf", "", nil)
	if w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF")) {
		t.Fatal("security PDF", w.Code)
	}
	if dir := os.Getenv("FILEX_EVIDENCE_DIR"); dir != "" {
		if e := os.WriteFile(filepath.Join(dir, "security-incident.pdf"), w.Body.Bytes(), 0600); e != nil {
			t.Fatal(e)
		}
	}
	report, e := c.a.Operations.Reconcile(ctx, "host_cli", "Reconcile the reviewed audit checkpoint")
	if e != nil || !report.Passed {
		t.Fatal("reconciliation", e, report.Findings)
	}
	if e = c.a.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Explicitly reopen after incident review", true); e != nil {
		t.Fatal(e)
	}
}

func TestIntegrationKEKRewrapAndFreshDatabaseRecovery(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	root := t.TempDir()
	provider, e := c.a.Files.Storage.Provision(ctx, root, "backups")
	if e != nil {
		t.Fatal(e)
	}
	tr := c.draft()
	c.upload(tr, "secret.txt", "retained through key recovery")
	c.publish(tr)
	backup := operationsJob(t, c, "backup", "", provider)
	if e = c.a.Operations.Execute(ctx, backup, nil); e != nil {
		t.Fatal(e)
	}
	old := append([]byte(nil), c.a.Keys.KEK...)
	next := bytes.Repeat([]byte{42}, 32)
	if e = c.a.Operations.Rewrap(ctx, next, provider, "host_cli", "Rotate externally escrowed master key"); e != nil {
		t.Fatal(e)
	}
	if e = database.CheckKEK(ctx, c.a.DB, old); e == nil {
		t.Fatal("old KEK accepted after rewrap")
	}
	newer, e := New(c.a.DB, next)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(newer.Close)
	if e = audit.Verify(ctx, newer.DB, newer.Key); e != nil {
		t.Fatal("audit key changed", e)
	}
	if _, e = newer.Operations.VerifyBackup(ctx, backup, nil); e == nil {
		t.Fatal("historical backup opened with new KEK")
	}
	if _, e = newer.Operations.VerifyBackup(ctx, backup, old); e != nil {
		t.Fatal("historical backup recovery", e)
	}
	report, e := newer.Operations.Reconcile(ctx, "host_cli", "Verify rewrapped data and historical keys")
	if e != nil || !report.Passed {
		t.Fatal(e, report.Findings)
	}
	if e = newer.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Reviewed capabilities after master rotation", true); e != nil {
		t.Fatal(e)
	}
	// An empty database cannot know the old master key's usage high-water mark.
	freshSame, _ := integrationAppWithKEK(t, old)
	sameProvider, e := freshSame.Files.Storage.Provision(ctx, root, "backups")
	if e != nil {
		t.Fatal(e)
	}
	if e = freshSame.Operations.ImportBackup(ctx, "host_cli", backup, sameProvider, "Recover external backup catalog", old); e != nil {
		t.Fatal(e)
	}
	if _, e = freshSame.Operations.QueueAndExecute(ctx, "host_cli", "restore", backup, sameProvider, "Reject reused master on a fresh instance", old); e == nil {
		t.Fatal("fresh instance reused untracked KEK")
	}
	fresh, _ := integrationAppWithKEK(t, bytes.Repeat([]byte{55}, 32))
	importedProvider, e := fresh.Files.Storage.Provision(ctx, root, "backups")
	if e != nil {
		t.Fatal(e)
	}
	if e = fresh.Operations.ImportBackup(ctx, "host_cli", backup, importedProvider, "Recover external backup catalog", old); e != nil {
		t.Fatal(e)
	}
	if _, e = fresh.Operations.QueueAndExecute(ctx, "host_cli", "restore", backup, importedProvider, "Restore using a fresh target master key", old); e != nil {
		t.Fatal("fresh restore", e)
	}
	restored, e := New(fresh.DB, fresh.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(restored.Close)
	report, e = restored.Operations.Reconcile(ctx, "host_cli", "Validate recovered storage and crypto")
	if e != nil || !report.Passed {
		t.Fatal(e, report.Findings)
	}
	var exists bool
	if e = restored.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM transfers WHERE id=$1)", tr).Scan(&exists); e != nil || !exists {
		t.Fatal("fresh recovery lost data", e)
	}
	if e = restored.Operations.Reopen(ctx, "host_cli", report.ID, report.Hash, "Reviewed restored capabilities and keys", true); e != nil {
		t.Fatal(e)
	}
}

func TestIntegrationRecoveryDrainsWritersAndRollsBackImport(t *testing.T) {
	c, _ := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	provider, e := c.a.Files.Storage.Provision(ctx, t.TempDir(), "backups")
	if e != nil {
		t.Fatal(e)
	}
	tr := c.draft()
	if _, e = c.a.DB.Exec(ctx, "UPDATE transfers SET title='forbidden snapshot title' WHERE id=$1", tr); e != nil {
		t.Fatal(e)
	}
	backup := operationsJob(t, c, "backup", "", provider)
	if e = c.a.Operations.Execute(ctx, backup, nil); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE transfers SET title='current title' WHERE id=$1", tr); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "ALTER TABLE transfers ADD CONSTRAINT injected_restore_fault CHECK(title<>'forbidden snapshot title') NOT VALID"); e != nil {
		t.Fatal(e)
	}
	// This is the same lease used around HTTP streams and worker side effects.
	work, release, e := c.a.Gate.Acquire(ctx, false)
	if e != nil {
		t.Fatal(e)
	}
	defer release()
	restore := operationsJob(t, c, "restore", backup, provider)
	done := make(chan error, 1)
	go func() { done <- c.a.Operations.Execute(ctx, restore, nil) }()
	select {
	case <-work.Done():
	case <-ctx.Done():
		t.Fatal("writer was not cancelled")
	}
	select {
	case e := <-done:
		t.Fatal("restore bypassed a live writer", e)
	default:
	}
	release()
	if e = <-done; e == nil {
		t.Fatal("constraint fault did not abort import")
	}
	var title string
	if e = c.a.DB.QueryRow(ctx, "SELECT title FROM transfers WHERE id=$1", tr).Scan(&title); e != nil || title != "current title" {
		t.Fatal("partial import leaked", e, title)
	}
	var budget int64
	if e = c.a.DB.QueryRow(ctx, "SELECT seal_count FROM crypto_usage WHERE name='kek'").Scan(&budget); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "ALTER TABLE transfers DROP CONSTRAINT injected_restore_fault"); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Operations.QueueAndExecute(ctx, "host_cli", "restore", backup, provider, "Retry after resolving injected constraint fault", nil); e != nil {
		t.Fatal(e)
	}
	var after int64
	if e = c.a.DB.QueryRow(ctx, "SELECT seal_count FROM crypto_usage WHERE name='kek'").Scan(&after); e != nil || after < budget+4 {
		t.Fatal("KEK reservation rolled back", e, budget, after)
	}
	if _, _, e = c.a.Gate.Acquire(ctx, true); !errors.Is(e, maintenance.ErrRestart) {
		t.Fatal("stale runtime admitted", e)
	}
	if e = c.a.Operations.Run(ctx); !errors.Is(e, maintenance.ErrRestart) {
		t.Fatal("stale worker admitted", e)
	}
	var unsafe int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM key_versions WHERE class<>'signing' AND NOT active AND seal_count<>1048576").Scan(&unsafe); e != nil || unsafe != 0 {
		t.Fatal("restored old root can still seal", e)
	}
}

func TestIntegrationOperationsSchedulesPrivacyAndCryptoContainment(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	provider, e := c.a.Files.Storage.Provision(ctx, t.TempDir(), "backups")
	if e != nil {
		t.Fatal(e)
	}
	if e = c.a.Operations.SetSchedule(ctx, "host_cli", "backup", 60, provider); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE ops_schedules SET next_at=clock_timestamp()-interval '1 minute'"); e != nil {
		t.Fatal(e)
	}
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { errs <- c.a.Operations.Schedule(ctx) }()
	}
	for i := 0; i < 2; i++ {
		if e = <-errs; e != nil {
			t.Fatal(e)
		}
	}
	var count int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM ops_jobs WHERE state='queued'").Scan(&count); e != nil || count != 3 {
		t.Fatal("schedule duplication", e, count)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp()-interval '1 hour'"); e != nil {
		t.Fatal(e)
	}
	if w := c.req("GET", "/api/v1/admin/operations", "", nil); w.Code != 403 {
		t.Fatal("stale reauth accepted", w.Code)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE users SET role='user'"); e != nil {
		t.Fatal(e)
	}
	if w := c.req("GET", "/api/v1/admin/operations", "", nil); w.Code != 403 {
		t.Fatal("USER operations leak", w.Code)
	}
	// Corrupt the configuration key itself: the encrypted incident cannot be
	// created, but access must already be blocked durably.
	kid, _, e := c.a.Keys.Active(ctx, "configuration")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE key_versions SET wrapped='broken'::bytea WHERE id=$1", kid); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Operations.CheckCrypto(ctx); e == nil {
		t.Fatal("bad key accepted")
	}
	state, e := c.a.Operations.State(ctx)
	if e != nil || state.Mode != "blocked" {
		t.Fatal("crypto fault failed open", e, state)
	}
	if w := c.req("GET", "/api/v1/status", "", nil); w.Code != 503 || strings.Contains(w.Body.String(), "crypto") {
		t.Fatal("public operational detail leak", w.Code, w.Body.String())
	}
}

func TestIntegrationOperationsLedgerCheckpointPreservesEvidence(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	if e := c.a.Operations.SetMaintenance(ctx, "host_cli", "Investigate operational ledger incident"); e != nil {
		t.Fatal(e)
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE ops_events SET payload='tampered'::bytea WHERE id=(SELECT min(id) FROM ops_events)"); e != nil {
		t.Fatal(e)
	}
	if e := c.a.Operations.CheckAudit(ctx); e == nil {
		t.Fatal("ops tamper undetected")
	}
	evidence, e := c.a.Operations.AuditEvidence(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Operations.Checkpoint(ctx, "host_cli", "recovery", "Preserve both damaged journals and resume", evidence); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Operations.CheckAudit(ctx); e != nil {
		t.Fatal("new ops segment", e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE ops_events SET payload='changed again'::bytea WHERE id=(SELECT min(id) FROM ops_events)"); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Operations.CheckAudit(ctx); e == nil {
		t.Fatal("history changed after checkpoint undetected")
	}
}

func TestIntegrationAdministrativeExportAccessAndTTL(t *testing.T) {
	c, _ := testClient(t)
	artifactProvider(t, c)
	ctx := context.Background()
	owner, e := c.a.Auth.CreateUser(ctx, "host_cli", "export-owner@example.test", "strong-password-0123", "user")
	if e != nil {
		t.Fatal(e)
	}
	artifact, e := c.a.artifactStore().Create(ctx, owner, "private.json", "application/json", func(w io.Writer) error { _, e := io.WriteString(w, `{"private":"export content"}`); return e })
	if e != nil {
		t.Fatal(e)
	}
	job := domain.NewID()
	if _, e = c.a.DB.Exec(ctx, "INSERT INTO export_jobs(id,owner_id,kind,format,compression,state,artifact_id,auth_epoch) SELECT $1,id,'user','json','none','completed',$3,auth_epoch FROM users WHERE id=$2", job, owner, artifact); e != nil {
		t.Fatal(e)
	}
	if w := c.req("POST", "/api/v1/artifacts/"+artifact+"/credentials", "", nil); w.Code != 404 {
		t.Fatal("owner boundary bypass", w.Code)
	}
	if w := c.req("POST", "/api/v1/admin/exports/"+job+"/credentials", `{"reason":"ключ"}`, nil); w.Code != 400 {
		t.Fatal("short Unicode reason accepted", w.Code)
	}
	w := c.req("POST", "/api/v1/admin/exports/"+job+"/credentials", fmt.Sprintf(`{"reason":%q}`, strings.Repeat("я", 2000)), nil)
	if w.Code != 201 {
		t.Fatal("ADMIN credential", w.Code, w.Body.String())
	}
	var credential struct{ ArtifactID, Token string }
	decode(t, w, &credential)
	w = c.req("POST", "/api/v1/artifacts/"+artifact+"/claim", fmt.Sprintf(`{"token":%q}`, credential.Token), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	headers := map[string]string{"Cookie": c.cookie.String() + "; " + cookie.String()}
	if w = c.req("HEAD", "/api/v1/artifacts/"+artifact+"/content", "", headers); w.Code != 200 {
		t.Fatal("ADMIN read", w.Code, w.Body.String())
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp()-interval '10 minutes'"); e != nil {
		t.Fatal(e)
	}
	if w = c.req("GET", "/api/v1/artifacts/"+artifact+"/content", "", headers); w.Code != 401 {
		t.Fatal("stale ADMIN stream", w.Code)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp(); UPDATE users SET role='user' WHERE email='sender@example.test'"); e != nil {
		t.Fatal(e)
	}
	if w = c.req("GET", "/api/v1/artifacts/"+artifact+"/content", "", headers); w.Code != 401 {
		t.Fatal("demoted ADMIN stream", w.Code)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE users SET role='admin' WHERE email='sender@example.test'"); e != nil {
		t.Fatal(e)
	}
	if w = c.req("PATCH", "/api/v1/admin/exports/"+job+"/ttl", fmt.Sprintf(`{"reason":%q,"ttlSeconds":60}`, strings.Repeat("я", 2000)), nil); w.Code != 204 {
		t.Fatal("TTL update", w.Code, w.Body.String())
	}
	if w = c.req("GET", "/api/v1/artifacts/"+artifact+"/content", "", headers); w.Code != 200 || !strings.Contains(w.Body.String(), "export content") {
		t.Fatal("ADMIN content", w.Code)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE artifacts SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", artifact); e != nil {
		t.Fatal(e)
	}
	if w = c.req("PATCH", "/api/v1/admin/exports/"+job+"/ttl", `{"reason":"Cannot resurrect expired artifacts","ttlSeconds":60}`, nil); w.Code != 409 {
		t.Fatal("expired artifact resurrected", w.Code)
	}
	var actions, notices int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM administrative_actions WHERE subject_id=$1", job).Scan(&actions); e != nil || actions != 2 {
		t.Fatal("administrative audit", e, actions)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE user_id=$1", owner).Scan(&notices); e != nil || notices != 0 {
		t.Fatal("owner notified", e)
	}
}

func TestIntegrationUpgradePreflightIsReadOnly(t *testing.T) {
	a, p := integrationApp(t)
	ctx := context.Background()
	_ = a
	plan, e := database.CheckUpgrade(ctx, p)
	if e != nil || !plan.Compatible || len(plan.Pending) != 0 {
		t.Fatal(plan, e)
	}
	if _, e = p.Exec(ctx, "DELETE FROM schema_migrations WHERE name='0016_administrative_exports.sql'"); e != nil {
		t.Fatal(e)
	}
	plan, e = database.CheckUpgrade(ctx, p)
	if e != nil || len(plan.Pending) != 1 {
		t.Fatal(plan, e)
	}
	var exists bool
	if e = p.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name='0016_administrative_exports.sql')").Scan(&exists); e != nil || exists {
		t.Fatal("preflight applied migration", e)
	}
	if _, e = p.Exec(ctx, "INSERT INTO schema_migrations(name,sha256) VALUES('9999_newer.sql','unknown')"); e != nil {
		t.Fatal(e)
	}
	if _, e = database.CheckUpgrade(ctx, p); e == nil {
		t.Fatal("downgrade accepted")
	}
}

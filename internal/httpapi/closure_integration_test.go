package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/worker"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

func TestIntegrationRotationAcrossClassesAndArtifactLease(t *testing.T) {
	c, _ := testClient(t)
	artifactProvider(t, c)
	ctx := context.Background()
	id := c.draft()
	c.upload(id, "a.txt", "rotation evidence")
	c.publish(id)
	links := c.req("GET", "/api/v1/transfers/"+id, "", nil).Body.String()
	w := c.req("PUT", "/api/v1/admin/smtp", `{"host":"mail.example.test","port":587,"tls":"starttls","username":"test","password":"fixture-secret","from":"sender@example.test"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var smtpID string
	if e := c.a.DB.QueryRow(ctx, "SELECT id::text FROM communication_configs WHERE kind='smtp'").Scan(&smtpID); e != nil {
		t.Fatal(e)
	}
	w = c.req("POST", "/api/v1/transfers/"+id+"/reports", `{"kind":"user","locale":"ru","timezone":"UTC"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var rep struct{ ID string }
	decode(t, w, &rep)
	if e := c.a.reportService().RenderPending(ctx); e != nil {
		t.Fatal(e)
	}
	before, hash, e := c.a.reportService().Snapshot(ctx, rep.ID)
	if e != nil {
		t.Fatal(e)
	}
	var artifact string
	if e = c.a.DB.QueryRow(ctx, "SELECT artifact_id::text FROM report_artifacts WHERE report_id=$1 AND format='json'", rep.ID).Scan(&artifact); e != nil {
		t.Fatal(e)
	}
	_, _ = c.claimArtifact(artifact)
	worker, e := worker.New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	for _, class := range []string{"configuration", "artifacts"} {
		w = c.req("POST", "/api/v1/admin/keys/rotate", `{"class":"`+class+`"}`, nil)
		if w.Code != 202 {
			t.Fatal(w.Code, w.Body.String())
		}
		var job struct {
			ID  string
			Key string `json:"keyId"`
		}
		decode(t, w, &job)
		for i := 0; i < 15; i++ {
			if e = worker.Rotate(ctx); e != nil {
				t.Fatal(class, e)
			}
		}
		if class == "artifacts" {
			var state string
			if e = c.a.DB.QueryRow(ctx, "SELECT state FROM rotation_jobs WHERE id=$1", job.ID).Scan(&state); e != nil || state != "running" {
				t.Fatal("live download not retained", state, e)
			}
			if _, e = c.a.DB.Exec(ctx, "UPDATE artifact_credentials SET expires_at=clock_timestamp()-interval '1 second'"); e != nil {
				t.Fatal(e)
			}
			for i := 0; i < 10; i++ {
				if e = worker.Rotate(ctx); e != nil {
					t.Fatal(e)
				}
			}
		}
		var state string
		if e = c.a.DB.QueryRow(ctx, "SELECT state FROM rotation_jobs WHERE id=$1", job.ID).Scan(&state); e != nil || state != "completed" {
			t.Fatal(class, state, e)
		}
	}
	if got := c.req("GET", "/api/v1/transfers/"+id, "", nil).Body.String(); got != links {
		t.Fatal("link or payload changed during re-encryption")
	}
	after, h, e := c.a.reportService().Snapshot(ctx, rep.ID)
	if e != nil || h != hash || !bytes.Equal(before, after) {
		t.Fatal("snapshot changed", e)
	}
	a, e := c.a.artifactStore().Get(ctx, artifact)
	if e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if e = c.a.artifactStore().Read(ctx, a, 0, a.Size, &out); e != nil || !bytes.Equal(before, out.Bytes()) {
		t.Fatal("re-encrypted artifact changed", e)
	}
	var smtp communications.SMTPConfig
	if e = (communications.Service{DB: c.a.DB, Keys: c.a.Keys}).Config(ctx, smtpID, &smtp); e != nil || smtp.Password != "fixture-secret" {
		t.Fatal("configuration changed", e)
	}
	old, _, e := c.a.Keys.Active(ctx, "configuration")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Keys.Rotate(ctx, "configuration"); e != nil {
		t.Fatal(e)
	}
	tx, e := c.a.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if !errors.Is(keyring.RequireActive(ctx, tx, old), keyring.ErrChanged) {
		t.Fatal("stale key reference accepted")
	}
}
func TestIntegrationForensicZIPDoesNotConsumeAndTusProtocol(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	metadata := "filename " + base64.StdEncoding.EncodeToString([]byte("empty.txt"))
	w := c.req("POST", "/api/v1/transfers/"+id+"/uploads", "", map[string]string{"Tus-Resumable": "1.0.0", "Upload-Length": "0", "Upload-Metadata": metadata})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	url := w.Header().Get("Location")
	w = c.req("HEAD", url, "", map[string]string{"Tus-Resumable": ""})
	if w.Code != 412 {
		t.Fatal("tus version not enforced", w.Code)
	}
	w = c.req("POST", url, "", map[string]string{"X-HTTP-Method-Override": "PATCH", "Upload-Offset": "0", "Content-Type": "application/offset+octet-stream"})
	if w.Code != 204 || w.Header().Get("Upload-Offset") != "0" {
		t.Fatal("empty PATCH", w.Code, w.Body.String())
	}
	w = c.req("POST", url+"/finalize", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	c.upload(id, "dir/a.txt", "forensic bytes")
	c.publish(id)
	w = c.req("POST", "/api/v1/admin/transfers/"+id+"/forensic-zip", `{"reason":"local investigation"}`, nil)
	if w.Code != 409 {
		t.Fatal("unheld archive", w.Code)
	}
	w = c.req("POST", "/api/v1/admin/transfers/"+id+"/hold", `{"frozen":true,"reason":"local investigation"}`, nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("POST", "/api/v1/admin/transfers/"+id+"/forensic-zip", `{"reason":"local investigation"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	z, e := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if e != nil || len(z.File) != 5 {
		t.Fatal("forensic ZIP", e)
	}
	var count int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state='ready'", id).Scan(&count); e != nil || count != 2 {
		t.Fatal("forensic consumed files", e, count)
	}
	w = c.req("POST", "/api/v1/transfers/"+id+"/revoke", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("POST", "/api/v1/admin/transfers/"+id+"/hold", `{"frozen":false,"reason":"investigation completed"}`, nil)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	for i := 0; i < 4; i++ {
		if e = c.a.Files.Cleanup(ctx); e != nil {
			t.Fatal(e)
		}
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state='deleted'", id).Scan(&count); e != nil || count != 2 {
		t.Fatal("zero-file retention stuck", count, e)
	}
}
func TestIntegrationWebhookReceiptAtomicReplay(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	s := communications.Service{DB: c.a.DB}
	id := domain.NewID()
	secret := strings.Repeat("s", 32)
	b, _ := json.Marshal(map[string]any{"eventId": id, "schemaVersion": 1, "type": "transfer.published"})
	at := time.Now().Unix()
	sig := communications.Signature(secret, at, b)
	if ok, e := s.AcceptWebhook(ctx, secret, id, at, b, sig, func(pgx.Tx) error { return errors.New("consumer crashed") }); e == nil || ok {
		t.Fatal("consumer failure accepted")
	}
	applied := 0
	fn := func(pgx.Tx) error { applied++; return nil }
	if ok, e := s.AcceptWebhook(ctx, secret, id, at, b, sig, fn); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ok, e := s.AcceptWebhook(ctx, secret, id, at, b, sig, fn); e != nil || ok || applied != 1 {
		t.Fatal("duplicate applied", ok, e, applied)
	}
	if _, e := s.AcceptWebhook(ctx, secret, domain.NewID(), at, b, sig, fn); e == nil {
		t.Fatal("unsigned event ID substituted")
	}
	if _, e := s.AcceptWebhook(ctx, secret, id, at-301, b, communications.Signature(secret, at-301, b), fn); e == nil {
		t.Fatal("stale replay accepted")
	}
}

func TestIntegrationAtomicTreeAndAdminArtifactGuards(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	a := c.upload(id, "folder/a.txt", "a")
	b := c.upload(id, "folder/b.txt", "b")
	w := c.req("POST", "/api/v1/transfers/"+id+"/folders", `{"path":"folder"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var folder struct{ ID string }
	decode(t, w, &folder)
	w = c.req("PATCH", "/api/v1/transfers/"+id+"/files", `{"moves":[{"id":"`+a+`","path":"same"},{"id":"`+b+`","path":"same"}]}`, nil)
	if w.Code != 409 {
		t.Fatal("duplicate move", w.Code, w.Body.String())
	}
	var path string
	if e := c.a.DB.QueryRow(ctx, "SELECT path FROM files WHERE id=$1", a).Scan(&path); e != nil || path != "folder/a.txt" {
		t.Fatal("partial batch mutation", path, e)
	}
	w = c.req("PATCH", "/api/v1/transfers/"+id+"/folders/"+folder.ID, `{"path":"renamed"}`, nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e := c.a.DB.QueryRow(ctx, "SELECT path FROM files WHERE id=$1", a).Scan(&path); e != nil || path != "renamed/a.txt" {
		t.Fatal(path, e)
	}
	w = c.req("DELETE", "/api/v1/transfers/"+id+"/folders/"+folder.ID, "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	c.upload(id, "folder/a.txt", "new bytes")
	c.publish(id)
	artifactProvider(t, c)
	w = c.req("POST", "/api/v1/transfers/"+id+"/reports", `{"kind":"admin","locale":"en","timezone":"UTC"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var rep struct{ ID string }
	decode(t, w, &rep)
	if e := c.a.reportService().RenderPending(ctx); e != nil {
		t.Fatal(e)
	}
	var artifact string
	if e := c.a.DB.QueryRow(ctx, "SELECT artifact_id::text FROM report_artifacts WHERE report_id=$1 AND format='json'", rep.ID).Scan(&artifact); e != nil {
		t.Fatal(e)
	}
	url, cookie := c.claimArtifact(artifact)
	if _, e := c.a.DB.Exec(ctx, "UPDATE users SET role='user' WHERE email='sender@example.test'"); e != nil {
		t.Fatal(e)
	}
	w = c.req("GET", url, "", map[string]string{"Cookie": cookie.String()})
	if w.Code != 401 {
		t.Fatal("demoted admin retained artifact", w.Code, w.Body.String())
	}
	w = c.req("POST", "/api/v1/artifacts/"+artifact+"/credentials", "", nil)
	if w.Code != 404 {
		t.Fatal("demoted admin minted credential", w.Code)
	}
}

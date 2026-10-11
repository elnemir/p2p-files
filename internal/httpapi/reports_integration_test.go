package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"filex.local/filex/internal/exports"
	"filex.local/filex/internal/storage"
	"filex.local/filex/internal/worker"
	"io"
	"net/http"
	"strings"
	"testing"
)

func artifactProvider(t *testing.T, c apiClient) {
	t.Helper()
	if _, e := (storage.Registry{DB: c.a.DB}).Provision(context.Background(), t.TempDir(), "artifacts"); e != nil {
		t.Fatal(e)
	}
}
func (c apiClient) claimArtifact(id string) (string, *http.Cookie) {
	c.t.Helper()
	w := c.req("POST", "/api/v1/artifacts/"+id+"/credentials", "", nil)
	if w.Code != 201 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ Token string }
	decode(c.t, w, &v)
	w = c.req("POST", "/api/v1/artifacts/"+id+"/claim", `{"token":"`+v.Token+`"}`, nil)
	if w.Code != 200 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	var target struct{ URL string }
	decode(c.t, w, &target)
	again := c.req("POST", "/api/v1/artifacts/"+id+"/claim", `{"token":"`+v.Token+`"}`, nil)
	if again.Code != 401 {
		c.t.Fatal("one-time credential reused")
	}
	return target.URL, cookie
}
func TestIntegrationReportPrivacyArtifactsAndTrash(t *testing.T) {
	c, _ := testClient(t)
	artifactProvider(t, c)
	ctx := context.Background()
	id := c.draft()
	c.upload(id, "=SUM(1).txt", "содержание")
	c.publish(id)
	w := c.req("POST", "/api/v1/transfers/"+id+"/reports", `{"kind":"user","locale":"ru","timezone":"Europe/Moscow"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var report struct {
		ID   string
		Hash string `json:"canonicalHash"`
	}
	decode(t, w, &report)
	if e := c.a.reportService().RenderPending(ctx); e != nil {
		t.Fatal(e)
	}
	raw, hash, e := c.a.reportService().Snapshot(ctx, report.ID)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	if hash != hex.EncodeToString(sum[:]) || hash != report.Hash {
		t.Fatal("canonical hash mismatch")
	}
	var content map[string]any
	if e = json.Unmarshal(raw, &content); e != nil {
		t.Fatal(e)
	}
	contentHash, ok := content["documentContentHash"].(string)
	if !ok {
		t.Fatal("missing document hash")
	}
	delete(content, "documentContentHash")
	contentBytes, e := json.Marshal(content)
	if e != nil {
		t.Fatal(e)
	}
	contentSum := sha256.Sum256(contentBytes)
	if contentHash != hex.EncodeToString(contentSum[:]) {
		t.Fatal("document hash mismatch")
	}
	for _, secret := range []string{"recipientEmail", "session", "administrativeEvents", "sender@example"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("USER report leaked", secret)
		}
	}
	w = c.req("GET", "/api/v1/reports/"+report.ID, "", nil)
	var detail struct{ Artifacts map[string]string }
	decode(t, w, &detail)
	if len(detail.Artifacts) != 3 {
		t.Fatal(w.Body.String())
	}
	for format, artifact := range detail.Artifacts {
		ar, e := c.a.artifactStore().Get(ctx, artifact)
		if e != nil {
			t.Fatal(e)
		}
		var b bytes.Buffer
		if e = c.a.artifactStore().Read(ctx, ar, 0, ar.Size, &b); e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b.Bytes())
		if ar.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatal("artifact hash mismatch", format)
		}
		switch format {
		case "json":
			if !bytes.Equal(raw, b.Bytes()) {
				t.Fatal("JSON differs")
			}
		case "pdf":
			if !bytes.HasPrefix(b.Bytes(), []byte("%PDF-")) {
				t.Fatal("PDF missing")
			}
		case "csv":
			if !strings.Contains(b.String(), "'=SUM(1).txt") {
				t.Fatal("CSV formula injection", b.String())
			}
		}
	}
	url, cookie := c.claimArtifact(detail.Artifacts["json"])
	headers := map[string]string{"Cookie": cookie.Name + "=" + cookie.Value, "Range": "bytes=0-15"}
	w = c.req("GET", url, "", headers)
	if w.Code != 206 || w.Body.String() != string(raw[:16]) {
		t.Fatal(w.Code, w.Body.String())
	}
	headers["Range"] = "bytes=16-"
	w = c.req("GET", url, "", headers)
	if w.Code != 206 || w.Body.String() != string(raw[16:]) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("GET", url, "", headers)
	if w.Code != 401 {
		t.Fatal("consumed artifact", w.Code)
	}
	w = c.req("GET", "/api/v1/verify/reports/"+report.ID, "", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), "SUM(") {
		t.Fatal("verify privacy")
	}
	w = c.req("POST", "/api/v1/verify-report", `{"reportId":"`+report.ID+`","revision":1,"canonicalHash":"`+hash+`"}`, nil)
	if w.Code != 200 {
		t.Fatal("verification tuple rejected", w.Code)
	}
	w = c.req("POST", "/api/v1/verify-report", `{"reportId":"`+report.ID+`","revision":2,"canonicalHash":"`+hash+`"}`, nil)
	if w.Code != 404 {
		t.Fatal("wrong revision verified")
	}
	for _, state := range []string{"trashed", "active", "trashed", "purged"} {
		w = c.req("POST", "/api/v1/reports/"+report.ID+"/state", `{"state":"`+state+`"}`, nil)
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w = c.req("POST", "/api/v1/reports/"+report.ID+"/state", `{"state":"active"}`, nil)
	if w.Code != 409 {
		t.Fatal("purged report resurrected")
	}
	w = c.req("GET", "/api/v1/verify/reports/"+report.ID, "", nil)
	if w.Code != 404 || strings.Contains(w.Body.String(), hash) {
		t.Fatal("purged report disclosed publicly")
	}
	w = c.req("POST", "/api/v1/transfers/"+id+"/reports", `{"kind":"admin","locale":"en","timezone":"UTC"}`, nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	decode(t, w, &report)
	public := apiClient{t: t, a: c.a, h: c.h}
	w = public.req("GET", "/api/v1/verify/reports/"+report.ID, "", nil)
	if w.Code != 404 {
		t.Fatal("ADMIN report public")
	}
}
func TestIntegrationExportSnapshotRetryAndBulk(t *testing.T) {
	c, _ := testClient(t)
	artifactProvider(t, c)
	ctx := context.Background()
	id := c.draft()
	if _, e := c.a.DB.Exec(ctx, "UPDATE transfers SET title='snapshot before' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	w := c.req("POST", "/api/v1/exports", `{"kind":"user","format":"json","compression":"gzip"}`, nil)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	var job struct{ ID string }
	decode(t, w, &job)
	s := exports.Service{DB: c.a.DB, Artifacts: c.a.artifactStore()}
	if e := s.Run(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE transfers SET title='changed after snapshot' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE export_jobs SET state='rendering',lease_until=clock_timestamp()-interval '1 second' WHERE id=$1", job.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.Run(ctx); e != nil {
		t.Fatal(e)
	}
	var artifact string
	if e := c.a.DB.QueryRow(ctx, "SELECT artifact_id::text FROM export_jobs WHERE id=$1 AND state='completed'", job.ID).Scan(&artifact); e != nil {
		t.Fatal(e)
	}
	ar, e := c.a.artifactStore().Get(ctx, artifact)
	if e != nil {
		t.Fatal(e)
	}
	var encryptedDownload bytes.Buffer
	if e = c.a.artifactStore().Read(ctx, ar, 0, ar.Size, &encryptedDownload); e != nil {
		t.Fatal(e)
	}
	gz, e := gzip.NewReader(&encryptedDownload)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(gz)
	gz.Close()
	if e != nil || !strings.Contains(string(raw), "snapshot before") || strings.Contains(string(raw), "changed after") {
		t.Fatal("snapshot drift", e, string(raw))
	}
	var n int
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM export_rows WHERE job_id=$1", job.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal("snapshot cleanup")
	}
	req, _ := json.Marshal(map[string]any{"ids": []string{id}, "action": "revoke", "dryRun": true})
	w = c.req("POST", "/api/v1/bulk", string(req), nil)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	decode(t, w, &job)
	worker, e := worker.New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	if e = worker.Bulk(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM transfers WHERE id=$1", id).Scan(&state); e != nil || state != "draft" {
		t.Fatal("dry-run mutated", e, state)
	}
	w = c.req("POST", "/api/v1/bulk/"+job.ID+"/run", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e = worker.Bulk(ctx); e != nil {
		t.Fatal(e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM transfers WHERE id=$1", id).Scan(&state); e != nil || state != "revoked" {
		t.Fatal("bulk not applied", e, state)
	}
}

package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"filex.local/filex/internal/exports"
	"io"
	"strings"
	"testing"
)

func TestIntegrationQRInvalidationAndExportAuthorization(t *testing.T) {
	c, _ := testClient(t)
	artifactProvider(t, c)
	ctx := context.Background()
	id := c.draft()
	c.upload(id, "a.txt", "secret payload")
	c.publish(id)
	w := c.req("POST", "/api/v1/transfers/"+id+"/qr-bundle", "", nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ ArtifactID string }
	decode(t, w, &v)
	ar, e := c.a.artifactStore().Get(ctx, v.ArtifactID)
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	if e = c.a.artifactStore().Read(ctx, ar, 0, ar.Size, &b); e != nil {
		t.Fatal(e)
	}
	z, e := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if e != nil {
		t.Fatal(e)
	}
	mapping, e := z.Open("recipients.csv")
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(mapping)
	mapping.Close()
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(raw), "linkId,email,url,qrFile\r\n") || !strings.Contains(string(raw), "/receive#token=") {
		t.Fatal("mapping missing")
	}
	var object string
	if e = c.a.DB.QueryRow(ctx, "SELECT object_key FROM artifact_chunks WHERE artifact_id=$1", v.ArtifactID).Scan(&object); e != nil {
		t.Fatal(e)
	}
	provider, e := c.a.Files.Storage.Get(ctx, ar.Provider)
	if e != nil {
		t.Fatal(e)
	}
	cipher, e := provider.Get(ctx, object)
	if e != nil || bytes.Contains(cipher, []byte("recipients.csv")) {
		t.Fatal("QR plaintext stored", e)
	}
	target, cookie := c.claimArtifact(v.ArtifactID)
	w = c.req("GET", target, "", map[string]string{"Cookie": cookie.Name + "=" + cookie.Value, "Range": "bytes=0-15"})
	if w.Code != 206 {
		t.Fatal(w.Code)
	}
	var link string
	if e = c.a.DB.QueryRow(ctx, "SELECT id::text FROM recipient_links WHERE transfer_id=$1 LIMIT 1", id).Scan(&link); e != nil {
		t.Fatal(e)
	}
	w = c.req("POST", "/api/v1/transfers/"+id+"/links/"+link+"/rotate", `{"password":""}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("GET", target, "", map[string]string{"Cookie": cookie.Name + "=" + cookie.Value, "Range": "bytes=16-"})
	if w.Code != 401 {
		t.Fatal("rotated QR still accessible", w.Code)
	}
	w = c.req("POST", "/api/v1/artifacts/"+v.ArtifactID+"/credentials", "", nil)
	if w.Code != 404 {
		t.Fatal("new credential for stale QR", w.Code)
	}
	w = c.req("POST", "/api/v1/exports", `{"kind":"admin","format":"json","compression":"none"}`, nil)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	var job struct{ ID string }
	decode(t, w, &job)
	if _, e = c.a.DB.Exec(ctx, "UPDATE users SET role='user' WHERE email='sender@example.test'"); e != nil {
		t.Fatal(e)
	}
	service := exports.Service{DB: c.a.DB, Artifacts: c.a.artifactStore()}
	if e = service.Run(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	var rows int
	if e = c.a.DB.QueryRow(ctx, "SELECT state,(SELECT count(*) FROM export_rows WHERE job_id=export_jobs.id) FROM export_jobs WHERE id=$1", job.ID).Scan(&state, &rows); e != nil || state != "cancelled" || rows != 0 {
		t.Fatal("revoked admin executed snapshot", state, rows, e)
	}
}
func TestIntegrationPayloadHeaderAndKeyBudget(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	file := c.upload(id, "a.txt", "authenticated bytes")
	var owner string
	if e := c.a.DB.QueryRow(ctx, "SELECT owner_id::text FROM transfers WHERE id=$1", id).Scan(&owner); e != nil {
		t.Fatal(e)
	}
	f, e := c.a.Files.Get(ctx, owner, file)
	if e != nil {
		t.Fatal(e)
	}
	wrong := f
	wrong.Size++
	if e = c.a.Files.Read(ctx, wrong, 0, f.Size, io.Discard); e == nil {
		t.Fatal("file length substitution accepted")
	}
	kid, key, e := c.a.Keys.Active(ctx, "configuration")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE key_versions SET seal_count=1048575 WHERE id=$1", kid); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Keys.Seal(ctx, kid, key, "filex/test/v1/budget", []byte("last allowed")); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Keys.Seal(ctx, kid, key, "filex/test/v1/budget", []byte("forbidden")); e == nil {
		t.Fatal("key budget bypass")
	}
	next, e := c.a.Keys.Rotate(ctx, "configuration")
	if e != nil {
		t.Fatal(e)
	}
	nextKey, e := c.a.Keys.Get(ctx, next)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Keys.Seal(ctx, next, nextKey, "filex/test/v1/budget", []byte("new key")); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = c.a.DB.QueryRow(ctx, "SELECT seal_count FROM key_versions WHERE id=$1", kid).Scan(&count); e != nil || count != 1048576 {
		t.Fatal("usage count rolled over", count, e)
	}
}

package httpapi

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/storage"
	"os"
	"testing"
)

func TestIntegrationStorageMigrationAndPayloadRotation(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	file := c.upload(id, "test.txt", "immutable payload")
	token := c.publish(id)
	ctx := context.Background()
	var source string
	if e := c.a.DB.QueryRow(ctx, "SELECT provider_id::text FROM files WHERE id=$1", file).Scan(&source); e != nil {
		t.Fatal(e)
	}
	targetRoot := t.TempDir()
	target, e := (storage.Registry{DB: c.a.DB}).ProvisionLocal(ctx, targetRoot)
	if e != nil {
		t.Fatal(e)
	}
	request, _ := json.Marshal(map[string]any{"sourceId": source, "targetId": target, "dryRun": true})
	w := c.req("POST", "/api/v1/admin/storage/migrations", string(request), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	request, _ = json.Marshal(map[string]any{"sourceId": source, "targetId": target, "dryRun": false})
	w = c.req("POST", "/api/v1/admin/storage/migrations", string(request), nil)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var job struct{ ID string }
	decode(t, w, &job)
	if e = c.a.Files.Maintain(ctx); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(targetRoot)
	if len(entries) != 0 {
		t.Fatal("paused migration ran")
	}
	w = c.req("PATCH", "/api/v1/admin/storage/migrations/"+job.ID, `{"state":"running"}`, nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e = c.a.Files.Maintain(ctx); e != nil {
		t.Fatal(e)
	}
	var actual string
	if e = c.a.DB.QueryRow(ctx, "SELECT provider_id::text FROM files WHERE id=$1", file).Scan(&actual); e != nil || actual != target {
		t.Fatal(actual, e)
	}
	if e = c.a.Files.Maintain(ctx); e != nil {
		t.Fatal(e)
	}
	w = c.req("POST", "/api/v1/admin/keys/rotate", `{"class":"payload"}`, nil)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	var key struct {
		KeyID string `json:"keyId"`
	}
	decode(t, w, &key)
	if e = c.a.Files.Maintain(ctx); e != nil {
		t.Fatal(e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT key_id::text FROM files WHERE id=$1", file).Scan(&actual); e != nil || actual != key.KeyID {
		t.Fatal(actual, key.KeyID, e)
	}
	r := c.recipient(token)
	url := r.lease(`{"fileRef":"` + r.refs()[0]["ref"].(string) + `"}`)
	w = r.req("GET", url, "", nil)
	if w.Code != 200 || w.Body.String() != "immutable payload" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestIntegrationFileLeaseFencesStaleUpload(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	file := c.upload(id, "a", "abc")
	ctx := context.Background()
	var owner, provider, key string
	if e := c.a.DB.QueryRow(ctx, "SELECT t.owner_id::text,f.provider_id::text,f.key_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id=$1", file).Scan(&owner, &provider, &key); e != nil {
		t.Fatal(e)
	}
	job := domain.NewID()
	if _, e := c.a.DB.Exec(ctx, "INSERT INTO rotation_jobs(id,class,target_key_id,state) VALUES($1,'payload',$2,'cancelled')", job, key); e != nil {
		t.Fatal(e)
	}
	if e := c.a.Files.Relocate(ctx, owner, file, provider, key, job, true); e == nil {
		t.Fatal("cancelled job swapped payload")
	}
}

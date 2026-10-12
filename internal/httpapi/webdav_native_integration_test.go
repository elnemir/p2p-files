package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/transfer"
	"filex.local/filex/internal/worker"
)

const davLockBody = `<lockinfo xmlns="DAV:"><lockscope><exclusive/></lockscope><locktype><write/></locktype><owner><href>local test client</href></owner></lockinfo>`

func TestIntegrationWebDAVNativeLocksAndConditions(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	other := davCredential(t, c, id)
	davStatus(t, davRequest(c, cred, "PUT", "notes.txt", "original", nil), 201)
	lock := davRequest(c, cred, "LOCK", "notes.txt", davLockBody, map[string]string{"Depth": "0", "Timeout": "Second-120"})
	davStatus(t, lock, 200)
	token := lock.Header().Get("Lock-Token")
	if token == "" {
		t.Fatal("missing lock token")
	}
	headers := map[string]string{"If": "(" + token + ")"}
	davStatus(t, davRequest(c, other, "PUT", "notes.txt", "intruder", headers), 423)
	davStatus(t, davRequest(c, cred, "PUT", "notes.txt", "unlocked", nil), 423)
	davStatus(t, davRequest(c, other, "LOCK", "notes.txt", davLockBody, nil), 423)
	davStatus(t, davRequest(c, cred, "PUT", "notes.txt", "changed", headers), 204)
	davStatus(t, davRequest(c, cred, "LOCK", "notes.txt", "", headers), 200)
	read := davRequest(c, cred, "GET", "notes.txt", "", map[string]string{"Range": "bytes=1-3"})
	davStatus(t, read, 206)
	if read.Body.String() != "han" {
		t.Fatal("range", read.Body.String())
	}
	davStatus(t, davRequest(c, cred, "GET", "notes.txt", "", map[string]string{"If-None-Match": read.Header().Get("ETag")}), 304)
	davStatus(t, davRequest(c, cred, "PUT", "notes.txt", "stale", map[string]string{"If": "(" + token + ")", "If-Match": "\"old\""}), 412)
	browse := davRequest(c, cred, "PROPFIND", "notes.txt", "", map[string]string{"Depth": "0"})
	davStatus(t, browse, 207)
	if !strings.Contains(browse.Body.String(), "activelock") || !strings.Contains(browse.Body.String(), "getlastmodified") {
		t.Fatal(browse.Body.String())
	}
	var owner, fileID string
	ctx := context.Background()
	if e := c.a.DB.QueryRow(ctx, "SELECT t.owner_id::text,f.id::text FROM transfers t JOIN files f ON f.transfer_id=t.id WHERE t.id=$1 AND f.path='notes.txt' AND f.state='ready'", id).Scan(&owner, &fileID); e != nil {
		t.Fatal(e)
	}
	if e := c.a.Files.Cancel(ctx, owner, fileID); e != transfer.ErrLocked {
		t.Fatal("REST/service lock bypass", e)
	}
	var revision int64
	if e := c.a.DB.QueryRow(ctx, "SELECT revision FROM transfers WHERE id=$1", id).Scan(&revision); e != nil {
		t.Fatal(e)
	}
	if _, e := transfer.New(c.a.DB, davSettings(t, c)).Publish(ctx, owner, id, revision, 300); e != transfer.ErrLocked {
		t.Fatal("publish lock bypass", e)
	}
	davStatus(t, davRequest(c, other, "UNLOCK", "notes.txt", "", map[string]string{"Lock-Token": token}), 409)
	davStatus(t, davRequest(c, cred, "UNLOCK", "notes.txt", "", map[string]string{"Lock-Token": token}), 204)
	davStatus(t, davRequest(c, cred, "PUT", "notes.txt", "released", nil), 204)
	var journals int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM download_journal WHERE transfer_id=$1", id).Scan(&journals); e != nil || journals != 0 {
		t.Fatal("owner GET consumed recipient state", journals, e)
	}
	davStatus(t, davRequest(c, cred, "LOCK", "new.txt", davLockBody, map[string]string{"Depth": "0"}), 201)
	if _, e := c.a.DB.Exec(ctx, "UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE id=$1", cred.ID); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, other, "PUT", "new.txt", "revoked lock released", nil), 204)
}

func TestIntegrationWebDAVNativeOverwriteAndUnknownLength(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	davStatus(t, davRequest(c, cred, "PUT", "file.txt", "safe original", nil), 201)
	c.a.Files.Scanner = davRejectScanner{}
	davStatus(t, davRequest(c, cred, "PUT", "file.txt", "bad overwrite", nil), 500)
	c.a.Files.Scanner = nil
	read := davRequest(c, cred, "GET", "file.txt", "", nil)
	davStatus(t, read, 200)
	if read.Body.String() != "safe original" {
		t.Fatal("failed overwrite lost old bytes")
	}
	data := strings.Repeat("unknown-size stream\n", 300000)
	for _, hint := range []bool{false, true} {
		r := httptest.NewRequest("PUT", cred.URL+"unknown.txt", strings.NewReader(data))
		r.ContentLength = -1
		r.TransferEncoding = []string{"chunked"}
		r.RemoteAddr = "127.0.0.1:1234"
		r.SetBasicAuth("filex", cred.Password)
		if hint {
			r.Header.Set("X-Expected-Entity-Length", "6000001")
		} // Deliberately inconsistent with the stream.
		w := httptest.NewRecorder()
		c.h.ServeHTTP(w, r)
		if hint {
			davStatus(t, w, 400)
		} else {
			davStatus(t, w, 201)
		}
	}
	read = davRequest(c, cred, "GET", "unknown.txt", "", nil)
	davStatus(t, read, 200)
	if read.Body.String() != data {
		t.Fatal("unknown-length crypto rewrap corrupted payload")
	}
	davStatus(t, davRequest(c, cred, "MOVE", "unknown.txt", "", map[string]string{"Destination": cred.URL + "file.txt", "Overwrite": "F"}), 412)
	davStatus(t, davRequest(c, cred, "MOVE", "unknown.txt", "", map[string]string{"Destination": cred.URL + "file.txt"}), 204)
	read = davRequest(c, cred, "GET", "file.txt", "", nil)
	davStatus(t, read, 200)
	if read.Body.String() != data {
		t.Fatal("MOVE overwrite lost payload")
	}
}

func TestIntegrationWebDAVNativePropertiesAtomic(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	davStatus(t, davRequest(c, cred, "PUT", "p.txt", "payload", nil), 201)
	body := `<propertyupdate xmlns="DAV:" xmlns:x="urn:test"><set><prop><getlastmodified>Wed, 01 Jan 2025 12:00:00 GMT</getlastmodified><x:note xml:lang="ru">before <x:strong>важно &amp; safe</x:strong> after</x:note></prop></set></propertyupdate>`
	davStatus(t, davRequest(c, cred, "PROPPATCH", "p.txt", body, nil), 207)
	get := davRequest(c, cred, "PROPFIND", "p.txt", "", map[string]string{"Depth": "0"})
	davStatus(t, get, 207)
	if !strings.Contains(get.Body.String(), "важно &amp; safe") || !strings.Contains(get.Body.String(), "Wed, 01 Jan 2025") {
		t.Fatal(get.Body.String())
	}
	rootProperty := `<propertyupdate xmlns="DAV:" xmlns:x="urn:test"><set><prop><x:rootnote>only at root</x:rootnote></prop></set></propertyupdate>`
	davStatus(t, davRequest(c, cred, "PROPPATCH", "", rootProperty, nil), 207)
	c.upload(id, "implicit/child.txt", "tus")
	implicit := davRequest(c, cred, "PROPFIND", "implicit", "", map[string]string{"Depth": "0"})
	davStatus(t, implicit, 207)
	if strings.Contains(implicit.Body.String(), "only at root") {
		t.Fatal("implicit folder inherited root properties")
	}
	bad := `<propertyupdate xmlns="DAV:" xmlns:x="urn:test"><set><prop><x:note>must rollback</x:note><getetag>bad</getetag></prop></set></propertyupdate>`
	w := davRequest(c, cred, "PROPPATCH", "p.txt", bad, nil)
	davStatus(t, w, 207)
	if !strings.Contains(w.Body.String(), "424") || !strings.Contains(w.Body.String(), "403") {
		t.Fatal(w.Body.String())
	}
	get = davRequest(c, cred, "PROPFIND", "p.txt", "", map[string]string{"Depth": "0"})
	if strings.Contains(get.Body.String(), "must rollback") {
		t.Fatal("partial PROPPATCH")
	}
}

func TestIntegrationWebDAVNativeStagingHiddenAndCleanup(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	var owner string
	if e := c.a.DB.QueryRow(ctx, "SELECT owner_id::text FROM transfers WHERE id=$1", id).Scan(&owner); e != nil {
		t.Fatal(e)
	}
	stage, e := c.a.Files.CreateStaged(ctx, owner, id, "hidden.txt", 3, davSettings(t, c), nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Files.Append(ctx, owner, stage.ID, 0, []byte("abc")); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Files.Finalize(ctx, owner, stage.ID, davSettings(t, c)); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "GET", "hidden.txt", "", nil), 404)
	detail := c.req("GET", "/api/v1/transfers/"+id, "", nil)
	var data struct {
		Files    []any
		Revision int64
	}
	if e = json.Unmarshal(detail.Body.Bytes(), &data); e != nil {
		t.Fatal(e)
	}
	if len(data.Files) != 0 {
		t.Fatal("staging exposed")
	}
	if _, e = transfer.New(c.a.DB, davSettings(t, c)).Publish(ctx, owner, id, data.Revision, 300); e != transfer.ErrConflict {
		t.Fatal("staging published", e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE files SET staged_until=clock_timestamp()-interval '1 second',frozen=true WHERE id=$1", stage.ID); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Files.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM files WHERE id=$1", stage.ID).Scan(&state); e != nil || state != "ready" {
		t.Fatal("hold ignored", state, e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE files SET frozen=false WHERE id=$1", stage.ID); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Files.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM files WHERE id=$1", stage.ID).Scan(&state); e != nil || state != "cancelled" {
		t.Fatal("abandoned stage retained", state, e)
	}
}

func davSettings(t *testing.T, c apiClient) config.Settings {
	t.Helper()
	v, e := database.Settings(context.Background(), c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

func TestIntegrationWebDAVNativeCollectionLocks(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	davStatus(t, davRequest(c, cred, "OPTIONS", "", "", nil), 200)
	// GVfs strips the final slash for discovery; both spellings address one root.
	r := httptest.NewRequest("OPTIONS", strings.TrimSuffix(cred.URL, "/"), nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.SetBasicAuth("filex", cred.Password)
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	davStatus(t, w, 200)
	davStatus(t, davRequest(c, cred, "MKCOL", "folder", "", nil), 201)
	davStatus(t, davRequest(c, cred, "PUT", "folder/a.txt", "original", nil), 201)
	locked := davRequest(c, cred, "LOCK", "folder", davLockBody, map[string]string{"Depth": "0"})
	davStatus(t, locked, 200)
	token := locked.Header().Get("Lock-Token")
	davStatus(t, davRequest(c, cred, "PUT", "folder/a.txt", "child bytes remain editable", nil), 204)
	davStatus(t, davRequest(c, cred, "PUT", "folder/b.txt", "membership change", nil), 423)
	davStatus(t, davRequest(c, cred, "MKCOL", "folder/sub", "", nil), 423)
	davStatus(t, c.req("POST", "/api/v1/transfers/"+id+"/folders", `{"path":"folder/rest"}`, nil), 409)
	davStatus(t, davRequest(c, cred, "DELETE", "folder/a.txt", "", nil), 423)
	var fileID string
	if e := c.a.DB.QueryRow(ctx, "SELECT id::text FROM files WHERE transfer_id=$1 AND path='folder/a.txt' AND state='ready' AND staged_until IS NULL", id).Scan(&fileID); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(map[string]any{"ids": []string{fileID}, "action": "cancel_upload"})
	job := c.req("POST", "/api/v1/bulk", string(raw), nil)
	davStatus(t, job, 202)
	wk, e := worker.New(c.a.DB, c.a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	if e = wk.Bulk(ctx); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = c.a.DB.QueryRow(ctx, "SELECT state FROM files WHERE id=$1", fileID).Scan(&state); e != nil || state != "ready" {
		t.Fatal("bulk bypassed membership lock", state, e)
	}
	tagged := map[string]string{"If": "<" + cred.URL + "folder> (" + token + ")"}
	davStatus(t, davRequest(c, cred, "PUT", "folder/b.txt", "authorized", tagged), 201)
	// Creating a locked empty child also changes the Depth:0 parent's membership.
	// Its positive parent condition must survive the internal zero-byte PUT.
	childLock := davRequest(c, cred, "LOCK", "folder/c.txt", davLockBody, tagged)
	davStatus(t, childLock, 201)
	davStatus(t, davRequest(c, cred, "UNLOCK", "folder/c.txt", "", map[string]string{"Lock-Token": childLock.Header().Get("Lock-Token")}), 204)
	if _, e = c.a.DB.Exec(ctx, "UPDATE dav_locks SET expires_at=clock_timestamp()-interval '1 second' WHERE transfer_id=$1", id); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "DELETE", "folder/b.txt", "", nil), 204)
}

func TestIntegrationWebDAVNativeMetadataBounds(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	davStatus(t, davRequest(c, cred, "PUT", "file.txt", "content", nil), 201)
	owner := strings.Replace(davLockBody, "local test client", strings.Repeat("x", 4096), 1)
	davStatus(t, davRequest(c, cred, "LOCK", "file.txt", owner, nil), 413)
	// Seed valid, individually bounded properties just below the aggregate limit.
	// A single additional PROPPATCH must roll back rather than poison browsing.
	ctx := context.Background()
	raw, _ := json.Marshal([]davStoredProperty{{Space: "urn:test", Local: "note", Inner: strings.Repeat("x", 31400)}})
	if _, e := c.a.DB.Exec(ctx, `INSERT INTO draft_folders(id,transfer_id,path,path_key,dav_properties) SELECT gen_random_uuid(),$1,'folder-'||n,'FOLDER-'||n,$2::jsonb FROM generate_series(1,133) n`, id, raw); e != nil {
		t.Fatal(e)
	}
	patch := `<propertyupdate xmlns="DAV:"><set><prop><x:note xmlns:x="urn:test">` + strings.Repeat("y", 30000) + `</x:note></prop></set></propertyupdate>`
	davStatus(t, davRequest(c, cred, "PROPPATCH", "", patch, nil), 507)
	var unchanged bool
	if e := c.a.DB.QueryRow(ctx, "SELECT dav_properties='[]'::jsonb FROM transfers WHERE id=$1", id).Scan(&unchanged); e != nil || !unchanged {
		t.Fatal("property budget was not atomic", e)
	}
	davStatus(t, davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "0"}), 207)
}

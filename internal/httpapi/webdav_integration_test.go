package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/files"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type davTestCredential struct{ ID, URL, Username, Password, ExpiresAt string }

func davCredential(t *testing.T, c apiClient, id string) davTestCredential {
	t.Helper()
	w := c.req("POST", "/api/v1/transfers/"+id+"/webdav-credentials", `{"ttlSeconds":3600}`, nil)
	if w.Code != 201 {
		t.Fatal("credential", w.Code, w.Body.String())
	}
	var v davTestCredential
	decode(t, w, &v)
	return v
}
func davRequest(c apiClient, cred davTestCredential, method, name, body string, headers map[string]string) *httptest.ResponseRecorder {
	u := strings.TrimSuffix(cred.URL, "/") + "/" + name
	r := httptest.NewRequest(method, u, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:9876"
	r.SetBasicAuth(cred.Username, cred.Password)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	return w
}
func davStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status=%d want=%d body=%s", w.Code, status, w.Body.String())
	}
}

func TestIntegrationWebDAVUploadTreeAndPublish(t *testing.T) {
	c, root := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	davStatus(t, davRequest(c, cred, "OPTIONS", "", "", nil), 200)
	davStatus(t, davRequest(c, cred, "MKCOL", "folder/", "", nil), 201)
	name := "folder/договор & новый.txt"
	escaped := (&url.URL{Path: name}).EscapedPath()
	data := strings.Repeat("private upload payload\n", 200000)
	w := davRequest(c, cred, "PUT", escaped, data, nil)
	davStatus(t, w, 201)
	var fileID, state, checksum string
	var offset, size int64
	e := c.a.DB.QueryRow(ctx, "SELECT id::text,state,sha256,size,upload_offset FROM files WHERE transfer_id=$1 AND path=$2", id, name).Scan(&fileID, &state, &checksum, &size, &offset)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256([]byte(data))
	if state != "ready" || checksum != hex.EncodeToString(sum[:]) || size != int64(len(data)) || offset != size {
		t.Fatal("server finalize", state, size, offset)
	}
	f, e := c.a.Files.Get(ctx, "", fileID)
	if e == nil {
		t.Fatal("empty owner read allowed", f)
	}
	var owner string
	if e = c.a.DB.QueryRow(ctx, "SELECT owner_id::text FROM transfers WHERE id=$1", id).Scan(&owner); e != nil {
		t.Fatal(e)
	}
	f, e = c.a.Files.Get(ctx, owner, fileID)
	if e != nil {
		t.Fatal(e)
	}
	var clear bytes.Buffer
	if e = c.a.Files.Read(ctx, f, 0, f.Size, &clear); e != nil {
		t.Fatal(e)
	}
	if clear.String() != data {
		t.Fatal("payload mismatch")
	}
	objects := 0
	if e = filepath.WalkDir(root, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		objects++
		if bytes.Contains(b, []byte("private upload payload")) {
			t.Error("plaintext in storage")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if objects != 2 {
		t.Fatal("expected bounded encrypted chunks", objects)
	}
	w = davRequest(c, cred, "PROPFIND", "folder/", `<d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:resourcetype/><d:getcontentlength/><d:getetag/><x:missing xmlns:x="urn:test"/></d:prop></d:propfind>`, map[string]string{"Depth": "1"})
	davStatus(t, w, 207)
	var multi struct {
		Responses []struct {
			Href  string `xml:"DAV: href"`
			Stats []struct {
				Status string `xml:"DAV: status"`
			} `xml:"DAV: propstat"`
		} `xml:"DAV: response"`
	}
	if e = xml.Unmarshal(w.Body.Bytes(), &multi); e != nil {
		t.Fatal(e)
	}
	if len(multi.Responses) != 2 || !strings.Contains(w.Body.String(), "404 Not Found") || !strings.Contains(w.Body.String(), "%D0") {
		t.Fatal("browse", w.Body.String())
	}
	davStatus(t, davRequest(c, cred, "HEAD", escaped, "", nil), 200)
	davStatus(t, davRequest(c, cred, "GET", escaped, "", nil), 200)
	davStatus(t, davRequest(c, cred, "PUT", escaped, "overwrite", nil), 204)
	davStatus(t, davRequest(c, cred, "PUT", escaped, "overwrite", map[string]string{"If-None-Match": "*"}), 412)
	davStatus(t, davRequest(c, cred, "MOVE", "folder/", "", map[string]string{"Destination": cred.URL + "renamed/", "Overwrite": "F"}), 201)
	davStatus(t, davRequest(c, cred, "HEAD", "folder/", "", nil), 404)
	davStatus(t, davRequest(c, cred, "PUT", "empty.txt", "", nil), 201)
	davStatus(t, davRequest(c, cred, "DELETE", "empty.txt", "", nil), 204)
	// A normal tus path creates implicit DAV ancestors without a metadata write.
	c.upload(id, "implicit/child/file.txt", "tus")
	c.upload(id, "IMPLICIT/Child/second.txt", "second")
	davStatus(t, davRequest(c, cred, "MOVE", "implicit/", "", map[string]string{"Destination": cred.URL + "moved/"}), 201)
	davStatus(t, davRequest(c, cred, "HEAD", "moved/child/file.txt", "", nil), 200)
	davStatus(t, davRequest(c, cred, "HEAD", "moved/child/second.txt", "", nil), 200)
	davStatus(t, davRequest(c, cred, "HEAD", "implicit/", "", nil), 404)
	davStatus(t, davRequest(c, cred, "DELETE", "moved/", "", nil), 204)
	davStatus(t, davRequest(c, cred, "HEAD", "moved/", "", nil), 404)
	// Unicode case folding may change byte length, so strip ancestors by segment.
	c.upload(id, "ſeed/child/a.txt", "a")
	c.upload(id, "SEED/CHILD/b.txt", "b")
	davStatus(t, davRequest(c, cred, "MOVE", "seed/", "", map[string]string{"Destination": cred.URL + "unicode/"}), 201)
	davStatus(t, davRequest(c, cred, "HEAD", "unicode/child/a.txt", "", nil), 200)
	davStatus(t, davRequest(c, cred, "HEAD", "unicode/child/b.txt", "", nil), 200)
	davStatus(t, davRequest(c, cred, "DELETE", "unicode/child/", "", nil), 204)
	davStatus(t, davRequest(c, cred, "HEAD", "unicode/", "", nil), 404)
	c.publish(id)
	for _, method := range []string{"OPTIONS", "PROPFIND", "PUT", "DELETE", "MOVE", "HEAD"} {
		davStatus(t, davRequest(c, cred, method, "", "", map[string]string{"Depth": "1"}), 403)
	}
}

func TestIntegrationWebDAVMetadataBounds(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	if _, e := c.a.DB.Exec(ctx, `INSERT INTO draft_folders(id,transfer_id,path,path_key) SELECT gen_random_uuid(),$1,'folder-'||n,'FOLDER-'||n FROM generate_series(1,1000) AS n`, id); e != nil {
		t.Fatal(e)
	}
	// A bounded input can still amplify a requested namespace across all children.
	query := `<d:propfind xmlns:d="DAV:"><d:prop><x:missing xmlns:x="urn:` + strings.Repeat("a", 16000) + `"/></d:prop></d:propfind>`
	w := davRequest(c, cred, "PROPFIND", "", query, map[string]string{"Depth": "1"})
	davStatus(t, w, 507)
	if strings.Contains(w.Body.String(), "multistatus") || w.Body.Len() > 1000 {
		t.Fatal("partial multistatus sent on response limit")
	}
	davStatus(t, davRequest(c, cred, "PROPFIND", "", query, map[string]string{"Depth": "0"}), 207)
	davStatus(t, davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "1"}), 207)
	// Metadata requests share the same global concurrency budget as uploads.
	for range cap(c.a.streamSlots) {
		c.a.streamSlots <- struct{}{}
	}
	w = davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "0"})
	for range cap(c.a.streamSlots) {
		<-c.a.streamSlots
	}
	davStatus(t, w, http.StatusTooManyRequests)
	if _, e := c.a.DB.Exec(ctx, `INSERT INTO draft_folders(id,transfer_id,path,path_key) SELECT gen_random_uuid(),$1,'folder-'||n,'FOLDER-'||n FROM generate_series(1001,9999) AS n`, id); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "0"}), 207)
	davStatus(t, davRequest(c, cred, "MKCOL", "overflow", "", nil), 507)
	davStatus(t, davRequest(c, cred, "PUT", "overflow.txt", "x", nil), 507)
	var count int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1", id).Scan(&count); e != nil || count != 0 {
		t.Fatal("metadata limit did not roll back upload reservation", count, e)
	}
}

func TestIntegrationWebDAVCredentialsAndBoundaries(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	bad := cred
	bad.Password = "wrong"
	w := davRequest(c, bad, "OPTIONS", "", "", nil)
	davStatus(t, w, 401)
	if !strings.HasPrefix(w.Header().Get("WWW-Authenticate"), "Basic ") {
		t.Fatal("missing challenge")
	}
	r := httptest.NewRequest("PROPFIND", cred.URL, nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.AddCookie(c.cookie)
	w = httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	davStatus(t, w, 401)
	other := c.draft()
	cross := cred
	cross.URL = strings.Replace(cred.URL, id, other, 1)
	davStatus(t, davRequest(c, cross, "PROPFIND", "", "", map[string]string{"Depth": "0"}), 403)
	davStatus(t, c.req("GET", "/api/v1/transfers", "", map[string]string{"Authorization": "Bearer " + cred.Password}), 403)
	davStatus(t, davRequest(c, cred, "MKCOL", "folder", "", map[string]string{"Origin": "https://evil.example"}), 403)
	davStatus(t, davRequest(c, cred, "PROPFIND", "", "", nil), 403)
	davStatus(t, davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "2"}), 400)
	davStatus(t, davRequest(c, cred, "PROPFIND", "", `<!DOCTYPE x><x/>`, map[string]string{"Depth": "0"}), 400)
	davStatus(t, davRequest(c, cred, "PROPFIND", "", strings.Repeat("x", 65537), map[string]string{"Depth": "0"}), 413)
	davStatus(t, davRequest(c, cred, "MKCOL", "missing/child", "", nil), 409)
	davStatus(t, davRequest(c, cred, "MKCOL", "folder", "body", nil), 415)
	davStatus(t, davRequest(c, cred, "PUT", "missing/file.txt", "x", nil), 409)
	davStatus(t, davRequest(c, cred, "PUT", "range.txt", "x", map[string]string{"Content-Range": "bytes 0-0/2"}), 400)
	davStatus(t, davRequest(c, cred, "PUT", "gzip.txt", "x", map[string]string{"Content-Encoding": "gzip"}), 415)
	for _, method := range []string{"TRACE"} {
		davStatus(t, davRequest(c, cred, method, "", "", nil), 405)
	}
	for _, name := range []string{"../secret", "a%2Fb", "a%5Cb", "NUL", "a?secret=value"} {
		davStatus(t, davRequest(c, cred, "PUT", name, "x", nil), 400)
	}
	davStatus(t, davRequest(c, cred, "PUT", "file.txt", "x", nil), 201)
	for _, destination := range []string{"https://evil.example/file", "http://localhost:8090" + davPrefix + other + "/x"} {
		davStatus(t, davRequest(c, cred, "MOVE", "file.txt", "", map[string]string{"Destination": destination}), 403)
	}
	davStatus(t, davRequest(c, cred, "MOVE", "file.txt", "", map[string]string{"Destination": cred.URL + "other.txt", "If": "(<opaquelocktoken:untrusted>)"}), 412)
	davStatus(t, davRequest(c, cred, "PROPPATCH", "file.txt", `<d:propertyupdate xmlns:d="DAV:"><d:set><d:prop><d:getetag>client</d:getetag></d:prop></d:set></d:propertyupdate>`, nil), 207)
	if _, e := c.a.DB.Exec(ctx, "UPDATE transfers SET frozen=true WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "PUT", "blocked.txt", "x", nil), 403)
	if _, e := c.a.DB.Exec(ctx, "UPDATE transfers SET frozen=false WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	davStatus(t, c.req("DELETE", "/api/v1/auth/tokens/"+cred.ID, "", nil), 204)
	w = davRequest(c, cred, "PROPFIND", "", "", map[string]string{"Depth": "0"})
	davStatus(t, w, 401)
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("revocation challenge missing")
	}
	cred = davCredential(t, c, id)
	if _, e := c.a.DB.Exec(ctx, "UPDATE api_tokens SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", cred.ID); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "OPTIONS", "", "", nil), 401)
	cred = davCredential(t, c, id)
	if _, e := c.a.DB.Exec(ctx, "UPDATE users SET auth_epoch=auth_epoch+1 WHERE id=(SELECT owner_id FROM transfers WHERE id=$1)", id); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "OPTIONS", "", "", nil), 401)
}

type davFaultReader struct {
	data  *strings.Reader
	after func()
	fired bool
}

func (r *davFaultReader) Read(b []byte) (int, error) {
	n, e := r.data.Read(b)
	if !r.fired {
		r.fired = true
		r.after()
	}
	return n, e
}

type davRejectScanner struct{}

func (davRejectScanner) Scan(context.Context, string, func(io.Writer) error) error {
	return errors.New("scanner rejected fixture")
}

type davActionScanner func() error

func (f davActionScanner) Scan(context.Context, string, func(io.Writer) error) error { return f() }

func TestIntegrationWebDAVFinalizationAndStorageFaults(t *testing.T) {
	c, root := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	c.a.Files.Scanner = davActionScanner(func() error {
		_, e := c.a.DB.Exec(ctx, "UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE id=$1", cred.ID)
		return e
	})
	davStatus(t, davRequest(c, cred, "PUT", "revoked-finalize.txt", "payload", nil), 401)
	var count int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state='ready'", id).Scan(&count); e != nil || count != 0 {
		t.Fatal("revoked finalize committed", count, e)
	}
	c.a.Files.Scanner = nil
	for range 3 {
		if e := c.a.Files.Cleanup(ctx); e != nil {
			t.Fatal(e)
		}
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) != 0 {
		t.Fatal("cancelled chunks not physically cleaned", len(entries), e)
	}
	cred = davCredential(t, c, id)
	body := &davFaultReader{data: strings.NewReader("data"), after: func() {
		if e := os.Remove(root); e != nil {
			t.Fatal(e)
		}
	}}
	r := httptest.NewRequest("PUT", cred.URL+"storage-fault.txt", body)
	r.ContentLength = 4
	r.RemoteAddr = "127.0.0.1:1234"
	r.SetBasicAuth("filex", cred.Password)
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	davStatus(t, w, 500)
	if e = c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted')", id).Scan(&count); e != nil || count != 0 {
		t.Fatal("storage fault committed", count, e)
	}
	if e = os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "PUT", "storage-fault.txt", "retry", nil), 201)
}

func TestIntegrationWebDAVTransportAndMaintenance(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	cfg, e := database.Settings(ctx, c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	cfg.PublicOrigin = "https://files.example.test"
	cfg.NetworkPolicy = config.NetworkPolicy{TrustedProxyCIDRs: []string{"10.0.0.2/32"}, RequireProxy: true, RequireHTTPS: true, AllowedClientCIDRs: []string{"203.0.113.0/24"}}
	raw, e := json.Marshal(cfg)
	if e != nil {
		t.Fatal(e)
	}
	// Direct fixture setup isolates header/auth integration; real policy CAS and
	// runtime restart are covered by the network/ingress regression suites.
	if _, e = c.a.DB.Exec(ctx, "UPDATE app_settings SET value=$1 WHERE active", raw); e != nil {
		t.Fatal(e)
	}
	call := func(peer, proto, forwarded, host string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PROPFIND", "http://"+host+davPrefix+id+"/", nil)
		r.RemoteAddr = peer
		r.SetBasicAuth("filex", cred.Password)
		r.Header.Set("Depth", "0")
		r.Header.Set("X-Forwarded-For", forwarded)
		r.Header.Set("X-Forwarded-Proto", proto)
		w := httptest.NewRecorder()
		c.h.ServeHTTP(w, r)
		return w
	}
	davStatus(t, call("10.0.0.2:1234", "https", "203.0.113.4", "files.example.test"), 207)
	davStatus(t, call("10.0.0.2:1234", "http", "203.0.113.4", "files.example.test"), 403)
	davStatus(t, call("10.0.0.3:1234", "https", "203.0.113.4", "files.example.test"), 403)
	davStatus(t, call("10.0.0.2:1234", "https", "198.51.100.1", "files.example.test"), 403)
	davStatus(t, call("10.0.0.2:1234", "https", "203.0.113.4", "evil.example.test"), 403)
	if _, e = c.a.DB.Exec(ctx, "UPDATE ops_control SET mode='maintenance'"); e != nil {
		t.Fatal(e)
	}
	davStatus(t, call("10.0.0.2:1234", "https", "203.0.113.4", "files.example.test"), 503)
}

func TestIntegrationWebDAVInterruptedAndRevokedWrites(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		length int64
		want   int
	}{{"truncated", files.ChunkSize + 1, 400}, {"oversized", 1<<40 + 1, 413}} {
		r := httptest.NewRequest("PUT", cred.URL+tc.name, strings.NewReader(strings.Repeat("a", files.ChunkSize)))
		r.RemoteAddr = "127.0.0.1:1234"
		r.ContentLength = tc.length
		r.SetBasicAuth("filex", cred.Password)
		w := httptest.NewRecorder()
		c.h.ServeHTTP(w, r)
		davStatus(t, w, tc.want)
	}
	var ready, pending int
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted')", id).Scan(&ready); e != nil || ready != 0 {
		t.Fatal("incomplete resource escaped", ready, e)
	}
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM deletion_jobs WHERE file_id IN (SELECT id FROM files WHERE transfer_id=$1)", id).Scan(&pending); e != nil || pending != 1 {
		t.Fatal("missing chunk compensation", pending, e)
	}
	body := &davFaultReader{data: strings.NewReader("data"), after: func() {
		if _, e := c.a.DB.Exec(ctx, "UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE id=$1", cred.ID); e != nil {
			t.Fatal(e)
		}
	}}
	r := httptest.NewRequest("PUT", cred.URL+"revoked", body)
	r.ContentLength = 4
	r.RemoteAddr = "127.0.0.1:1234"
	r.SetBasicAuth("filex", cred.Password)
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	davStatus(t, w, 401)
	cred = davCredential(t, c, id)
	c.a.Files.Scanner = davRejectScanner{}
	davStatus(t, davRequest(c, cred, "PUT", "scanner", "must not be ready", nil), 500)
	if e := c.a.DB.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted')", id).Scan(&ready); e != nil || ready != 0 {
		t.Fatal("failed upload retained as ready", ready, e)
	}
	c.a.Files.Scanner = nil
	davStatus(t, davRequest(c, cred, "PUT", "scanner", "retry after explicit failure", nil), 201)
}

func TestIntegrationWebDAVConcurrentNamesAndFreeze(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	var group sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		group.Go(func() {
			codes <- davRequest(c, cred, "PUT", "same.txt", "content", map[string]string{"If-None-Match": "*"}).Code
		})
	}
	group.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		if code == 201 {
			success++
		} else if code == 412 {
			conflict++
		} else {
			t.Fatal("unexpected race result", code)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
	ctx := context.Background()
	var fileID, owner string
	if e := c.a.DB.QueryRow(ctx, "SELECT f.id::text,t.owner_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE t.id=$1 AND f.path='same.txt' AND f.staged_until IS NULL AND f.state='ready'", id).Scan(&fileID, &owner); e != nil {
		t.Fatal(e)
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE files SET frozen=true WHERE id=$1", fileID); e != nil {
		t.Fatal(e)
	}
	davStatus(t, davRequest(c, cred, "DELETE", "same.txt", "", nil), 403)
	if e := c.a.Files.Cancel(ctx, owner, fileID); e == nil {
		t.Fatal("shared cancellation ignored hold")
	}
	if _, e := c.a.DB.Exec(ctx, "UPDATE files SET frozen=false WHERE id=$1", fileID); e != nil {
		t.Fatal(e)
	}
	settings, e := database.Settings(ctx, c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	u, e := c.a.Files.Create(ctx, owner, id, "freeze.txt", 4, settings)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE transfers SET frozen=true WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	if _, e = c.a.Files.Append(ctx, owner, u.ID, 0, []byte("data")); e == nil {
		t.Fatal("frozen write allowed")
	}
	if _, e = c.a.Files.Create(ctx, owner, id, "new.txt", 0, settings); e == nil {
		t.Fatal("frozen reservation allowed")
	}
}

func TestIntegrationWebDAVCredentialIssuePrivacy(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	ctx := context.Background()
	var raw string
	if e := c.a.DB.QueryRow(ctx, "SELECT token_hash FROM api_tokens WHERE id=$1", cred.ID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if raw != auth.Digest(cred.Password) || raw == cred.Password {
		t.Fatal("credential not digested")
	}
	for _, ttl := range []string{"0", "2592001"} {
		davStatus(t, c.req("POST", "/api/v1/transfers/"+id+"/webdav-credentials", `{"ttlSeconds":`+ttl+`}`, nil), 400)
	}
	davStatus(t, c.req("POST", "/api/v1/transfers/"+id+"/webdav-credentials", `{"ttlSeconds":3600}`, map[string]string{"X-CSRF-Token": "wrong"}), 403)
	if _, e := c.a.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp()-interval '10 minutes'"); e != nil {
		t.Fatal(e)
	}
	davStatus(t, c.req("POST", "/api/v1/transfers/"+id+"/webdav-credentials", `{"ttlSeconds":3600}`, nil), 403)
}

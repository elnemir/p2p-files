package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type interceptWriter struct {
	*httptest.ResponseRecorder
	after  func()
	writes int
}

func (w *interceptWriter) Write(b []byte) (int, error) {
	n, e := w.ResponseRecorder.Write(b)
	w.writes++
	if w.after != nil {
		f := w.after
		w.after = nil
		f()
	}
	return n, e
}
func TestIntegrationFreezeStopsWriterAndPreservesPayload(t *testing.T) {
	c, root := testClient(t)
	id := c.draft()
	file := c.upload(id, "large.txt", strings.Repeat("z", 128*1024))
	r := c.recipient(c.publish(id))
	url := r.lease(`{"all":true}`)
	req := httptest.NewRequest("GET", url, nil)
	req.AddCookie(r.cookie)
	rec := &interceptWriter{ResponseRecorder: httptest.NewRecorder()}
	rec.after = func() {
		w := c.req("POST", "/api/v1/admin/transfers/"+id+"/hold", `{"frozen":true,"reason":"incident test"}`, nil)
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	start := time.Now()
	aborted := false
	func() {
		defer func() {
			if x := recover(); x != nil {
				if x != http.ErrAbortHandler {
					panic(x)
				}
				aborted = true
			}
		}()
		r.h.ServeHTTP(rec, req)
	}()
	if !aborted || rec.writes != 1 || time.Since(start) > 2*time.Second {
		t.Fatal("freeze guard", aborted, rec.writes, time.Since(start))
	}
	if e := c.a.Files.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(root)
	if e != nil || len(entries) == 0 {
		t.Fatal("frozen bytes removed")
	}
	var state string
	if e = c.a.DB.QueryRow(context.Background(), "SELECT state FROM files WHERE id=$1", file).Scan(&state); e != nil || state == "deleting" {
		t.Fatal("partial ZIP consumed", state, e)
	}
	w := c.req("POST", "/api/v1/transfers/"+id+"/revoke", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	if e = c.a.Files.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	entries, _ = os.ReadDir(root)
	if len(entries) == 0 {
		t.Fatal("revoke discarded hold")
	}
	w = c.req("POST", "/api/v1/admin/transfers/"+id+"/hold", `{"frozen":false,"reason":"investigation finished"}`, nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e = c.a.Files.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	entries, _ = os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("cleanup after unfreeze", len(entries))
	}
	if e = c.a.DB.QueryRow(context.Background(), "SELECT state FROM transfers WHERE id=$1", id).Scan(&state); e != nil || state != "revoked" {
		t.Fatal("terminal resurrected", state, e)
	}
}
func TestIntegrationIntegrityFailureRetainedOnExpiry(t *testing.T) {
	c, root := testClient(t)
	id := c.draft()
	file := c.upload(id, "a.txt", "secret bytes")
	r := c.recipient(c.publish(id))
	var object string
	if e := c.a.DB.QueryRow(context.Background(), "SELECT object_key FROM file_chunks WHERE file_id=$1", file).Scan(&object); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(root, object))
	if e != nil {
		t.Fatal(e)
	}
	b[len(b)-1] ^= 1
	if e = os.WriteFile(filepath.Join(root, object), b, 0600); e != nil {
		t.Fatal(e)
	}
	url := r.lease(`{"all":true}`)
	aborted := false
	func() {
		defer func() {
			if x := recover(); x != nil {
				if x != http.ErrAbortHandler {
					panic(x)
				}
				aborted = true
			}
		}()
		r.req("GET", url, "", nil)
	}()
	if !aborted {
		t.Fatal("corruption accepted")
	}
	var state string
	if e = c.a.DB.QueryRow(context.Background(), "SELECT state FROM files WHERE id=$1", file).Scan(&state); e != nil || state != "integrity_failed" {
		t.Fatal(state, e)
	}
	if _, e = c.a.DB.Exec(context.Background(), "UPDATE transfers SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Files.Sweep(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e = c.a.Files.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(root, object)); e != nil {
		t.Fatal("evidence lost", e)
	}
	w := c.req("POST", "/api/v1/admin/files/"+file+"/forensic", `{"reason":"examine ciphertext","rawCiphertext":true,"objectKey":"`+object+`"}`, nil)
	if w.Code != 200 || len(w.Body.Bytes()) != len(b) {
		t.Fatal(w.Code, w.Body.String())
	}
	var count int
	if e = c.a.DB.QueryRow(context.Background(), "SELECT count(*) FROM download_journal").Scan(&count); e != nil || count != 0 {
		t.Fatal("failed ZIP journal", e, count)
	}
}
func TestIntegrationFilePolicyAndPIN(t *testing.T) {
	c, _ := testClient(t)
	_, e := c.a.DB.Exec(context.Background(), `UPDATE app_settings SET value=value||'{"fileTypeMode":"deny","fileTypes":["text/plain"]}'::jsonb WHERE active`)
	if e != nil {
		t.Fatal(e)
	}
	id := c.draft()
	w := c.req("POST", "/api/v1/transfers/"+id+"/uploads", "", map[string]string{"Tus-Resumable": "1.0.0", "Upload-Length": "3", "Upload-Metadata": "filename YS50eHQ="})
	url := w.Header().Get("Location")
	w = c.req("PATCH", url, "abc", map[string]string{"Tus-Resumable": "1.0.0", "Upload-Offset": "0", "Content-Type": "application/offset+octet-stream"})
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = c.req("POST", url+"/finalize", "", nil)
	if w.Code != 400 {
		t.Fatal("deny bypass", w.Code, w.Body.String())
	}
	if _, e = c.a.DB.Exec(context.Background(), `UPDATE app_settings SET value=value||'{"fileTypeMode":"disabled"}'::jsonb WHERE active`); e != nil {
		t.Fatal(e)
	}
	id = c.draft()
	c.upload(id, "b.txt", "abc")
	w = c.req("GET", "/api/v1/transfers/"+id, "", nil)
	var detail struct{ Revision int64 }
	decode(t, w, &detail)
	request, _ := json.Marshal(map[string]any{"expectedRevision": detail.Revision, "ttlSeconds": 60, "recipients": []map[string]string{{"email": "recipient@example.test", "password": "683947"}}})
	w = c.req("POST", "/api/v1/transfers/"+id+"/publish", string(request), nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ Links []struct{ URL string } }
	decode(t, w, &v)
	token := strings.Split(v.Links[0].URL, "#token=")[1]
	r := apiClient{t: t, a: c.a, h: c.h}
	w = r.req("POST", "/api/v1/recipient/session", `{"token":"`+token+`","password":"111111"}`, nil)
	if w.Code != 401 {
		t.Fatal("PIN bypass")
	}
	w = r.req("POST", "/api/v1/recipient/session", `{"token":"`+token+`","password":"683947"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

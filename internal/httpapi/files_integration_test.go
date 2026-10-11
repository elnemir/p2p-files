package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"filex.local/filex/internal/domain"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type apiClient struct {
	t      *testing.T
	a      *App
	h      http.Handler
	cookie *http.Cookie
	csrf   string
}

func testClient(t *testing.T) (apiClient, string) {
	a, p := integrationApp(t)
	root := t.TempDir()
	raw, _ := json.Marshal(map[string]string{"root": root})
	if _, e := p.Exec(context.Background(), "INSERT INTO storage_providers(id,kind,config) VALUES($1,'local',$2)", domain.NewID(), raw); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Auth.CreateUser(context.Background(), "host_cli", "sender@example.test", "strong-password-0123", "admin"); e != nil {
		t.Fatal(e)
	}
	c := apiClient{t: t, a: a, h: a.Handler()}
	w := c.req("POST", "/api/v1/auth/login", `{"email":"sender@example.test","password":"strong-password-0123"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var v struct {
		CSRF string `json:"csrfToken"`
	}
	decode(t, w, &v)
	c.cookie = w.Result().Cookies()[0]
	c.csrf = v.CSRF
	return c, root
}
func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if e := json.Unmarshal(w.Body.Bytes(), v); e != nil {
		t.Fatal(e, w.Code, w.Body.String())
	}
}
func (c apiClient) req(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	c.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:9876"
	r.Header.Set("Origin", "http://localhost:8090")
	r.Header.Set("X-CSRF-Token", c.csrf)
	if strings.HasPrefix(path, "/api/v1/uploads/") {
		r.Header.Set("Tus-Resumable", "1.0.0")
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	return w
}
func (c apiClient) draft() string {
	w := c.req("POST", "/api/v1/transfers", "", nil)
	if w.Code != 201 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ ID string }
	decode(c.t, w, &v)
	return v.ID
}
func (c apiClient) upload(id, path, data string) string {
	w := c.req("POST", "/api/v1/transfers/"+id+"/uploads", "", map[string]string{"Tus-Resumable": "1.0.0", "Upload-Length": fmt.Sprint(len(data)), "Upload-Metadata": "filename " + base64.StdEncoding.EncodeToString([]byte(path))})
	if w.Code != 201 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	url := w.Header().Get("Location")
	if len(data) > 0 {
		w = c.req("PATCH", url, data, map[string]string{"Tus-Resumable": "1.0.0", "Content-Type": "application/offset+octet-stream", "Upload-Offset": "0"})
		if w.Code != 204 {
			c.t.Fatal(w.Code, w.Body.String())
		}
		bad := c.req("PATCH", url, "x", map[string]string{"Tus-Resumable": "1.0.0", "Content-Type": "application/offset+octet-stream", "Upload-Offset": "0"})
		if bad.Code != 409 {
			c.t.Fatal("offset replay", bad.Code, bad.Body.String())
		}
	}
	w = c.req("POST", url+"/finalize", "", nil)
	if w.Code != 204 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	return strings.TrimPrefix(url, "/api/v1/uploads/")
}
func (c apiClient) publish(id string) string {
	w := c.req("GET", "/api/v1/transfers/"+id, "", nil)
	var d struct{ Revision int64 }
	decode(c.t, w, &d)
	w = c.req("POST", "/api/v1/transfers/"+id+"/publish", fmt.Sprintf(`{"expectedRevision":%d,"ttlSeconds":3600,"commonLink":true}`, d.Revision), nil)
	if w.Code != 200 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	var v struct {
		Links []struct {
			URL string `json:"url"`
		}
	}
	decode(c.t, w, &v)
	if len(v.Links) != 1 {
		c.t.Fatal(w.Body.String())
	}
	return strings.Split(v.Links[0].URL, "#token=")[1]
}
func (c apiClient) recipient(token string) apiClient {
	r := apiClient{t: c.t, a: c.a, h: c.h}
	w := r.req("POST", "/api/v1/recipient/session", `{"token":"`+token+`"}`, nil)
	if w.Code != 200 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	r.cookie = w.Result().Cookies()[0]
	return r
}
func (c apiClient) refs() []map[string]any {
	w := c.req("GET", "/api/v1/recipient/files", "", nil)
	if w.Code != 200 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	var v []map[string]any
	decode(c.t, w, &v)
	return v
}
func (c apiClient) lease(body string) string {
	w := c.req("POST", "/api/v1/recipient/downloads", body, nil)
	if w.Code != 201 {
		c.t.Fatal(w.Code, w.Body.String())
	}
	var v struct{ URL string }
	decode(c.t, w, &v)
	return v.URL
}
func TestIntegrationOneTimeRangesZIPAndCleanup(t *testing.T) {
	c, root := testClient(t)
	_ = root
	id := c.draft()
	c.upload(id, "first.txt", "abcdefghij")
	c.upload(id, "папка/second.txt", "zip payload")
	token := c.publish(id)
	r := c.recipient(token)
	refs := r.refs()
	if _, found := refs[0]["id"]; found {
		t.Fatal("raw recipient id leaked")
	}
	var count int
	if e := c.a.DB.QueryRow(context.Background(), "SELECT count(*) FROM downloads").Scan(&count); e != nil || count != 0 {
		t.Fatal("GET consumed data")
	}
	url := r.lease(`{"fileRef":"` + refs[0]["ref"].(string) + `"}`)
	w := r.req("HEAD", url, "", nil)
	if w.Code != 200 || w.Header().Get("Content-Length") != "10" {
		t.Fatal(w.Code, w.Header())
	}
	w = r.req("GET", url, "", map[string]string{"Range": "bytes=0-3"})
	if w.Code != 206 || w.Body.String() != "abcd" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = r.req("GET", url, "", map[string]string{"Range": "bytes=2-6"})
	if w.Body.String() != "cdefg" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = r.req("GET", url, "", map[string]string{"Range": "bytes=7-"})
	if w.Body.String() != "hij" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = r.req("GET", url, "", nil)
	if w.Code != 404 {
		t.Fatal("consumed replay", w.Code)
	}
	zipURL := r.lease(`{"all":true}`)
	w = r.req("GET", zipURL, "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	z, e := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if e != nil {
		t.Fatal(e)
	}
	if len(z.File) != 4 {
		t.Fatal("ZIP membership", len(z.File))
	}
	f, e := z.Open("папка/second.txt")
	if e != nil {
		t.Fatal(e)
	}
	data, e := io.ReadAll(f)
	f.Close()
	if e != nil || string(data) != "zip payload" {
		t.Fatal(e, string(data))
	}
	if e = c.a.DB.QueryRow(context.Background(), "SELECT count(*) FROM download_journal").Scan(&count); e != nil || count != 2 {
		t.Fatal("aggregate journal", e, count)
	}
	for i := 0; i < 4; i++ {
		if e = c.a.Files.Cleanup(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	if e = c.a.DB.QueryRow(context.Background(), "SELECT count(*) FROM files WHERE state='deleted'").Scan(&count); e != nil || count != 2 {
		t.Fatal("cleanup", e, count)
	}
}
func TestIntegrationConcurrentAcquireAndRevoke(t *testing.T) {
	c, _ := testClient(t)
	id := c.draft()
	c.upload(id, "a.txt", "payload")
	r := c.recipient(c.publish(id))
	ref := r.refs()[0]["ref"].(string)
	var wg sync.WaitGroup
	codes := make(chan int, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := r.req("POST", "/api/v1/recipient/downloads", `{"fileRef":"`+ref+`"}`, nil)
			codes <- w.Code
		}()
	}
	wg.Wait()
	close(codes)
	wins := 0
	for code := range codes {
		if code == 201 {
			wins++
		} else if code != 409 {
			t.Fatal("unexpected", code)
		}
	}
	if wins != 1 {
		t.Fatal("multiple leases", wins)
	}
	w := c.req("POST", "/api/v1/transfers/"+id+"/revoke", "", nil)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = r.req("GET", "/api/v1/recipient/files", "", nil)
	if w.Code != 401 {
		t.Fatal("revoked session", w.Code)
	}
}

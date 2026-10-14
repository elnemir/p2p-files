package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIntegrationSetupAtomicRaceReplayAndFault(t *testing.T) {
	a, db := integrationApp(t)
	ctx := context.Background()
	token := strings.Repeat("s", 43)
	sum := sha256.Sum256([]byte(token))
	t.Setenv("FILEX_SETUP_TOKEN_HASH", hex.EncodeToString(sum[:]))
	t.Setenv("FILEX_SETUP_EXPIRES_AT", time.Now().Add(time.Hour).Format(time.RFC3339))
	h := a.Handler()
	request := func(token, origin, body string) int {
		r := httptest.NewRequest("POST", "/api/v1/setup", strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Setup-Token", token)
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	body := `{"email":"first@example.test","password":"a-safe-first-password-123"}`
	if code := request("wrong", "http://localhost:8090", body); code != 403 {
		t.Fatal(code)
	}
	if code := request(token, "https://evil.test", body); code != 403 {
		t.Fatal(code)
	}
	if code := request(token, "http://localhost:8090", `{"email":"x","password":"short"}`); code != 400 {
		t.Fatal(code)
	}
	// Fail after the user INSERT, proving completion and user/password audit roll back together.
	if _, e := db.Exec(ctx, `CREATE FUNCTION setup_fault() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$; CREATE TRIGGER setup_fault BEFORE INSERT ON password_history FOR EACH ROW EXECUTE FUNCTION setup_fault()`); e != nil {
		t.Fatal(e)
	}
	if code := request(token, "http://localhost:8090", body); code != 503 {
		t.Fatal(code)
	}
	var n int
	var complete bool
	if e := db.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&n); e != nil || n != 0 {
		t.Fatal("partial user", n, e)
	}
	if e := db.QueryRow(ctx, "SELECT completed FROM ops_setup").Scan(&complete); e != nil || complete {
		t.Fatal("partial latch", e)
	}
	if _, e := db.Exec(ctx, "DROP TRIGGER setup_fault ON password_history; UPDATE ops_license SET envelope='',pending_envelope=''"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for range 2 {
		wg.Go(func() { results <- request(token, "http://localhost:8090", body) })
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal(counts)
	}
	if code := request(token, "http://localhost:8090", body); code != 409 {
		t.Fatal("replay", code)
	}
	b, e := New(db, a.Keys.KEK)
	if e != nil {
		t.Fatal(e)
	}
	defer b.Close()
	if w := call(b.Handler(), "GET", "/api/v1/setup", "", nil, "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "complete") {
		t.Fatal("restart reopened", w.Code, w.Body.String())
	}
	if _, e := db.Exec(ctx, "UPDATE ops_setup SET completed=false"); e == nil {
		t.Fatal("latch reopened")
	}
	if w := call(h, "POST", "/api/v1/auth/login", body, nil, "", "http://localhost:8090"); w.Code != 200 {
		t.Fatal("first admin cannot login", w.Code)
	}
}

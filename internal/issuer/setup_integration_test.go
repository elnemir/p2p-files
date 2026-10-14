package issuer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"filex.local/filex/internal/setup"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIntegrationIssuerSetupEncryptedRaceRestart(t *testing.T) {
	db := fixture(t)
	ctx := context.Background()
	key := bytes.Repeat([]byte{7}, 32)
	var base string
	idp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"issuer": base, "authorization_endpoint": base + "/authorize", "token_endpoint": base + "/token", "jwks_uri": base + "/keys"})
	}))
	defer idp.Close()
	base = idp.URL
	token := strings.Repeat("i", 43)
	sum := sha256.Sum256([]byte(token))
	access := setup.Access{Hash: hex.EncodeToString(sum[:]), Origin: "https://issuer.test", Expires: time.Now().Add(time.Hour)}
	runtime := &SetupRuntime{DB: db, Key: key, Access: access}
	raw, _ := json.Marshal(SetupInput{base, "client", "super-secret-client-value", "mfa", "operator-sub"})
	request := func(token string) int {
		r := httptest.NewRequest("POST", "https://issuer.test/issuer/api/setup", bytes.NewReader(raw))
		r = r.WithContext(context.WithValue(r.Context(), oauth2.HTTPClient, idp.Client()))
		r.Header.Set("Origin", access.Origin)
		r.Header.Set("X-Setup-Token", token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		runtime.ServeHTTP(w, r)
		return w.Code
	}
	if code := request("wrong"); code != 403 {
		t.Fatal(code)
	}
	if _, e := db.Exec(ctx, `CREATE FUNCTION setup_fault() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected'; END $$; CREATE TRIGGER setup_fault BEFORE INSERT ON issuer_setup FOR EACH ROW EXECUTE FUNCTION setup_fault()`); e != nil {
		t.Fatal(e)
	}
	if code := request(token); code != 503 {
		t.Fatal("fault", code)
	}
	var n int
	if e := db.QueryRow(ctx, "SELECT count(*) FROM issuer_setup").Scan(&n); e != nil || n != 0 {
		t.Fatal("partial config", n, e)
	}
	if _, e := db.Exec(ctx, "DROP TRIGGER setup_fault ON issuer_setup"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for range 2 {
		wg.Go(func() { results <- request(token) })
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
	var box []byte
	if e := db.QueryRow(ctx, "SELECT config FROM issuer_setup").Scan(&box); e != nil || bytes.Contains(box, []byte("super-secret")) {
		t.Fatal("cleartext secret", e)
	}
	if _, e := loadSetup(ctx, db, bytes.Repeat([]byte{8}, 32)); e == nil {
		t.Fatal("wrong key accepted")
	}
	c, e := loadSetup(ctx, db, key)
	if e != nil || c.Operators["operator-sub"] != "security_admin" {
		t.Fatal("readback", e)
	}
	if code := request(token); code != 409 {
		t.Fatal("replay", code)
	}
	if _, e := db.Exec(ctx, "DELETE FROM issuer_setup"); e == nil {
		t.Fatal("config mutable")
	}
	restarted := &SetupRuntime{DB: db, Key: key, Access: setup.Access{}}
	r := httptest.NewRequest("GET", "https://issuer.test/issuer/auth/login", nil)
	r = r.WithContext(context.WithValue(r.Context(), oauth2.HTTPClient, idp.Client()))
	w := httptest.NewRecorder()
	restarted.ServeHTTP(w, r)
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "acr_values=mfa") {
		t.Fatal("restart lost config", w.Code, w.Body.String())
	}
}

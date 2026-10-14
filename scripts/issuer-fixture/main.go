// Local browser qualification fixture. Never included in customer/vendor images.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/httpapi"
	"filex.local/filex/internal/issuer"
	"filex.local/filex/internal/setup"
	"fmt"
	"github.com/go-jose/go-jose/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	admin, e := pgx.Connect(ctx, os.Getenv("FILEX_TEST_DATABASE_URL"))
	if e != nil {
		return e
	}
	defer admin.Close(context.Background())
	schema := "issuer_browser_" + strings.ReplaceAll(domain.NewID(), "-", "")
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		return e
	}
	defer func() {
		if _, e := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			fmt.Fprintln(os.Stderr, "fixture cleanup failed")
		}
	}()
	cfg, e := pgxpool.ParseConfig(os.Getenv("FILEX_TEST_DATABASE_URL"))
	if e != nil {
		return e
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return e
	}
	defer db.Close()
	if e = issuer.Migrate(ctx, db); e != nil {
		return e
	}
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		return e
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "fixture"))
	if e != nil {
		return e
	}
	var idpURL string
	var mu sync.Mutex
	codes := map[string]url.Values{}
	idp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": idpURL, "authorization_endpoint": idpURL + "/authorize", "token_endpoint": idpURL + "/token", "jwks_uri": idpURL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "fixture", Algorithm: "RS256", Use: "sig"}}})
		case "/authorize":
			code := domain.NewID()
			mu.Lock()
			codes[code] = r.URL.Query()
			mu.Unlock()
			http.Redirect(w, r, r.URL.Query().Get("redirect_uri")+"?code="+code+"&state="+url.QueryEscape(r.URL.Query().Get("state")), http.StatusFound)
		case "/token":
			if e := r.ParseForm(); e != nil {
				http.Error(w, "invalid", 400)
				return
			}
			mu.Lock()
			v, ok := codes[r.Form.Get("code")]
			delete(codes, r.Form.Get("code"))
			mu.Unlock()
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != v.Get("code_challenge") {
				http.Error(w, "invalid", http.StatusUnauthorized)
				return
			}
			raw, e := json.Marshal(map[string]any{"iss": idpURL, "sub": "browser-operator", "aud": "fixture-client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "auth_time": time.Now().Unix(), "nonce": v.Get("nonce"), "acr": "fixture-mfa"})
			if e != nil {
				http.Error(w, "error", 500)
				return
			}
			signed, e := signer.Sign(raw)
			if e != nil {
				http.Error(w, "error", 500)
				return
			}
			token, e := signed.CompactSerialize()
			if e != nil {
				http.Error(w, "error", 500)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "fixture-unused", "token_type": "Bearer", "id_token": token})
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	idpURL = idp.URL
	target, e := url.Parse("http://127.0.0.1:3002")
	if e != nil {
		return e
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	mux := http.NewServeMux()
	public := httptest.NewTLSServer(mux)
	defer public.Close()
	providerContext := context.WithValue(ctx, oauth2.HTTPClient, idp.Client())
	api, e := issuer.New(providerContext, db, issuer.Config{Origin: public.URL, Issuer: idpURL, ClientID: "fixture-client", ClientSecret: "fixture-secret", RequiredACR: "fixture-mfa", Operators: map[string]string{"browser-operator": "issuer"}})
	if e != nil {
		return e
	}
	var handler http.Handler = api.Handler()
	if os.Getenv("FILEX_SETUP_BROWSER") == "1" {
		token := strings.Repeat("fixture-only-", 4)
		sum := sha256.Sum256([]byte(token))
		handler = &issuer.SetupRuntime{DB: db, Key: bytes.Repeat([]byte{3}, 32), Access: setup.Access{Hash: hex.EncodeToString(sum[:]), Origin: public.URL, Expires: time.Now().Add(time.Hour)}}
		if e = database.Migrate(ctx, db); e != nil {
			return e
		}
		product, e := httpapi.New(db, bytes.Repeat([]byte{4}, 32))
		if e != nil {
			return e
		}
		defer product.Close()
		if _, e = db.Exec(ctx, `UPDATE app_settings SET value=value || jsonb_build_object('publicOrigin',$1::text) WHERE active`, public.URL); e != nil {
			return e
		}
		if e = os.Setenv("FILEX_SETUP_TOKEN_HASH", hex.EncodeToString(sum[:])); e != nil {
			return e
		}
		if e = os.Setenv("FILEX_SETUP_EXPIRES_AT", time.Now().Add(time.Hour).Format(time.RFC3339)); e != nil {
			return e
		}
		productURL, e := url.Parse("http://127.0.0.1:3000")
		if e != nil {
			return e
		}
		mux.Handle("/api/", product.Handler())
		mux.Handle("/", httputil.NewSingleHostReverseProxy(productURL))
		fixtureJSON, e := json.Marshal(map[string]string{"token": token, "issuer": idpURL})
		if e != nil {
			return e
		}
		if e = os.WriteFile(os.Getenv("ISSUER_FIXTURE_URL_FILE")+".setup.json", fixtureJSON, 0600); e != nil {
			return e
		}
	}
	mux.Handle("/issuer/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/issuer/api/") || strings.HasPrefix(r.URL.Path, "/issuer/auth/") {
			handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), oauth2.HTTPClient, idp.Client())))
			return
		}
		proxy.ServeHTTP(w, r)
	}))
	mux.Handle("/issuer", proxy)
	_, vendor, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return e
	}
	if e = os.WriteFile(os.Getenv("ISSUER_FIXTURE_URL_FILE"), []byte(public.URL), 0600); e != nil {
		return e
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, e = issuer.SignNext(ctx, db, "browser-fixture-only", vendor); e != nil {
				return e
			}
		}
	}
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "issuer fixture failed", e)
		os.Exit(1)
	}
}

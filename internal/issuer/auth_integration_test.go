package issuer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIntegrationIssuerOIDCMFAAndCSRF(t *testing.T) {
	db := fixture(t)
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "fixture"))
	if e != nil {
		t.Fatal(e)
	}
	var base, nonce string
	scenario := "valid"
	idp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": base, "authorization_endpoint": base + "/authorize", "token_endpoint": base + "/token", "jwks_uri": base + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "fixture", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			if e := r.ParseForm(); e != nil {
				t.Error(e)
				return
			}
			if r.Form.Get("code_verifier") == "" || r.Form.Get("redirect_uri") != "https://issuer.test/issuer/auth/callback" {
				t.Error("exchange lost PKCE/redirect binding")
			}
			claims := map[string]any{"iss": base, "sub": "named-operator", "aud": "client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": nonce, "acr": "mfa", "auth_time": time.Now().Unix()}
			switch scenario {
			case "weak":
				claims["acr"] = "password"
			case "stale":
				claims["auth_time"] = time.Now().Add(-time.Hour).Unix()
			case "unknown":
				claims["sub"] = "unknown"
			case "nonce":
				claims["nonce"] = "wrong"
			}
			raw, e := json.Marshal(claims)
			if e != nil {
				t.Error(e)
				return
			}
			signed, e := signer.Sign(raw)
			if e != nil {
				t.Error(e)
				return
			}
			jwt, e := signed.CompactSerialize()
			if e != nil {
				t.Error(e)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "unused", "token_type": "Bearer", "id_token": jwt})
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	base = idp.URL
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, idp.Client())
	s, e := New(ctx, db, Config{Origin: "https://issuer.test", Issuer: base, ClientID: "client", ClientSecret: "secret", RequiredACR: "mfa", Operators: map[string]string{"named-operator": "issuer"}})
	if e != nil {
		t.Fatal(e)
	}
	call := func(method, path, body string, c *http.Cookie, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Origin", "https://issuer.test")
		r.Header.Set("X-CSRF-Token", csrf)
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	for _, name := range []string{"weak", "stale", "unknown", "nonce", "valid"} {
		t.Run(name, func(t *testing.T) {
			scenario = name
			login := call("GET", "/issuer/auth/login", "", nil, "")
			if login.Code != 302 {
				t.Fatal(login.Code)
			}
			u, e := url.Parse(login.Header().Get("Location"))
			if e != nil {
				t.Fatal(e)
			}
			nonce = u.Query().Get("nonce")
			if u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("acr_values") != "mfa" {
				t.Fatal("MFA/PKCE missing")
			}
			callback := "/issuer/auth/callback?code=fixture&state=" + url.QueryEscape(u.Query().Get("state"))
			w := call("GET", callback, "", login.Result().Cookies()[0], "")
			if name != "valid" {
				if w.Code != 401 && w.Code != 403 {
					t.Fatal("weak identity accepted", w.Code)
				}
				return
			}
			if w.Code != 303 {
				t.Fatal("valid identity rejected", w.Code)
			}
			var session *http.Cookie
			for _, c := range w.Result().Cookies() {
				if c.Name == "issuerSession" {
					session = c
				}
			}
			if session == nil || !session.Secure || !session.HttpOnly {
				t.Fatal("session cookie")
			}
			me := call("GET", "/issuer/api/session", "", session, "")
			var p principal
			if e = json.Unmarshal(me.Body.Bytes(), &p); e != nil || p.Role != "issuer" {
				t.Fatal(e, p)
			}
			if w = call("POST", "/issuer/api/issuances", "{}", session, ""); w.Code != 403 {
				t.Fatal("CSRF bypass", w.Code)
			}
			request := input()
			raw, e := json.Marshal(request)
			if e != nil {
				t.Fatal(e)
			}
			if w = call("POST", "/issuer/api/issuances", string(raw), session, p.CSRF); w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			if w = call("GET", callback, "", login.Result().Cookies()[0], ""); w.Code != 401 {
				t.Fatal("flow replay", w.Code)
			}
		})
	}
}

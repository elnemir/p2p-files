package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOIDCCodePKCENonceAndClaims(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "fixture"))
	if e != nil {
		t.Fatal(e)
	}
	f := Flow{Origin: "https://app.test", Purpose: "login", Nonce: "nonce", Verifier: oauth2.GenerateVerifier()}
	claims := map[string]any{"iss": "https://idp.test", "sub": "stable-subject", "aud": "client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": f.Nonce, "email": "person@example.test", "email_verified": true, "auth_time": time.Now().Unix()}
	tokenCalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": "https://idp.test", "authorization_endpoint": "https://idp.test/auth", "token_endpoint": "https://idp.test/token", "jwks_uri": "https://idp.test/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "fixture", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			tokenCalls++
			if e := r.ParseForm(); e != nil {
				t.Error(e)
			}
			if r.Form.Get("code_verifier") != f.Verifier || r.Form.Get("code") != "one-use-code" || r.Form.Get("redirect_uri") != "https://app.test/api/v1/auth/providers/provider/callback" {
				t.Error("exchange binding")
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
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "never-persisted", "token_type": "Bearer", "id_token": jwt})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := server.Client()
	tr := client.Transport.(*http.Transport)
	tr.TLSClientConfig.ServerName = "example.com"
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "https://"))
	}
	p := Provider{ID: "provider", Config: oidcConfig(), httpClient: client}
	target, e := p.OIDCStart(context.Background(), "state", f)
	if e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(target)
	if u.Query().Get("code_challenge") != oauth2.S256ChallengeFromVerifier(f.Verifier) || u.Query().Get("nonce") != f.Nonce || u.Query().Get("state") != "state" {
		t.Fatal("start correlation")
	}
	proof, e := p.OIDCVerify(context.Background(), "one-use-code", f)
	if e != nil || proof.Subject != "stable-subject" {
		t.Fatal(proof, e)
	}
	cases := []struct {
		claim string
		value any
	}{{"nonce", "wrong"}, {"iss", "https://evil.test"}, {"aud", "other"}, {"sub", ""}, {"email_verified", false}, {"exp", time.Now().Add(-time.Minute).Unix()}, {"azp", "other"}, {"auth_time", time.Now().Add(time.Hour).Unix()}}
	for _, tc := range cases {
		t.Run(tc.claim, func(t *testing.T) {
			old, exists := claims[tc.claim]
			claims[tc.claim] = tc.value
			defer func() {
				if exists {
					claims[tc.claim] = old
				} else {
					delete(claims, tc.claim)
				}
			}()
			if _, e := p.OIDCVerify(context.Background(), "one-use-code", f); e == nil {
				t.Fatal("bad claim accepted")
			}
		})
	}
	delete(claims, "auth_time")
	if _, e = p.OIDCVerify(context.Background(), "one-use-code", f); e != nil {
		t.Fatal("ordinary login without step-up", e)
	}
	f.Purpose = "reauth"
	if _, e = p.OIDCVerify(context.Background(), "one-use-code", f); e == nil {
		t.Fatal("reauth without auth_time")
	}
	f.Purpose = "login"
	p.Config.HostedDomain = "example.test"
	if _, e = p.OIDCVerify(context.Background(), "one-use-code", f); e == nil {
		t.Fatal("workspace without hd")
	}
	claims["hd"] = "example.test"
	if _, e = p.OIDCVerify(context.Background(), "one-use-code", f); e != nil {
		t.Fatal(e)
	}
	if tokenCalls < 10 {
		t.Fatal("protocol was not exercised")
	}
}

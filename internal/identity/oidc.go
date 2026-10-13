package identity

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"net/url"
	"time"
)

func (p Provider) oidc(ctx context.Context, origin string) (context.Context, *oidc.Provider, *oauth2.Config, error) {
	client, e := p.Config.client()
	if e != nil {
		return ctx, nil, nil, e
	}
	if p.httpClient != nil {
		client = p.httpClient
	}
	ctx = oidc.ClientContext(ctx, client)
	provider, e := oidc.NewProvider(ctx, p.Config.Issuer)
	if e != nil {
		return ctx, nil, nil, auth.ErrCredentials
	}
	ep := provider.Endpoint()
	if !p.Config.allowedEndpoint(ep.AuthURL) || !p.Config.allowedEndpoint(ep.TokenURL) {
		return ctx, nil, nil, auth.ErrCredentials
	}
	c := &oauth2.Config{ClientID: p.Config.ClientID, ClientSecret: p.secret, Endpoint: ep, RedirectURL: origin + "/api/v1/auth/providers/" + p.ID + "/callback", Scopes: []string{oidc.ScopeOpenID, "email"}}
	return ctx, provider, c, nil
}
func (p Provider) OIDCStart(ctx context.Context, state string, f Flow) (string, error) {
	_, _, c, e := p.oidc(ctx, f.Origin)
	if e != nil {
		return "", e
	}
	prompt := "login"
	if p.Config.Issuer == "https://accounts.google.com" {
		prompt = "select_account"
	}
	options := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(f.Verifier), oidc.Nonce(f.Nonce), oauth2.SetAuthURLParam("max_age", "0"), oauth2.SetAuthURLParam("prompt", prompt), oauth2.SetAuthURLParam("claims", `{"id_token":{"auth_time":{"essential":true}}}`)}
	if p.Config.HostedDomain != "" {
		options = append(options, oauth2.SetAuthURLParam("hd", p.Config.HostedDomain))
	}
	return c.AuthCodeURL(state, options...), nil
}
func (p Provider) OIDCVerify(ctx context.Context, code string, f Flow) (Proof, error) {
	if code == "" || len(code) > 8192 {
		return Proof{}, auth.ErrCredentials
	}
	ctx, provider, c, e := p.oidc(ctx, f.Origin)
	if e != nil {
		return Proof{}, e
	}
	token, e := c.Exchange(ctx, code, oauth2.VerifierOption(f.Verifier))
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || len(raw) > 65536 {
		return Proof{}, auth.ErrCredentials
	}
	id, e := provider.Verifier(&oidc.Config{ClientID: p.Config.ClientID, SupportedSigningAlgs: []string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512", "EdDSA"}}).Verify(ctx, raw)
	if e != nil || id.Nonce != f.Nonce || id.Subject == "" || id.IssuedAt.IsZero() || id.IssuedAt.After(time.Now().Add(time.Minute)) || !id.Expiry.After(time.Now()) {
		return Proof{}, auth.ErrCredentials
	}
	var claims struct {
		Verified bool        `json:"email_verified"`
		AuthTime json.Number `json:"auth_time"`
		AZP      string      `json:"azp"`
		HD       string      `json:"hd"`
	}
	var all map[string]json.RawMessage
	if id.Claims(&claims) != nil || id.Claims(&all) != nil {
		return Proof{}, auth.ErrCredentials
	}
	var email string
	if json.Unmarshal(all[p.Config.EmailAttribute], &email) != nil {
		return Proof{}, auth.ErrCredentials
	}
	if (!claims.Verified && !p.Config.TrustEmail) || (claims.AZP != "" && claims.AZP != p.Config.ClientID) || (len(id.Audience) > 1 && claims.AZP != p.Config.ClientID) || (p.Config.HostedDomain != "" && claims.HD != p.Config.HostedDomain) {
		return Proof{}, auth.ErrCredentials
	}
	stamp := int64(0)
	if claims.AuthTime != "" {
		stamp, e = claims.AuthTime.Int64()
		if e != nil || stamp > time.Now().Add(time.Minute).Unix() {
			return Proof{}, auth.ErrCredentials
		}
	}
	if f.Purpose == "reauth" && stamp < time.Now().Add(-5*time.Minute).Unix() {
		return Proof{}, auth.ErrCredentials
	}
	return p.Config.CheckProof(Proof{Subject: id.Subject, Email: email, AuthenticatedAt: stamp})
}
func CallbackQuery(q url.Values) (string, string, error) {
	if len(q["state"]) != 1 || len(q["code"]) != 1 || q.Get("error") != "" {
		return "", "", auth.ErrCredentials
	}
	return q.Get("state"), q.Get("code"), nil
}

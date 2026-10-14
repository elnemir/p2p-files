package issuer

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/auth"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

type Config struct {
	Origin       string
	Issuer       string
	ClientID     string
	ClientSecret string
	RequiredACR  string
	Operators    map[string]string
}
type Server struct {
	DB       *pgxpool.Pool
	config   Config
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func New(ctx context.Context, db *pgxpool.Pool, c Config) (*Server, error) {
	u, e := url.Parse(c.Origin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || c.ClientID == "" || c.RequiredACR == "" || len(c.Operators) == 0 {
		return nil, errors.New("invalid issuer configuration")
	}
	for sub, role := range c.Operators {
		if sub == "" || !slices.Contains([]string{"viewer", "issuer", "security_admin"}, role) {
			return nil, errors.New("invalid operator allowlist")
		}
	}
	issuer, e := url.Parse(c.Issuer)
	if e != nil || issuer.Scheme != "https" || issuer.Host == "" {
		return nil, errors.New("HTTPS identity provider required")
	}
	provider, e := oidc.NewProvider(ctx, c.Issuer)
	if e != nil {
		return nil, errors.New("identity provider unavailable")
	}
	return &Server{DB: db, config: c, oauth: oauth2.Config{ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: c.Origin + "/issuer/auth/callback", Scopes: []string{oidc.ScopeOpenID}}, verifier: provider.Verifier(&oidc.Config{ClientID: c.ClientID})}, nil
}
func cookie(w http.ResponseWriter, name, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/issuer", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	state, binding, nonce := auth.Token(), auth.Token(), auth.Token()
	verifier := oauth2.GenerateVerifier()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		reject(w, 503)
		return
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(724185042)"); e != nil {
		reject(w, 503)
		return
	}
	if _, e = tx.Exec(r.Context(), "DELETE FROM issuer_flows WHERE expires_at<clock_timestamp()"); e != nil {
		reject(w, 503)
		return
	}
	var count int
	if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM issuer_flows").Scan(&count); e != nil || count >= 1000 {
		reject(w, 429)
		return
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO issuer_flows(state_hash,binding_hash,verifier,nonce) VALUES($1,$2,$3,$4)", auth.Digest(state), auth.Digest(binding), verifier, nonce); e != nil {
		reject(w, 503)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		reject(w, 503)
		return
	}
	cookie(w, "issuerBinding", binding, 300)
	http.Redirect(w, r, s.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce), oauth2.SetAuthURLParam("max_age", "0"), oauth2.SetAuthURLParam("acr_values", s.config.RequiredACR)), http.StatusFound)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	binding, e := r.Cookie("issuerBinding")
	if e != nil || len(r.URL.Query()["state"]) != 1 || len(r.URL.Query()["code"]) != 1 {
		reject(w, 401)
		return
	}
	var verifier, nonce string
	e = s.DB.QueryRow(r.Context(), "DELETE FROM issuer_flows WHERE state_hash=$1 AND binding_hash=$2 AND expires_at>clock_timestamp() RETURNING verifier,nonce", auth.Digest(r.URL.Query().Get("state")), auth.Digest(binding.Value)).Scan(&verifier, &nonce)
	cookie(w, "issuerBinding", "", -1)
	if e != nil {
		reject(w, 401)
		return
	}
	token, e := s.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if e != nil {
		reject(w, 401)
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		reject(w, 401)
		return
	}
	id, e := s.verifier.Verify(r.Context(), raw)
	if e != nil || id.Nonce != nonce {
		reject(w, 401)
		return
	}
	var claims struct {
		ACR      string `json:"acr"`
		AuthTime int64  `json:"auth_time"`
	}
	if e = id.Claims(&claims); e != nil || claims.ACR != s.config.RequiredACR || claims.AuthTime < time.Now().Add(-5*time.Minute).Unix() || claims.AuthTime > time.Now().Add(time.Minute).Unix() || s.config.Operators[id.Subject] == "" {
		reject(w, 403)
		return
	}
	if _, e = s.DB.Exec(r.Context(), "DELETE FROM issuer_sessions WHERE expires_at<clock_timestamp()"); e != nil {
		reject(w, 503)
		return
	}
	session := auth.Token()
	if _, e = s.DB.Exec(r.Context(), "INSERT INTO issuer_sessions(token_hash,subject,authenticated_at) VALUES($1,$2,$3)", auth.Digest(session), id.Subject, time.Unix(claims.AuthTime, 0)); e != nil {
		reject(w, 503)
		return
	}
	if old, err := r.Cookie("issuerSession"); err == nil {
		if _, e = s.DB.Exec(r.Context(), "DELETE FROM issuer_sessions WHERE token_hash=$1", auth.Digest(old.Value)); e != nil {
			reject(w, 503)
			return
		}
	}
	cookie(w, "issuerSession", session, 900)
	http.Redirect(w, r, s.config.Origin+"/issuer", http.StatusSeeOther)
}

type principal struct {
	Subject         string    `json:"subject"`
	Role            string    `json:"role"`
	CSRF            string    `json:"csrfToken"`
	AuthenticatedAt time.Time `json:"authenticatedAt"`
}

func (s *Server) operator(r *http.Request, write bool) (principal, error) {
	var p principal
	c, e := r.Cookie("issuerSession")
	if e != nil {
		return p, e
	}
	e = s.DB.QueryRow(r.Context(), "SELECT subject,authenticated_at FROM issuer_sessions WHERE token_hash=$1 AND expires_at>clock_timestamp()", auth.Digest(c.Value)).Scan(&p.Subject, &p.AuthenticatedAt)
	if e != nil {
		return p, e
	}
	p.Role = s.config.Operators[p.Subject]
	if p.Role == "" {
		return p, errors.New("operator removed")
	}
	p.CSRF = auth.Digest(c.Value + "/csrf")
	if write && (r.Header.Get("Origin") != s.config.Origin || r.Header.Get("X-CSRF-Token") != p.CSRF || p.AuthenticatedAt.Before(time.Now().Add(-5*time.Minute))) {
		return p, errors.New("reauth or CSRF required")
	}
	return p, nil
}
func reject(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": strings.ReplaceAll(strings.ToLower(http.StatusText(status)), " ", "_")})
}

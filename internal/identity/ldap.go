package identity

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"filex.local/filex/internal/auth"
	"github.com/go-ldap/ldap/v3"
	"net"
	"net/url"
	"time"
)

func (p Provider) LDAPVerify(ctx context.Context, username, password string) (Proof, error) {
	if !normalizedUsername(username) || password == "" || len(password) > 1024 {
		return Proof{}, auth.ErrCredentials
	}
	u, e := url.Parse(p.Config.LDAPURL)
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	port := u.Port()
	if port == "" {
		port = "636"
		if u.Scheme == "ldap" {
			port = "389"
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	dial := p.Config.dial
	if p.dial != nil {
		dial = p.dial
	}
	conn, e := dial(ctx, "tcp", net.JoinHostPort(u.Hostname(), port))
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	defer conn.Close()
	if e = conn.SetDeadline(time.Now().Add(12 * time.Second)); e != nil {
		return Proof{}, auth.ErrCredentials
	}
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { rawConn.Close() })
	defer stop()
	cfg, e := p.Config.tls(u.Hostname())
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	if u.Scheme == "ldaps" {
		secure := tls.Client(conn, cfg)
		if e = secure.HandshakeContext(ctx); e != nil {
			return Proof{}, auth.ErrCredentials
		}
		conn = secure
	}
	l := ldap.NewConn(conn, u.Scheme == "ldaps")
	l.Start()
	defer l.Close()
	l.SetTimeout(5 * time.Second)
	if u.Scheme == "ldap" {
		if e = l.StartTLS(cfg); e != nil {
			return Proof{}, auth.ErrCredentials
		}
	}
	if e = l.Bind(p.Config.BindDN, p.secret); e != nil {
		return Proof{}, auth.ErrCredentials
	}
	search := ldap.NewSearchRequest(p.Config.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, 5, false, "("+p.Config.LoginAttribute+"="+ldap.EscapeFilter(username)+")", []string{p.Config.SubjectAttribute, p.Config.EmailAttribute}, nil)
	result, e := l.Search(search)
	if e != nil || len(result.Entries) != 1 || len(result.Referrals) != 0 {
		return Proof{}, auth.ErrCredentials
	}
	entry := result.Entries[0]
	subjects := entry.GetRawAttributeValues(p.Config.SubjectAttribute)
	emails := entry.GetAttributeValues(p.Config.EmailAttribute)
	if entry.DN == "" || len(subjects) != 1 || len(subjects[0]) == 0 || len(emails) != 1 {
		return Proof{}, auth.ErrCredentials
	}
	subject := string(subjects[0])
	if p.Config.SubjectAttribute == "objectGUID" {
		if len(subjects[0]) != 16 {
			return Proof{}, auth.ErrCredentials
		}
		subject = base64.RawURLEncoding.EncodeToString(subjects[0])
	}
	if e = l.Bind(entry.DN, password); e != nil {
		return Proof{}, auth.ErrCredentials
	}
	return p.Config.CheckProof(Proof{Subject: subject, Email: emails[0], AuthenticatedAt: time.Now().Unix()})
}

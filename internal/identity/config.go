// Package identity integrates external identity authorities with local FileX accounts.
package identity

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"filex.local/filex/internal/auth"
	"github.com/crewjam/saml"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Config struct {
	TrustEmail          bool     `json:"trustEmail"`
	HostedDomain        string   `json:"hostedDomain"`
	Name                string   `json:"name"`
	Protocol            string   `json:"protocol"`
	Enabled             bool     `json:"enabled"`
	AllowRegistration   bool     `json:"allowRegistration"`
	AllowedEmailDomains []string `json:"allowedEmailDomains"`
	PrivateCIDRs        []string `json:"privateCidrs"`
	CAPEM               string   `json:"caPem"`
	Issuer              string   `json:"issuer"`
	ClientID            string   `json:"clientId"`
	ExtraOrigins        []string `json:"extraOrigins"`
	MetadataXML         string   `json:"metadataXml"`
	EmailAttribute      string   `json:"emailAttribute"`
	LDAPURL             string   `json:"ldapUrl"`
	BaseDN              string   `json:"baseDn"`
	BindDN              string   `json:"bindDn"`
	LoginAttribute      string   `json:"loginAttribute"`
	SubjectAttribute    string   `json:"subjectAttribute"`
}
type Provider struct {
	httpClient *http.Client
	dial       func(context.Context, string, string) (net.Conn, error)
	ID         string `json:"id"`
	Revision   int64  `json:"revision"`
	Config     Config `json:"config"`
	HasSecret  bool   `json:"hasSecret"`
	secret     string
}
type Public struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Protocol          string `json:"protocol"`
	AllowRegistration bool   `json:"allowRegistration"`
}
type Input struct {
	ExpectedRevision int64   `json:"expectedRevision"`
	Config           Config  `json:"config"`
	Secret           *string `json:"secret,omitempty"`
}
type Proof struct {
	AuthenticatedAt int64  `json:"authenticatedAt"`
	Subject         string `json:"subject"`
	Email           string `json:"email"`
}

var attrPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{0,63}$`)

func cleanText(s string, max int) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= max && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.In(r, unicode.Cf) })
}
func httpsURL(s string) (*url.URL, error) {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || len(s) > 2048 {
		return nil, auth.ErrInvalid
	}
	return u, nil
}
func (c Config) Validate(secret string) error {
	if c.Name == "" || !cleanText(c.Name, 80) || len(secret) > 4096 || len(c.CAPEM) > 16384 || len(c.MetadataXML) > 45000 || len(c.PrivateCIDRs) > 16 || len(c.ExtraOrigins) > 8 || len(c.AllowedEmailDomains) > 100 {
		return auth.ErrInvalid
	}
	if c.CAPEM != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(c.CAPEM)) {
			return auth.ErrInvalid
		}
	}
	for _, v := range c.PrivateCIDRs {
		ip, n, e := net.ParseCIDR(v)
		if e != nil || !ip.IsPrivate() || !ip.Equal(n.IP) {
			return auth.ErrInvalid
		}
		ones, _ := n.Mask.Size()
		if (ip.To4() != nil && ones < 8) || (ip.To4() == nil && ones < 7) {
			return auth.ErrInvalid
		}
	}
	for _, d := range c.AllowedEmailDomains {
		e, err := auth.Email("test@" + d)
		if err != nil || e != "test@"+d || strings.ContainsAny(d, " /:@") {
			return auth.ErrInvalid
		}
	}
	switch c.Protocol {
	case "oidc":
		if c.EmailAttribute == "" || !attrPattern.MatchString(c.EmailAttribute) || !cleanText(c.HostedDomain, 253) {
			return auth.ErrInvalid
		}
		if c.TrustEmail && len(c.AllowedEmailDomains) == 0 {
			return auth.ErrInvalid
		}
		u, e := httpsURL(c.Issuer)
		if e != nil || u.RawQuery != "" || strings.HasSuffix(c.Issuer, "/") || c.ClientID == "" || !cleanText(c.ClientID, 512) {
			return auth.ErrInvalid
		}
		for _, v := range c.ExtraOrigins {
			u, e := httpsURL(v)
			if e != nil || u.Path != "" || u.RawQuery != "" {
				return auth.ErrInvalid
			}
		}
	case "saml":
		var m saml.EntityDescriptor
		if xml.Unmarshal([]byte(c.MetadataXML), &m) != nil || m.EntityID == "" || len(m.IDPSSODescriptors) != 1 || c.EmailAttribute == "" || !cleanText(c.EmailAttribute, 256) {
			return auth.ErrInvalid
		}
		found := false
		for _, ep := range m.IDPSSODescriptors[0].SingleSignOnServices {
			if ep.Binding == saml.HTTPRedirectBinding {
				if _, e := httpsURL(ep.Location); e != nil {
					return e
				}
				found = true
			}
		}
		signingKeys := 0
		for _, descriptor := range m.IDPSSODescriptors[0].KeyDescriptors {
			if descriptor.Use != "" && descriptor.Use != "signing" {
				continue
			}
			for _, encoded := range descriptor.KeyInfo.X509Data.X509Certificates {
				der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(encoded.Data), ""))
				if err != nil {
					return auth.ErrInvalid
				}
				cert, err := x509.ParseCertificate(der)
				if err != nil {
					return auth.ErrInvalid
				}
				switch key := cert.PublicKey.(type) {
				case *rsa.PublicKey:
					if key.N.BitLen() < 2048 {
						return auth.ErrInvalid
					}
				case *ecdsa.PublicKey:
					if key.Curve.Params().BitSize < 256 {
						return auth.ErrInvalid
					}
				default:
					return auth.ErrInvalid
				}
				signingKeys++
			}
		}
		if !found || signingKeys == 0 {
			return auth.ErrInvalid
		}
	case "ldap":
		u, e := url.Parse(c.LDAPURL)
		if e != nil || (u.Scheme != "ldaps" && u.Scheme != "ldap") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || len(c.LDAPURL) > 2048 {
			return auth.ErrInvalid
		}
		if c.BaseDN == "" || c.BindDN == "" || len(c.BaseDN) > 1024 || len(c.BindDN) > 1024 || !attrPattern.MatchString(c.LoginAttribute) || !attrPattern.MatchString(c.EmailAttribute) || (c.SubjectAttribute != "entryUUID" && c.SubjectAttribute != "objectGUID") || secret == "" {
			return auth.ErrInvalid
		}
	default:
		return auth.ErrInvalid
	}
	return nil
}
func (c Config) SameAuthority(other Config) bool {
	return c.Protocol == other.Protocol && c.Issuer == other.Issuer && c.ClientID == other.ClientID && c.LDAPURL == other.LDAPURL && c.BaseDN == other.BaseDN && c.SubjectAttribute == other.SubjectAttribute && c.EmailAttribute == other.EmailAttribute && samlEntity(c.MetadataXML) == samlEntity(other.MetadataXML)
}
func samlEntity(raw string) string {
	var v saml.EntityDescriptor
	if xml.Unmarshal([]byte(raw), &v) != nil {
		return ""
	}
	return v.EntityID
}
func (c Config) CheckProof(p Proof) (Proof, error) {
	if p.Subject == "" || !cleanText(p.Subject, 512) {
		return Proof{}, auth.ErrCredentials
	}
	email, e := auth.Email(p.Email)
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	if len(c.AllowedEmailDomains) > 0 {
		ok := false
		for _, d := range c.AllowedEmailDomains {
			if strings.HasSuffix(email, "@"+d) {
				ok = true
			}
		}
		if !ok {
			return Proof{}, auth.ErrCredentials
		}
	}
	return Proof{Subject: p.Subject, Email: email, AuthenticatedAt: p.AuthenticatedAt}, nil
}

// UnmarshalJSON keeps the typed public configuration contract strict, including nulls.
func (c *Config) UnmarshalJSON(raw []byte) error {
	type plain Config
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return auth.ErrInvalid
	}
	typ := reflect.TypeFor[plain]()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Tag.Get("json")
		v, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return auth.ErrInvalid
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value plain
	if decoder.Decode(&value) != nil {
		return auth.ErrInvalid
	}
	*c = Config(value)
	return nil
}

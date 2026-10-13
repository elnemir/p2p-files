package identity

import (
	"encoding/base64"
	"encoding/xml"
	"filex.local/filex/internal/auth"
	"github.com/crewjam/saml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (p Provider) saml(origin string) (*saml.ServiceProvider, error) {
	var metadata saml.EntityDescriptor
	if xml.Unmarshal([]byte(p.Config.MetadataXML), &metadata) != nil {
		return nil, auth.ErrInvalid
	}
	base := origin + "/api/v1/auth/providers/" + p.ID
	mu, e := url.Parse(base + "/metadata")
	if e != nil {
		return nil, e
	}
	acs, e := url.Parse(base + "/saml")
	if e != nil {
		return nil, e
	}
	force := true
	return &saml.ServiceProvider{EntityID: mu.String(), MetadataURL: *mu, AcsURL: *acs, IDPMetadata: &metadata, AuthnNameIDFormat: saml.PersistentNameIDFormat, ForceAuthn: &force, AllowIDPInitiated: false}, nil
}
func (p Provider) SAMLMetadata(origin string) ([]byte, error) {
	sp, e := p.saml(origin)
	if e != nil {
		return nil, e
	}
	return xml.Marshal(sp.Metadata())
}
func (p Provider) SAMLStart(state string, f *Flow) (string, error) {
	sp, e := p.saml(f.Origin)
	if e != nil {
		return "", e
	}
	request, e := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if e != nil {
		return "", auth.ErrCredentials
	}
	f.RequestID = request.ID
	target, e := request.Redirect(state, sp)
	if e != nil {
		return "", auth.ErrCredentials
	}
	return target.String(), nil
}
func (p Provider) SAMLVerify(r *http.Request, f Flow) (Proof, error) {
	if len(r.PostForm["SAMLResponse"]) != 1 || r.PostForm.Get("SAMLart") != "" {
		return Proof{}, auth.ErrCredentials
	}
	raw, e := base64.StdEncoding.DecodeString(r.PostForm.Get("SAMLResponse"))
	if e != nil || len(raw) > maxResponse {
		return Proof{}, auth.ErrCredentials
	}
	// Reject legacy hashes and directives before handing XML to the signature library.
	assertions := 0
	decoder := xml.NewDecoder(strings.NewReader(string(raw)))
	for {
		token, e := decoder.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return Proof{}, auth.ErrCredentials
		}
		switch t := token.(type) {
		case xml.Directive:
			return Proof{}, auth.ErrCredentials
		case xml.StartElement:
			if t.Name.Local == "Assertion" {
				assertions++
				if assertions > 1 {
					return Proof{}, auth.ErrCredentials
				}
			}
			if t.Name.Local == "SignatureMethod" || t.Name.Local == "DigestMethod" {
				for _, a := range t.Attr {
					if a.Name.Local == "Algorithm" && (!strings.Contains(a.Value, "sha256") && !strings.Contains(a.Value, "sha384") && !strings.Contains(a.Value, "sha512")) {
						return Proof{}, auth.ErrCredentials
					}
				}
			}
		}
	}
	var shape saml.Response
	if xml.Unmarshal(raw, &shape) != nil || shape.Assertion == nil || shape.EncryptedAssertion != nil || shape.Assertion.Subject == nil || shape.Assertion.Conditions == nil {
		return Proof{}, auth.ErrCredentials
	}
	for _, confirmation := range shape.Assertion.Subject.SubjectConfirmations {
		if confirmation.SubjectConfirmationData == nil {
			return Proof{}, auth.ErrCredentials
		}
	}
	sp, e := p.saml(f.Origin)
	if e != nil {
		return Proof{}, e
	}
	// The library validates Destination against req.URL; never trust incoming Host.
	clone := r.Clone(r.Context())
	u := sp.AcsURL
	clone.URL = &u
	clone.Form = r.PostForm
	assertion, e := sp.ParseResponse(clone, []string{f.RequestID})
	if e != nil {
		return Proof{}, auth.ErrCredentials
	}
	if assertion.Subject == nil || assertion.Subject.NameID == nil || assertion.Subject.NameID.Format != string(saml.PersistentNameIDFormat) || len(assertion.Subject.SubjectConfirmations) == 0 || assertion.Conditions == nil || len(assertion.Conditions.AudienceRestrictions) == 0 || len(assertion.AuthnStatements) == 0 {
		return Proof{}, auth.ErrCredentials
	}
	now := time.Now()
	if !assertion.Conditions.NotOnOrAfter.After(now) || assertion.Conditions.NotBefore.After(now.Add(time.Minute)) {
		return Proof{}, auth.ErrCredentials
	}
	for _, restriction := range assertion.Conditions.AudienceRestrictions {
		if restriction.Audience.Value != sp.EntityID {
			return Proof{}, auth.ErrCredentials
		}
	}
	for _, confirmation := range assertion.Subject.SubjectConfirmations {
		data := confirmation.SubjectConfirmationData
		if confirmation.Method != "urn:oasis:names:tc:SAML:2.0:cm:bearer" || data == nil || !data.NotOnOrAfter.After(now) || data.NotBefore.After(now.Add(time.Minute)) {
			return Proof{}, auth.ErrCredentials
		}
	}
	if assertion.IssueInstant.Before(now.Add(-5*time.Minute)) || assertion.IssueInstant.After(now.Add(time.Minute)) {
		return Proof{}, auth.ErrCredentials
	}
	for _, statement := range assertion.AuthnStatements {
		if statement.AuthnInstant.Before(now.Add(-5*time.Minute)) || statement.AuthnInstant.After(now.Add(time.Minute)) {
			return Proof{}, auth.ErrCredentials
		}
	}
	emails := []string{}
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			if attribute.Name == p.Config.EmailAttribute {
				for _, v := range attribute.Values {
					emails = append(emails, v.Value)
				}
			}
		}
	}
	if len(emails) != 1 {
		return Proof{}, auth.ErrCredentials
	}
	return p.Config.CheckProof(Proof{Subject: assertion.Subject.NameID.Value, Email: emails[0], AuthenticatedAt: assertion.AuthnStatements[0].AuthnInstant.Unix()})
}

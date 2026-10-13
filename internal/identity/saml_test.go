package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"math/big"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSAMLSignatureAndRequestBinding(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	certificate, e := x509.CreateCertificate(rand.Reader, &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}, &x509.Certificate{SerialNumber: big.NewInt(1)}, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	metadata := fmt.Sprintf(`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.test/saml"><IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><KeyDescriptor use="signing"><KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><X509Data><X509Certificate>%s</X509Certificate></X509Data></KeyInfo></KeyDescriptor><SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.test/sso"/></IDPSSODescriptor></EntityDescriptor>`, base64.StdEncoding.EncodeToString(certificate))
	p := Provider{ID: "provider", Config: Config{Name: "SAML", Protocol: "saml", MetadataXML: metadata, EmailAttribute: "mail"}}
	if e = p.Config.Validate(""); e != nil {
		t.Fatal(e)
	}
	badMetadata := p.Config
	badMetadata.MetadataXML = strings.Replace(metadata, base64.StdEncoding.EncodeToString(certificate), "not-a-certificate", 1)
	if badMetadata.Validate("") == nil {
		t.Fatal("invalid signing certificate accepted")
	}
	badMetadata.MetadataXML = strings.Replace(metadata, `use="signing"`, `use="encryption"`, 1)
	if badMetadata.Validate("") == nil {
		t.Fatal("no signing certificate accepted")
	}
	f := Flow{Origin: "https://app.test", Purpose: "login"}
	target, e := p.SAMLStart("state", &f)
	if e != nil || f.RequestID == "" || !strings.HasPrefix(target, "https://idp.test/sso?") {
		t.Fatal(target, e)
	}
	acs := "https://app.test/api/v1/auth/providers/provider/saml"
	audience := "https://app.test/api/v1/auth/providers/provider/metadata"
	assertion := saml.Assertion{ID: "assertion-fixture", Version: "2.0", IssueInstant: now, Issuer: saml.Issuer{Value: "https://idp.test/saml"}, Subject: &saml.Subject{NameID: &saml.NameID{Format: string(saml.PersistentNameIDFormat), Value: "persistent-subject"}, SubjectConfirmations: []saml.SubjectConfirmation{{Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer", SubjectConfirmationData: &saml.SubjectConfirmationData{InResponseTo: f.RequestID, Recipient: acs, NotOnOrAfter: now.Add(time.Minute)}}}}, Conditions: &saml.Conditions{NotBefore: now.Add(-time.Minute), NotOnOrAfter: now.Add(time.Minute), AudienceRestrictions: []saml.AudienceRestriction{{Audience: saml.Audience{Value: audience}}}}, AuthnStatements: []saml.AuthnStatement{{AuthnInstant: now}}, AttributeStatements: []saml.AttributeStatement{{Attributes: []saml.Attribute{{Name: "mail", Values: []saml.AttributeValue{{Value: "person@example.test"}}}}}}}
	sign := func(a saml.Assertion) string {
		t.Helper()
		sc, e := dsig.NewSigningContext(key, [][]byte{certificate})
		if e != nil {
			t.Fatal(e)
		}
		sc.IdAttribute = "ID"
		if e = sc.SetSignatureMethod(dsig.RSASHA256SignatureMethod); e != nil {
			t.Fatal(e)
		}
		sc.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
		signed, e := sc.SignEnveloped(a.Element())
		if e != nil {
			t.Fatal(e)
		}
		response := saml.Response{ID: "response-fixture", Version: "2.0", IssueInstant: now, Destination: acs, InResponseTo: f.RequestID, Issuer: &saml.Issuer{Value: "https://idp.test/saml"}, Status: saml.Status{StatusCode: saml.StatusCode{Value: saml.StatusSuccess}}}
		root := response.Element()
		root.AddChild(signed)
		doc := etree.NewDocument()
		doc.SetRoot(root)
		raw, e := doc.WriteToString()
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	verify := func(raw string, flow Flow) error {
		r := httptest.NewRequest("POST", acs, nil)
		r.PostForm = url.Values{"SAMLResponse": {base64.StdEncoding.EncodeToString([]byte(raw))}}
		proof, e := p.SAMLVerify(r, flow)
		if e == nil && proof.Subject != "persistent-subject" {
			t.Fatal(proof)
		}
		return e
	}
	good := sign(assertion)
	if e = verify(good, f); e != nil {
		t.Fatal("valid SAML", e)
	}
	if e = verify(strings.Replace(good, "person@example.test", "attacker@example.test", 1), f); e == nil {
		t.Fatal("tampered assertion")
	}
	doc := etree.NewDocument()
	if e = doc.ReadFromString(good); e != nil {
		t.Fatal(e)
	}
	for _, child := range doc.Root().ChildElements() {
		if child.Tag == "Assertion" {
			doc.Root().AddChild(child.Copy())
			break
		}
	}
	duplicate, e := doc.WriteToString()
	if e != nil {
		t.Fatal(e)
	}
	if verify(duplicate, f) == nil {
		t.Fatal("multiple assertions accepted")
	}
	wrong := f
	wrong.RequestID = "other"
	if e = verify(good, wrong); e == nil {
		t.Fatal("request mixup")
	}
	assertion.Conditions.AudienceRestrictions[0].Audience.Value = "https://other.test"
	if e = verify(sign(assertion), f); e == nil {
		t.Fatal("audience mismatch")
	}
	assertion.Conditions.AudienceRestrictions[0].Audience.Value = audience
	assertion.Subject.NameID.Format = string(saml.TransientNameIDFormat)
	if e = verify(sign(assertion), f); e == nil {
		t.Fatal("transient subject")
	}
	for _, raw := range []string{`<Response/>`, `<!DOCTYPE x><Response/>`, strings.Replace(good, "rsa-sha256", "rsa-sha1", 1)} {
		if e = verify(raw, f); e == nil {
			t.Fatal("malformed or weak signature")
		}
	}
}

// Command identity-fixture is a disposable, offline SAML signer for browser tests.
// It is not included in runtime images and never reads production credentials.
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"math/big"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "identity fixture failed")
		os.Exit(1)
	}
}
func run() error {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		return e
	}
	now := time.Now().UTC()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		return e
	}
	metadata := fmt.Sprintf(`<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://identity-fixture.example.test"><IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><KeyDescriptor use="signing"><KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><X509Data><X509Certificate>%s</X509Certificate></X509Data></KeyInfo></KeyDescriptor><SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://identity-fixture.example.test/login"/></IDPSSODescriptor></EntityDescriptor>`, base64.StdEncoding.EncodeToString(der))
	out := json.NewEncoder(os.Stdout)
	if e = out.Encode(map[string]string{"metadata": metadata}); e != nil {
		return e
	}
	scan := bufio.NewScanner(os.Stdin)
	scan.Buffer(make([]byte, 4096), 65536)
	for scan.Scan() {
		var in struct{ RequestID, ACS, EntityID, Email, Subject string }
		if e = json.Unmarshal(scan.Bytes(), &in); e != nil {
			return e
		}
		now = time.Now().UTC()
		a := saml.Assertion{ID: fmt.Sprintf("fixture-%d", now.UnixNano()), Version: "2.0", IssueInstant: now, Issuer: saml.Issuer{Value: "https://identity-fixture.example.test"}, Subject: &saml.Subject{NameID: &saml.NameID{Format: string(saml.PersistentNameIDFormat), Value: in.Subject}, SubjectConfirmations: []saml.SubjectConfirmation{{Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer", SubjectConfirmationData: &saml.SubjectConfirmationData{InResponseTo: in.RequestID, Recipient: in.ACS, NotOnOrAfter: now.Add(time.Minute)}}}}, Conditions: &saml.Conditions{NotBefore: now.Add(-time.Minute), NotOnOrAfter: now.Add(time.Minute), AudienceRestrictions: []saml.AudienceRestriction{{Audience: saml.Audience{Value: in.EntityID}}}}, AuthnStatements: []saml.AuthnStatement{{AuthnInstant: now}}, AttributeStatements: []saml.AttributeStatement{{Attributes: []saml.Attribute{{Name: "mail", Values: []saml.AttributeValue{{Value: in.Email}}}}}}}
		signer, e := dsig.NewSigningContext(key, [][]byte{der})
		if e != nil {
			return e
		}
		signer.IdAttribute = "ID"
		if e = signer.SetSignatureMethod(dsig.RSASHA256SignatureMethod); e != nil {
			return e
		}
		signer.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
		signed, e := signer.SignEnveloped(a.Element())
		if e != nil {
			return e
		}
		response := saml.Response{ID: fmt.Sprintf("response-%d", now.UnixNano()), Version: "2.0", IssueInstant: now, InResponseTo: in.RequestID, Destination: in.ACS, Issuer: &saml.Issuer{Value: "https://identity-fixture.example.test"}, Status: saml.Status{StatusCode: saml.StatusCode{Value: saml.StatusSuccess}}}
		root := response.Element()
		root.AddChild(signed)
		doc := etree.NewDocument()
		doc.SetRoot(root)
		raw, e := doc.WriteToBytes()
		if e != nil {
			return e
		}
		if e = out.Encode(map[string]string{"response": base64.StdEncoding.EncodeToString(raw)}); e != nil {
			return e
		}
	}
	return scan.Err()
}

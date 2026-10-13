package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	ber "github.com/go-asn1-ber/asn1-ber"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLDAPTLSBindSearchAndFailure(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(2), DNSNames: []string{"directory.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	for _, tc := range []struct {
		name, scheme, password         string
		duplicate, missingEmail, badCA bool
	}{{name: "ldaps", scheme: "ldaps", password: "directory-password"}, {name: "starttls", scheme: "ldap", password: "directory-password"}, {name: "wrong_password", scheme: "ldaps", password: "wrong"}, {name: "empty_password", scheme: "ldaps"}, {name: "ambiguous", scheme: "ldaps", password: "directory-password", duplicate: true}, {name: "missing_email", scheme: "ldaps", password: "directory-password", missingEmail: true}, {name: "untrusted_tls", scheme: "ldaps", password: "directory-password", badCA: true}} {
		t.Run(tc.name, func(t *testing.T) {
			ln, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			var wg sync.WaitGroup
			var userBinds atomic.Int32
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					raw, e := ln.Accept()
					if e != nil {
						return
					}
					wg.Add(1)
					go func(conn net.Conn) {
						defer wg.Done()
						defer conn.Close()
						if conn.SetDeadline(time.Now().Add(5*time.Second)) != nil {
							return
						}
						secure := tc.scheme == "ldaps"
						if secure {
							conn = tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
						}
						for {
							packet, e := ber.ReadPacket(conn)
							if e != nil {
								return
							}
							if len(packet.Children) < 2 {
								return
							}
							id := packet.Children[0].Value.(int64)
							op := packet.Children[1]
							write := func(operation *ber.Packet) bool {
								message := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
								message.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, id, ""))
								message.AppendChild(operation)
								_, e := conn.Write(message.Bytes())
								return e == nil
							}
							result := func(tag ber.Tag, code int64) *ber.Packet {
								p := ber.Encode(ber.ClassApplication, ber.TypeConstructed, tag, nil, "")
								p.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, code, ""))
								p.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
								p.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "", ""))
								return p
							}
							switch op.Tag {
							case 23:
								if secure || !write(result(24, 0)) {
									return
								}
								conn = tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
								secure = true
							case 0:
								if !secure {
									t.Error("cleartext bind")
									return
								}
								if len(op.Children) != 3 {
									return
								}
								dn := op.Children[1].Value.(string)
								password := op.Children[2].Data.String()
								code := int64(49)
								if dn == "cn=service,dc=example,dc=test" && password == "service-secret" {
									code = 0
								}
								if dn == "uid=person,dc=example,dc=test" {
									userBinds.Add(1)
									if password == "directory-password" {
										code = 0
									}
								}
								if !write(result(1, code)) {
									return
								}
							case 3:
								filter := op.Children[6]
								if filter.Tag != 3 || len(filter.Children) != 2 || filter.Children[1].Data.String() != "person*)(uid=*)" {
									t.Error("LDAP filter was not escaped literal input")
									return
								}
								count := 1
								if tc.duplicate {
									count = 2
								}
								for i := 0; i < count; i++ {
									entry := ber.Encode(ber.ClassApplication, ber.TypeConstructed, 4, nil, "")
									entry.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, "uid=person,dc=example,dc=test", ""))
									attrs := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
									values := map[string]string{"entryUUID": "entry-stable", "mail": "person@example.test"}
									if tc.missingEmail {
										delete(values, "mail")
									}
									for name, value := range values {
										a := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "")
										a.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, name, ""))
										v := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSet, nil, "")
										v.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, value, ""))
										a.AppendChild(v)
										attrs.AppendChild(a)
									}
									entry.AppendChild(attrs)
									if !write(entry) {
										return
									}
								}
								if !write(result(5, 0)) {
									return
								}
							default:
								return
							}
						}
					}(raw)
				}
			}()
			defer func() { ln.Close(); wg.Wait() }()
			c := Config{Protocol: "ldap", LDAPURL: tc.scheme + "://directory.test", BaseDN: "dc=example,dc=test", BindDN: "cn=service,dc=example,dc=test", LoginAttribute: "uid", EmailAttribute: "mail", SubjectAttribute: "entryUUID", CAPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}
			if tc.badCA {
				c.CAPEM = ""
			}
			p := Provider{Config: c, secret: "service-secret", dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				if !strings.HasPrefix(address, "directory.test:") {
					t.Error(address)
				}
				return (&net.Dialer{}).DialContext(ctx, network, ln.Addr().String())
			}}
			proof, e := p.LDAPVerify(context.Background(), "person*)(uid=*)", tc.password)
			wantOK := tc.name == "ldaps" || tc.name == "starttls"
			if wantOK {
				if e != nil || proof.Subject != "entry-stable" || proof.AuthenticatedAt == 0 {
					t.Fatal(proof, e)
				}
			} else if e == nil {
				t.Fatal("invalid directory authentication accepted")
			}
			if (tc.duplicate || tc.missingEmail || tc.password == "" || tc.badCA) && userBinds.Load() != 0 {
				t.Fatal("user bind occurred before validation")
			}
		})
	}
}

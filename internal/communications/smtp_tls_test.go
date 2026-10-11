package communications

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSMTPTLSDeliveryAndCertificateValidation(t *testing.T) {
	fixture := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	cert := fixture.TLS.Certificates[0]
	root := x509.NewCertPool()
	root.AddCert(fixture.Certificate())
	fixture.Close()
	for _, mode := range []string{"starttls", "smtps"} {
		t.Run(mode, func(t *testing.T) {
			l, e := net.Listen("tcp4", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			defer l.Close()
			serverTLS := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
			received := make(chan string, 1)
			failed := make(chan error, 1)
			go func() {
				conn, e := l.Accept()
				if e != nil {
					failed <- e
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				encrypted := mode == "smtps"
				if encrypted {
					conn = tls.Server(conn, serverTLS)
				}
				io.WriteString(conn, "220 filex-test ESMTP\r\n")
				reader := textproto.NewReader(bufio.NewReader(conn))
				for {
					line, e := reader.ReadLine()
					if e != nil {
						failed <- e
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"):
						io.WriteString(conn, "250-filex-test\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
					case line == "STARTTLS":
						io.WriteString(conn, "220 Begin TLS\r\n")
						conn = tls.Server(conn, serverTLS)
						reader = textproto.NewReader(bufio.NewReader(conn))
						encrypted = true
					case strings.HasPrefix(line, "AUTH"):
						if !encrypted {
							failed <- io.ErrUnexpectedEOF
							return
						}
						io.WriteString(conn, "235 Authenticated\r\n")
					case strings.HasPrefix(line, "MAIL FROM:") || strings.HasPrefix(line, "RCPT TO:"):
						if !encrypted {
							failed <- io.ErrUnexpectedEOF
							return
						}
						io.WriteString(conn, "250 OK\r\n")
					case line == "DATA":
						io.WriteString(conn, "354 Send data\r\n")
						b, e := reader.ReadDotBytes()
						if e != nil {
							failed <- e
							return
						}
						received <- string(b)
						io.WriteString(conn, "250 Stored\r\n")
					case line == "QUIT":
						io.WriteString(conn, "221 Bye\r\n")
						failed <- nil
						return
					default:
						failed <- io.ErrUnexpectedEOF
						return
					}
				}
			}()
			port, _ := strconv.Atoi(strings.Split(l.Addr().String(), ":")[1])
			c := SMTPConfig{Host: "127.0.0.1", Port: port, TLS: mode, Username: "fixture", Password: "fixture", From: "a@example.test"}
			e = sendMailTLS(context.Background(), c, "stable-event-id", "b@example.test", "Тест FileX", "UTF-8 body", &tls.Config{RootCAs: root, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12})
			if e != nil {
				t.Fatal(e)
			}
			select {
			case message := <-received:
				if !strings.Contains(message, "Message-ID: <stable-event-id@filex.local>") || !strings.Contains(message, "UTF-8 body") {
					t.Fatal("invalid MIME message")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("mail missing")
			}
			if e = <-failed; e != nil {
				t.Fatal(e)
			}
		})
	}
	// Production clients must reject an untrusted server certificate.
	l, e := tls.Listen("tcp4", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, e := l.Accept()
		if e == nil {
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(3 * time.Second))
			conn.(*tls.Conn).Handshake()
		}
	}()
	if e = SendMail(context.Background(), SMTPConfig{Host: "127.0.0.1", Port: l.Addr().(*net.TCPAddr).Port, TLS: "smtps", From: "a@example.test"}, "id", "b@example.test", "subject", "body"); e == nil {
		t.Fatal("untrusted SMTP certificate accepted")
	}
	<-done
}

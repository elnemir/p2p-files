package storage

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestS3TLSWireConditionalWriteAndMissing(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.TLS == nil || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("unsigned or unencrypted request")
			w.WriteHeader(403)
			return
		}
		switch r.Method {
		case "HEAD":
			w.WriteHeader(200)
		case "PUT":
			if r.Header.Get("If-None-Match") != "*" {
				t.Error("overwrite guard absent")
			}
			if _, ok := objects[r.URL.Path]; ok {
				w.WriteHeader(412)
				return
			}
			b, e := io.ReadAll(r.Body)
			if e != nil {
				t.Error(e)
			}
			objects[r.URL.Path] = b
		case "GET":
			b, ok := objects[r.URL.Path]
			if !ok {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(404)
				io.WriteString(w, "<Error><Code>NoSuchKey</Code></Error>")
				return
			}
			w.Write(b)
		case "DELETE":
			delete(objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	client := s3.New(s3.Options{Region: "test-region", Credentials: credentials.NewStaticCredentialsProvider("test-access", "test-secret", ""), BaseEndpoint: aws.String(server.URL), UsePathStyle: true, HTTPClient: server.Client(), RetryMaxAttempts: 1})
	p := S3{client, "fixture"}
	ctx := context.Background()
	id := "00000000-0000-4000-8000-000000000001"
	want := bytes.Repeat([]byte{1, 2, 255}, 65536)
	if e := p.Check(ctx); e != nil {
		t.Fatal(e)
	}
	if e := p.Put(ctx, id, want); e != nil {
		t.Fatal(e)
	}
	if e := p.Put(ctx, id, []byte("overwrite")); e == nil {
		t.Fatal("object overwritten")
	}
	got, e := p.Get(ctx, id)
	if e != nil || !bytes.Equal(want, got) {
		t.Fatal("wire bytes differ", e, len(got))
	}
	if e = p.Delete(ctx, id); e != nil {
		t.Fatal(e)
	}
	if _, e = p.Get(ctx, id); !errors.Is(e, ErrMissing) {
		t.Fatal("missing mapping", e)
	}
	if e = p.Delete(ctx, id); e != nil {
		t.Fatal("delete not idempotent", e)
	}
	for _, bad := range []string{"../escape", "bad/object"} {
		if p.Put(ctx, bad, nil) == nil {
			t.Fatal("invalid key accepted")
		}
	}
	if _, e = NewS3(S3Config{Endpoint: "http://example.test", Region: "x", Bucket: "b", AccessKey: "a", SecretKey: "s"}); e == nil {
		t.Fatal("plaintext endpoint accepted")
	}
}

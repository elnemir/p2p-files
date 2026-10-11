package operations

import (
	"bytes"
	"context"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type failedPut struct {
	storage.Provider
	n int
}

func (p *failedPut) Put(ctx context.Context, k string, b []byte) error {
	p.n++
	if p.n == 2 {
		return errors.New("injected storage fault")
	}
	return p.Provider.Put(ctx, k, b)
}

func TestArchiveTamperTruncateWrongKeyAndIncomplete(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	p := storage.Local{Root: root}
	id, kid := domain.NewID(), domain.NewID()
	kek := bytes.Repeat([]byte{19}, 32)
	key := bytes.Repeat([]byte{20}, 32)
	wrapped, e := cryptobox.Seal(kek, "filex/keys/v1/"+kid, key)
	if e != nil {
		t.Fatal(e)
	}
	plain := bytes.Repeat([]byte("private database rows "), archiveChunk/20+1)
	if e = writeArchive(ctx, id, p, kid, key, wrapped, kek, func(w io.Writer) error { _, e := w.Write(plain); return e }); e != nil {
		t.Fatal(e)
	}
	read := func(k []byte) ([]byte, error) {
		r, e := readArchive(ctx, p, id, k)
		if e != nil {
			return nil, e
		}
		return io.ReadAll(r)
	}
	got, e := read(kek)
	if e != nil || !bytes.Equal(got, plain) {
		t.Fatal("roundtrip", e)
	}
	if _, e = read(bytes.Repeat([]byte{99}, 32)); e == nil {
		t.Fatal("wrong master key accepted")
	}
	chunk := filepath.Join(root, "backup-"+id+"-0000")
	original, e := os.ReadFile(chunk)
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"truncate", "tamper", "swap", "missing"} {
		t.Run(mode, func(t *testing.T) {
			bad := bytes.Clone(original)
			switch mode {
			case "truncate":
				bad = bad[:len(bad)-1]
			case "tamper":
				bad[30] ^= 1
			case "swap":
				var e error
				bad, e = os.ReadFile(filepath.Join(root, "backup-"+id+"-0001"))
				if e != nil {
					t.Fatal(e)
				}
			case "missing":
				if e := os.Remove(chunk); e != nil {
					t.Fatal(e)
				}
			}
			if mode != "missing" {
				if e := os.WriteFile(chunk, bad, 0600); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := read(kek); e == nil {
				t.Fatal("bad archive accepted")
			}
			if e := os.WriteFile(chunk, original, 0600); e != nil {
				t.Fatal(e)
			}
		})
	}
	failedID := domain.NewID()
	if e = writeArchive(ctx, failedID, &failedPut{Provider: p}, kid, key, wrapped, kek, func(w io.Writer) error { _, e := w.Write(plain); return e }); e == nil {
		t.Fatal("fault ignored")
	}
	if _, e = p.Get(ctx, "backup-"+failedID); !errors.Is(e, storage.ErrMissing) {
		t.Fatal("partial archive published", e)
	}
	if _, e = readArchive(ctx, p, failedID, kek); e == nil {
		t.Fatal("partial archive readable")
	}
}

func TestArchiveS3TLSRoundTripAndInventory(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.TLS == nil || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("unsecured S3 request")
			w.WriteHeader(403)
			return
		}
		switch r.Method {
		case "PUT":
			if r.Header.Get("If-None-Match") != "*" {
				t.Error("backup overwrite guard absent")
			}
			if _, ok := objects[r.URL.Path]; ok {
				w.WriteHeader(412)
				return
			}
			b, e := io.ReadAll(r.Body)
			if e != nil {
				t.Error(e)
				return
			}
			objects[r.URL.Path] = b
		case "GET":
			if r.URL.Query().Get("list-type") == "2" {
				w.Header().Set("Content-Type", "application/xml")
				if r.URL.Query().Get("continuation-token") == "" {
					io.WriteString(w, `<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>next-page</NextContinuationToken><Contents><Key>first</Key></Contents></ListBucketResult>`)
				} else {
					io.WriteString(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>last</Key></Contents></ListBucketResult>`)
				}
				return
			}
			b, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(b)
		default:
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	client := s3.New(s3.Options{Region: "fixture", Credentials: credentials.NewStaticCredentialsProvider("fixture-access", "fixture-secret", ""), BaseEndpoint: aws.String(server.URL), UsePathStyle: true, HTTPClient: server.Client(), RetryMaxAttempts: 1})
	p := storage.S3{Client: client, Bucket: "backups"}
	ctx := context.Background()
	key, kek := bytes.Repeat([]byte{31}, 32), bytes.Repeat([]byte{32}, 32)
	id, kid := domain.NewID(), domain.NewID()
	wrapped, e := cryptobox.Seal(kek, "filex/keys/v1/"+kid, key)
	if e != nil {
		t.Fatal(e)
	}
	plain := []byte("private database backup via S3")
	if e = writeArchive(ctx, id, p, kid, key, wrapped, kek, func(w io.Writer) error { _, e := w.Write(plain); return e }); e != nil {
		t.Fatal(e)
	}
	r, e := readArchive(ctx, p, id, kek)
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(r)
	if e != nil || !bytes.Equal(got, plain) {
		t.Fatal("S3 backup differs", e)
	}
	first, next, e := p.List(ctx, "", 1)
	if e != nil || len(first) != 1 || next != "next-page" {
		t.Fatal("inventory page", e, first, next)
	}
	last, next, e := p.List(ctx, next, 1)
	if e != nil || len(last) != 1 || next != "" {
		t.Fatal("inventory continuation", e, last, next)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, box := range objects {
		if bytes.Contains(box, plain) {
			t.Fatal("S3 plaintext leak")
		}
	}
}

package verify

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"filex.local/filex/internal/bundle"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustedSignatureAndTamperedZIP(t *testing.T) {
	seed := bytes.Repeat([]byte{7}, 32)
	plain := []byte("payload bytes")
	sum := sha256.Sum256(plain)
	ent := bundle.Entry{Path: "каталог/a.txt", Size: int64(len(plain)), CRC32: crc32.ChecksumIEEE(plain), SHA256: hex.EncodeToString(sum[:]), FileID: "fixture"}
	z, e := bundle.New([]bundle.Entry{ent}, seed, "key")
	if e != nil {
		t.Fatal(e)
	}
	var b bytes.Buffer
	if e = z.WriteRange(0, z.Size(), &b, func(_ bundle.Entry, from, to int64, w io.Writer) error { _, e := w.Write(plain[from:to]); return e }); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "verified.zip")
	if e = os.WriteFile(path, b.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	key := ed25519.NewKeyFromSeed(seed)
	der, _ := x509.MarshalPKIXPublicKey(key.Public())
	trusted := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if n, e := ZIP(path, trusted); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	wrong := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, 32))
	der, _ = x509.MarshalPKIXPublicKey(wrong.Public())
	if _, e = ZIP(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})); e == nil {
		t.Fatal("embedded key incorrectly trusted")
	}
	zr, e := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if e != nil {
		t.Fatal(e)
	}
	var bad bytes.Buffer
	zw := zip.NewWriter(&bad)
	for _, f := range zr.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if f.Name == ent.Path {
			data = []byte("modified payload")
		}
		w, e := zw.Create(f.Name)
		if e != nil {
			t.Fatal(e)
		}
		w.Write(data)
	}
	zw.Close()
	os.WriteFile(path, bad.Bytes(), 0600)
	if _, e = ZIP(path, trusted); e == nil {
		t.Fatal("modified payload accepted")
	}
	if n, e := Check(path, trusted, "", false); e != nil || n != 1 {
		t.Fatal("manifest-only check must not imply payload verification", n, e)
	}
	if _, e := Check(path, trusted, strings.Repeat("0", 64), false); e == nil {
		t.Fatal("incorrect independent fingerprint accepted")
	}
}
func TestRFC8032Ed25519Vector(t *testing.T) {
	seed, _ := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	pub, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	sig, _ := hex.DecodeString("e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
	key := ed25519.NewKeyFromSeed(seed)
	if !bytes.Equal(key.Public().(ed25519.PublicKey), pub) || !bytes.Equal(ed25519.Sign(key, nil), sig) || !ed25519.Verify(ed25519.PublicKey(pub), nil, sig) {
		t.Fatal("RFC8032 vector mismatch")
	}
}

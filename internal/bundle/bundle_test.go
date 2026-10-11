package bundle

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hash/crc32"
	"io"
	"testing"
)

func TestDeterministicZIP64RangesAndSignature(t *testing.T) {
	data := []byte("hello\x00world")
	sum := sha256.Sum256(data)
	seed := bytes.Repeat([]byte{7}, 32)
	b, e := New([]Entry{{Path: "папка/a.txt", Size: int64(len(data)), CRC32: crc32.ChecksumIEEE(data), SHA256: hex.EncodeToString(sum[:]), FileID: "f"}}, seed, "key")
	if e != nil {
		t.Fatal(e)
	}
	read := func(_ Entry, s, e int64, w io.Writer) error { _, err := w.Write(data[s:e]); return err }
	var full bytes.Buffer
	if e = b.WriteRange(0, b.Size(), &full, read); e != nil {
		t.Fatal(e)
	}
	if int64(full.Len()) != b.Size() {
		t.Fatal(full.Len(), b.Size())
	}
	z, e := zip.NewReader(bytes.NewReader(full.Bytes()), b.Size())
	if e != nil {
		t.Fatal(e)
	}
	found := map[string][]byte{}
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		v, e := io.ReadAll(r)
		if e != nil {
			t.Fatal(e)
		}
		r.Close()
		found[f.Name] = v
	}
	if !bytes.Equal(found["папка/a.txt"], data) {
		t.Fatal("payload changed")
	}
	sig, e := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(found["SHA256SUMS.sig"])))
	if e != nil || !ed25519.Verify(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey), found["SHA256SUMS.txt"], sig) {
		t.Fatal("signature")
	}
	for start := int64(0); start < b.Size(); start += 17 {
		end := min(start+71, b.Size())
		var part bytes.Buffer
		if e = b.WriteRange(start, end, &part, read); e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(part.Bytes(), full.Bytes()[start:end]) {
			t.Fatal("range mismatch", start)
		}
	}
}

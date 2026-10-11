package verify

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"filex.local/filex/internal/domain"
	"fmt"
	"io"
	"strings"
)

// ZIP requires a trusted key supplied out of band; the embedded public key is not a trust anchor.
func ZIP(path string, trustedPEM []byte) (int, error) {
	return Check(path, trustedPEM, "", true)
}

// Check always verifies signed metadata. Reading payload bytes is an explicit option.
func Check(path string, trustedPEM []byte, expectedFingerprint string, verifyFiles bool) (int, error) {
	block, rest := pem.Decode(trustedPEM)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return 0, errors.New("invalid trusted PEM")
	}
	v, e := x509.ParsePKIXPublicKey(block.Bytes)
	if e != nil {
		return 0, e
	}
	key, ok := v.(ed25519.PublicKey)
	if !ok {
		return 0, errors.New("Ed25519 public key required")
	}
	fingerprint := sha256.Sum256(block.Bytes)
	if expectedFingerprint != "" && expectedFingerprint != hex.EncodeToString(fingerprint[:]) {
		return 0, errors.New("trusted public key fingerprint mismatch")
	}
	z, e := zip.OpenReader(path)
	if e != nil {
		return 0, e
	}
	defer z.Close()
	if len(z.File) > 100003 {
		return 0, errors.New("too many entries")
	}
	entries := map[string]*zip.File{}
	for _, f := range z.File {
		if entries[f.Name] != nil {
			return 0, errors.New("duplicate entry")
		}
		entries[f.Name] = f
	}
	small := func(name string, max int64) ([]byte, error) {
		f := entries[name]
		if f == nil || f.UncompressedSize64 > uint64(max) {
			return nil, errors.New("missing or oversized checksum metadata")
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		defer r.Close()
		return io.ReadAll(io.LimitReader(r, max+1))
	}
	sums, e := small("SHA256SUMS.txt", 64<<20)
	if e != nil {
		return 0, e
	}
	encoded, e := small("SHA256SUMS.sig", 256)
	if e != nil {
		return 0, e
	}
	sig, e := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(encoded)))
	if e != nil || !ed25519.Verify(key, sums, sig) {
		return 0, errors.New("untrusted checksum signature")
	}
	seen := map[string]bool{}
	previous := ""
	if len(sums) == 0 || sums[len(sums)-1] != '\n' {
		return 0, errors.New("checksum manifest must end with LF")
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(sums), "\n"), "\n") {
		if len(line) < 67 || line[64:66] != "  " || !domain.ValidChecksum(line[:64]) {
			return 0, errors.New("invalid checksum record")
		}
		name := line[66:]
		pathKey, e := domain.PathKey(name, 4096)
		if e != nil || seen[pathKey] || (previous != "" && name <= previous) {
			return 0, errors.New("invalid checksum path")
		}
		previous = name
		seen[pathKey] = true
		f := entries[name]
		if f == nil || f.UncompressedSize64 > 1<<40 {
			return 0, errors.New("payload missing or too large")
		}
		if !verifyFiles {
			continue
		}
		r, e := f.Open()
		if e != nil {
			return 0, e
		}
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(r, int64(f.UncompressedSize64)+1))
		r.Close()
		if e != nil || uint64(n) != f.UncompressedSize64 || hex.EncodeToString(h.Sum(nil)) != line[:64] {
			return 0, fmt.Errorf("payload verification failed: %q", name)
		}
	}
	if len(entries) != len(seen)+3 {
		return 0, errors.New("unlisted payload")
	}
	embedded, e := small("SHA256SUMS.pub", 4096)
	if e != nil {
		return 0, e
	}
	pub, rest := pem.Decode(embedded)
	if pub == nil || len(strings.TrimSpace(string(rest))) != 0 || sha256.Sum256(pub.Bytes) != fingerprint {
		return 0, errors.New("embedded key does not match trusted key")
	}
	return len(seen), nil
}

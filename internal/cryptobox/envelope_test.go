package cryptobox

import (
	"bytes"
	"testing"
)

func TestAuthenticatedContextAndTamper(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	aad := "filex/test/v1/object_1"
	plain := []byte("sensitive data")
	box, err := Seal(key, aad, plain)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(key, aad, box)
	if err != nil || !bytes.Equal(plain, opened) {
		t.Fatal("roundtrip failed")
	}
	other, err := Seal(key, aad, plain)
	if err != nil || bytes.Equal(box, other) {
		t.Fatal("nonce reused")
	}
	for i := range box {
		bad := bytes.Clone(box)
		bad[i] ^= 1
		if _, err := Open(key, aad, bad); err == nil {
			t.Fatalf("tampering at byte %d accepted", i)
		}
	}
	for i := 0; i < len(box); i++ {
		if _, err := Open(key, aad, box[:i]); err == nil {
			t.Fatalf("truncation at %d accepted", i)
		}
	}
	if _, err := Open(key, "filex/test/v1/object_2", box); err == nil {
		t.Fatal("context swap accepted")
	}
	if _, err := Open(bytes.Repeat([]byte{2}, 32), aad, box); err == nil {
		t.Fatal("wrong key accepted")
	}
	if _, err := Seal(key, "unsafe context", plain); err == nil {
		t.Fatal("invalid context accepted")
	}
	if _, err := Seal(key[:31], aad, plain); err == nil {
		t.Fatal("wrong key size accepted")
	}
}

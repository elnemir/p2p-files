package cryptobox

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestPayloadBinaryVectorAndBindings(t *testing.T) {
	raw, e := os.ReadFile("testdata/payload-v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Root, FileKey, Salt, ObjectID, FileID, KeyID, Plain, Box string
		Size, Offset, Index                                      uint64
	}
	if e = json.Unmarshal(raw, &v); e != nil {
		t.Fatal(e)
	}
	dec := func(s string) []byte {
		b, e := hex.DecodeString(s)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	plain, box := dec(v.Plain), dec(v.Box)
	key, e := hkdf.Key(sha256.New, dec(v.Root), dec(v.Salt), "filex/payload/object/v1/"+v.FileID, 32)
	if e != nil || !bytes.Equal(key, dec(v.FileKey)) {
		t.Fatal("HKDF vector", e)
	}
	h := PayloadHeader{ObjectID: v.ObjectID, FileID: v.FileID, KeyID: v.KeyID, Salt: dec(v.Salt), Size: v.Size, Offset: v.Offset, Index: v.Index, Length: uint32(len(plain))}
	got, e := SealPayload(key, h, plain)
	if e != nil || !bytes.Equal(got, box) {
		t.Fatal("wire vector", e)
	}
	got, e = OpenPayload(key, h, box)
	if e != nil || !bytes.Equal(got, plain) {
		t.Fatal("read vector", e)
	}
	for i := range box {
		bad := bytes.Clone(box)
		bad[i] ^= 1
		if _, e = OpenPayload(key, h, bad); e == nil {
			t.Fatalf("unauthenticated byte %d", i)
		}
	}
	for i := 0; i < len(box); i++ {
		if _, e = OpenPayload(key, h, box[:i]); e == nil {
			t.Fatal("truncation accepted")
		}
	}
	if _, e = OpenPayload(key, h, append(bytes.Clone(box), 0)); e == nil {
		t.Fatal("trailing data")
	}
	for _, mutate := range []func(*PayloadHeader){func(p *PayloadHeader) { p.FileID = p.KeyID }, func(p *PayloadHeader) { p.KeyID = p.FileID }, func(p *PayloadHeader) { p.Size++ }, func(p *PayloadHeader) { p.Offset-- }, func(p *PayloadHeader) { p.Index++ }, func(p *PayloadHeader) { p.Length-- }} {
		different := h
		mutate(&different)
		if _, e = OpenPayload(key, different, box); e == nil {
			t.Fatal("metadata substitution accepted")
		}
	}
	// A failed write retried with different bytes gets a fresh object identity/subkey.
	retry := h
	retry.ObjectID = "44444444-4444-4444-8444-444444444444"
	other, e := SealPayload(key, retry, plain)
	if e != nil || bytes.Equal(other[PayloadHeaderSize:], box[PayloadHeaderSize:]) {
		t.Fatal("retry subkey reused", e)
	}
}

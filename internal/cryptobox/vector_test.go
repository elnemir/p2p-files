package cryptobox

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Fixed vector generated independently with Node/OpenSSL AES-256-GCM.
func TestEnvelopeBinaryVector(t *testing.T) {
	key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	box, _ := hex.DecodeString("01000102030405060708090a0b016bba7e9dc5b47eee35f8f98bc9a8d95366578920c68ef7e8d2af9cece04823873c16683d9b9e2509dd")
	want, _ := hex.DecodeString("46696c655820766563746f723a20d0b4d0b0d0bdd0bdd18bd0b5")
	got, e := Open(key, "filex/test/v1/vector_1", box)
	if e != nil || !bytes.Equal(got, want) {
		t.Fatal("wire format changed", e)
	}
}

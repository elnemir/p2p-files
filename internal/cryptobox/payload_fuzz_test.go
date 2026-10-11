package cryptobox

import (
	"bytes"
	"testing"
)

func FuzzPayloadAuthentication(f *testing.F) {
	f.Add([]byte("immutable payload"), uint64(0), uint16(0))
	f.Add([]byte{0, 255, 13, 10}, uint64(1<<32), uint16(177))
	f.Fuzz(func(t *testing.T, plain []byte, index uint64, mutation uint16) {
		if len(plain) == 0 || len(plain) > 65536 {
			return
		}
		key := bytes.Repeat([]byte{42}, 32)
		h := PayloadHeader{ObjectID: "11111111-1111-4111-8111-111111111111", FileID: "22222222-2222-4222-8222-222222222222", KeyID: "33333333-3333-4333-8333-333333333333", Salt: bytes.Repeat([]byte{1}, 32), Size: uint64(len(plain)), Length: uint32(len(plain)), Index: index}
		box, err := SealPayload(key, h, plain)
		if err != nil {
			t.Fatal(err)
		}
		got, err := OpenPayload(key, h, box)
		if err != nil || !bytes.Equal(got, plain) {
			t.Fatal("round-trip failed", err)
		}
		box[int(mutation)%len(box)] ^= 1
		if _, err = OpenPayload(key, h, box); err == nil {
			t.Fatal("unauthenticated mutation accepted")
		}
	})
}

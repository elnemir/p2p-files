package cryptobox

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"regexp"
)

const PayloadHeaderSize = 177
const PayloadChunkSize = 4 << 20

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ObjectID is a fresh chunk UUID for every write attempt, including failed writes.
// Its subkey prevents nonce reuse after retries or restoration of an older DB.
type PayloadHeader struct {
	ObjectID, FileID, KeyID string
	Salt                    []byte
	Size, Offset, Index     uint64
	Length                  uint32
}

func (h PayloadHeader) bytes() ([]byte, error) {
	if !uuidPattern.MatchString(h.ObjectID) || !uuidPattern.MatchString(h.FileID) || !uuidPattern.MatchString(h.KeyID) || len(h.Salt) != 32 || h.Size > 1<<40 || h.Offset > h.Size || h.Length == 0 || h.Length > PayloadChunkSize || uint64(h.Length) > h.Size-h.Offset {
		return nil, ErrEnvelope
	}
	b := make([]byte, PayloadHeaderSize)
	copy(b, "FXP1")
	b[4] = 1 // Payload class; other encrypted classes cannot use this format.
	copy(b[5:41], h.ObjectID)
	copy(b[41:77], h.FileID)
	copy(b[77:113], h.KeyID)
	copy(b[113:145], h.Salt)
	binary.BigEndian.PutUint64(b[145:153], h.Size)
	binary.BigEndian.PutUint32(b[153:157], PayloadChunkSize)
	binary.BigEndian.PutUint64(b[157:165], h.Offset)
	binary.BigEndian.PutUint32(b[165:169], h.Length)
	binary.BigEndian.PutUint64(b[169:177], h.Index)
	return b, nil
}
func payloadAEAD(fileKey []byte, h PayloadHeader) (cipher.AEAD, []byte, error) {
	if len(fileKey) != 32 {
		return nil, nil, ErrEnvelope
	}
	key, e := hkdf.Key(sha256.New, fileKey, nil, "filex/payload/chunk/v1/"+h.ObjectID, 32)
	if e != nil {
		return nil, nil, e
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, nil, e
	}
	a, e := cipher.NewGCM(block)
	if e != nil {
		return nil, nil, e
	}
	nonce := make([]byte, 12)
	binary.BigEndian.PutUint64(nonce[4:], h.Index)
	return a, nonce, nil
}
func SealPayload(fileKey []byte, h PayloadHeader, plain []byte) ([]byte, error) {
	header, e := h.bytes()
	if e != nil || len(plain) != int(h.Length) {
		return nil, ErrEnvelope
	}
	a, nonce, e := payloadAEAD(fileKey, h)
	if e != nil {
		return nil, e
	}
	return a.Seal(header, nonce, plain, header), nil
}
func OpenPayload(fileKey []byte, h PayloadHeader, box []byte) ([]byte, error) {
	header, e := h.bytes()
	if e != nil || len(box) != PayloadHeaderSize+int(h.Length)+16 || !bytes.Equal(box[:PayloadHeaderSize], header) {
		return nil, ErrEnvelope
	}
	a, nonce, e := payloadAEAD(fileKey, h)
	if e != nil {
		return nil, e
	}
	plain, e := a.Open(nil, nonce, box[PayloadHeaderSize:], header)
	if e != nil {
		return nil, ErrEnvelope
	}
	return plain, nil
}

package cryptobox

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"regexp"
)

var ErrEnvelope = errors.New("invalid encrypted envelope")
var contextPattern = regexp.MustCompile(`^filex/[a-z_]+/v[1-9][0-9]*/[a-zA-Z0-9_-]{1,128}$`)

func aead(key []byte, context string) (cipher.AEAD, error) {
	if len(key) != 32 || !contextPattern.MatchString(context) {
		return nil, ErrEnvelope
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, ErrEnvelope
	}
	return cipher.NewGCMWithRandomNonce(block)
}
func Seal(key []byte, context string, plaintext []byte) ([]byte, error) {
	a, err := aead(key, context)
	if err != nil {
		return nil, err
	}
	return a.Seal([]byte{1}, nil, plaintext, []byte(context)), nil
}
func Open(key []byte, context string, envelope []byte) ([]byte, error) {
	a, err := aead(key, context)
	if err != nil {
		return nil, err
	}
	if len(envelope) < 1+a.Overhead() || envelope[0] != 1 {
		return nil, ErrEnvelope
	}
	b, err := a.Open(nil, nil, envelope[1:], []byte(context))
	if err != nil {
		return nil, ErrEnvelope
	}
	return b, nil
}

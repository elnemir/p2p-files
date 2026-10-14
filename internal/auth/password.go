package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"net/mail"
	"strings"
)

var ErrCredentials = errors.New("invalid credentials")
var ErrForbidden = errors.New("forbidden")
var ErrRate = errors.New("rate limited")
var ErrConflict = errors.New("setup already completed")
var ErrInvalid = errors.New("invalid input")
var work = make(chan struct{}, 4)

func Email(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	a, e := mail.ParseAddress(v)
	if e != nil || a.Address != v || len(v) > 254 {
		return "", ErrInvalid
	}
	return v, nil
}
func ValidatePassword(p string) error {
	if len(p) < 12 || len(p) > 1024 {
		return ErrInvalid
	}
	switch strings.ToLower(p) {
	case "password1234", "password12345", "123456789012", "qwerty123456", "qwerty123456789", "changeme12345":
		return ErrInvalid
	}
	return nil
}
func Hash(p string) (string, error) {
	if err := ValidatePassword(p); err != nil {
		return "", err
	}
	return hash(p)
}

// LinkSecret permits a numeric PIN, protected by the capability's durable limiter.
func LinkSecret(p string) (string, error) {
	pin := len(p) >= 6 && len(p) <= 12
	for _, c := range p {
		if c < '0' || c > '9' {
			pin = false
		}
	}
	if !pin {
		if e := ValidatePassword(p); e != nil {
			return "", e
		}
	}
	return hash(p)
}
func hash(p string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	work <- struct{}{}
	defer func() { <-work }()
	hash := argon2.IDKey([]byte(p), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func Verify(encoded, p string) bool {
	if len(p) > 1024 {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=3,p=2" {
		return false
	}
	salt, e := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if e != nil || len(salt) != 16 {
		return false
	}
	expected, e := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if e != nil || len(expected) != 32 {
		return false
	}
	work <- struct{}{}
	defer func() { <-work }()
	actual := argon2.IDKey([]byte(p), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

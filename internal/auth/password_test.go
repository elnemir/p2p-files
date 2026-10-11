package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashBounds(t *testing.T) {
	h, e := Hash("strong-password-0123")
	if e != nil {
		t.Fatal(e)
	}
	if !Verify(h, "strong-password-0123") || Verify(h, "wrong-password-0123") {
		t.Fatal("password check failed")
	}
	for _, p := range []string{"short", "password1234", strings.Repeat("x", 1025)} {
		if _, e := Hash(p); e == nil {
			t.Fatal("weak/oversized password")
		}
	}
	if Verify(strings.Replace(h, "m=65536", "m=999999999", 1), "strong-password-0123") {
		t.Fatal("unbounded stored hash accepted")
	}
}

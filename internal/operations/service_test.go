package operations

import (
	"strings"
	"testing"
)

func TestReasonUnicodeBounds(t *testing.T) {
	for _, reason := range []string{"сверка", strings.Repeat("я", 2000), strings.Repeat("🔑", 2000)} {
		if !validReason(reason) {
			t.Fatal("valid Unicode reason rejected")
		}
	}
	for _, reason := range []string{" \t\n ", "ключ", "  ключ  ", strings.Repeat("я", 2001)} {
		if validReason(reason) {
			t.Fatal("reason outside character bounds accepted")
		}
	}
}

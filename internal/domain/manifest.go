package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrInvalid = errors.New("invalid input")

type File struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func ValidID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	h := strings.ReplaceAll(s, "-", "")
	_, err := hex.DecodeString(h)
	return err == nil && len(h) == 32 && strings.ToLower(s) == s
}

func ValidChecksum(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && strings.ToLower(s) == s
}

func PathKey(path string, maxBytes int) (string, error) {
	if maxBytes < 1 || len(path) == 0 || len(path) > maxBytes || !utf8.ValidString(path) || !norm.NFC.IsNormalString(path) || strings.Contains(path, "\\") {
		return "", ErrInvalid
	}
	for _, r := range path {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", ErrInvalid
		}
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || strings.ContainsAny(part, ":*?\"<>|") {
			return "", ErrInvalid
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
			return "", ErrInvalid
		}
		switch strings.ToLower(part) {
		case "sha256sums.txt", "sha256sums.sig", "sha256sums.pub":
			return "", ErrInvalid
		}
	}
	// Simple case folding matches Unicode EqualFold and avoids ambiguous ZIP entries.
	key := []rune(path)
	for i, r := range key {
		min := r
		for n := unicode.SimpleFold(r); n != r; n = unicode.SimpleFold(n) {
			if n < min {
				min = n
			}
		}
		key[i] = min
	}
	return string(key), nil
}

func BuildManifest(files []File, maxFiles, maxPathBytes int) ([]byte, string, error) {
	if len(files) == 0 || len(files) > maxFiles {
		return nil, "", ErrInvalid
	}
	copyFiles := slices.Clone(files)
	slices.SortFunc(copyFiles, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	seen := make(map[string]bool, len(files))
	ids := make(map[string]bool, len(files))
	for _, f := range copyFiles {
		key, err := PathKey(f.Path, maxPathBytes)
		if err != nil || !ValidID(f.ID) || !ValidChecksum(f.SHA256) || f.Size < 0 || seen[key] || ids[f.ID] {
			return nil, "", ErrInvalid
		}
		seen[key] = true
		ids[f.ID] = true
	}
	// A file may not also be a directory, including under a case-folded spelling.
	for key := range seen {
		for i := strings.IndexByte(key, '/'); i >= 0; {
			if seen[key[:i]] {
				return nil, "", ErrInvalid
			}
			next := strings.IndexByte(key[i+1:], '/')
			if next < 0 {
				break
			}
			i += next + 1
		}
	}
	b, err := json.Marshal(Manifest{Version: 1, Files: copyFiles})
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}

func Expiry(now time.Time, ttlSeconds, maxTTLSeconds int64) (time.Time, time.Time, error) {
	if ttlSeconds < 1 || maxTTLSeconds < 1 || ttlSeconds > maxTTLSeconds || maxTTLSeconds > 31536000 {
		return time.Time{}, time.Time{}, fmt.Errorf("ttl: %w", ErrInvalid)
	}
	return now.Add(time.Duration(ttlSeconds) * time.Second), now.Add(time.Duration(maxTTLSeconds) * time.Second), nil
}

func ResumeExpiry(now, absolute time.Time, remaining time.Duration) time.Time {
	if remaining < 0 {
		remaining = 0
	}
	resumed := now.Add(remaining)
	if resumed.After(absolute) {
		return absolute
	}
	return resumed
}

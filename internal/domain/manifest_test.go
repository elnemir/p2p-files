package domain

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPathsRejectUnsafeAndAmbiguousNames(t *testing.T) {
	for _, p := range []string{"", "/etc/passwd", "../x", "a/../b", "a//b", "a/./b", "C:/x", "a\\b", "a/", "x\n.txt", "x\x00", "x\u202eexe", "NUL.txt", "dir/COM1", "foo.", "foo ", "SHA256SUMS.txt", "a:b", "e\u0301.txt", string([]byte{0xff})} {
		t.Run(p, func(t *testing.T) {
			if _, err := PathKey(p, 1024); err == nil {
				t.Fatalf("accepted unsafe path %q", p)
			}
		})
	}
	for _, p := range []string{"файлы/отчёт.pdf", "dir/CONtent.txt", ".hidden", "a b/é.txt"} {
		if _, err := PathKey(p, 1024); err != nil {
			t.Errorf("valid path %q: %v", p, err)
		}
	}
	if _, err := PathKey("long", 3); err == nil {
		t.Fatal("path bound ignored")
	}
}
func fixture(path, id string) File {
	return File{ID: id, Path: path, Size: 0, SHA256: strings.Repeat("a", 64)}
}
func TestManifestDeterminismAndCollisions(t *testing.T) {
	a := fixture("z/файл.txt", "00000000-0000-4000-8000-000000000001")
	b := fixture("a.txt", "00000000-0000-4000-8000-000000000002")
	original := []File{a, b}
	one, h1, err := BuildManifest(original, 10, 1024)
	if err != nil {
		t.Fatal(err)
	}
	two, h2, err := BuildManifest([]File{b, a}, 10, 1024)
	if err != nil || !bytes.Equal(one, two) || h1 != h2 {
		t.Fatal("manifest depends on insertion order")
	}
	if original[0] != a {
		t.Fatal("caller slice mutated")
	}
	for _, paths := range [][2]string{{"A.txt", "a.txt"}, {"folder", "folder/x"}, {"Folder", "folder/x"}, {"Σ.txt", "ς.txt"}} {
		a.Path = paths[0]
		b.Path = paths[1]
		if _, _, err := BuildManifest([]File{a, b}, 10, 1024); err == nil {
			t.Errorf("collision accepted %v", paths)
		}
	}
	if _, _, err := BuildManifest(nil, 10, 1024); err == nil {
		t.Fatal("empty manifest accepted")
	}
	a.Path = "a"
	a.Size = -1
	if _, _, err := BuildManifest([]File{a}, 10, 1024); err == nil {
		t.Fatal("negative size accepted")
	}
}
func TestExpiryAndFreezeHardCap(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	expiry, absolute, err := Expiry(now, 60, 120)
	if err != nil || !expiry.Equal(now.Add(time.Minute)) || !absolute.Equal(now.Add(2*time.Minute)) {
		t.Fatal("incorrect deadlines")
	}
	for _, ttl := range []int64{-1, 0, 121} {
		if _, _, err := Expiry(now, ttl, 120); err == nil {
			t.Fatal("invalid ttl accepted")
		}
	}
	if got := ResumeExpiry(now.Add(5*time.Minute), absolute, time.Hour); !got.Equal(absolute) {
		t.Fatal("freeze extended absolute expiry")
	}
	if got := ResumeExpiry(now, absolute, 30*time.Second); !got.Equal(now.Add(30 * time.Second)) {
		t.Fatal("remaining ttl lost")
	}
}
func FuzzPathKey(f *testing.F) {
	for _, p := range []string{"a/b.txt", "../x", "файл.txt", "a\x00"} {
		f.Add(p)
	}
	f.Fuzz(func(t *testing.T, p string) {
		key, err := PathKey(p, 1024)
		if err == nil {
			if key == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") {
				t.Fatalf("unsafe accepted %q", p)
			}
		}
	})
}

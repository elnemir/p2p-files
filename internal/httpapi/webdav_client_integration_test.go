package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The optional client is installed separately from runtime dependencies and its
// verified binary path is explicit. No user rclone configuration is consulted.
func TestIntegrationWebDAVRcloneClient(t *testing.T) {
	binary := os.Getenv("FILEX_RCLONE_PATH")
	if binary == "" {
		t.Skip("requires an independently installed rclone client via FILEX_RCLONE_PATH")
	}
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	server := httptest.NewServer(c.h)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, e := c.a.DB.Exec(ctx, "UPDATE app_settings SET value=jsonb_set(value,'{publicOrigin}',to_jsonb($1::text)) WHERE active", server.URL); e != nil {
		t.Fatal(e)
	}
	env := []string{}
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "RCLONE_") {
			env = append(env, item)
		}
	}
	obscure := exec.CommandContext(ctx, binary, "obscure", "-")
	obscure.Stdin = strings.NewReader(cred.Password + "\n")
	obscure.Env = env
	encoded, e := obscure.Output()
	if e != nil {
		t.Fatal("rclone could not encode disposable credential", e)
	}
	env = append(env, "RCLONE_CONFIG_DAV_TYPE=webdav", "RCLONE_CONFIG_DAV_VENDOR=other", "RCLONE_CONFIG_DAV_URL="+server.URL+davPrefix+id+"/", "RCLONE_CONFIG_DAV_USER=filex", "RCLONE_CONFIG_DAV_PASS="+strings.TrimSpace(string(encoded)))
	run := func(args ...string) string {
		t.Helper()
		args = append(args, "--config", os.DevNull, "--retries", "1", "--low-level-retries", "1", "--log-level", "ERROR")
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env = env
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("rclone %s failed: %v\n%s", args[0], e, strings.ReplaceAll(string(out), cred.Password, "[redacted]"))
		}
		return string(out)
	}
	data := []byte(strings.Repeat("independent WebDAV client\n", 200000))
	source := filepath.Join(t.TempDir(), "sample.txt")
	if e = os.WriteFile(source, data, 0600); e != nil {
		t.Fatal(e)
	}
	run("mkdir", "dav:documents")
	run("copyto", source, "dav:documents/договор.txt")
	listing := run("lsf", "dav:", "--recursive")
	if !strings.Contains(listing, "documents/договор.txt") {
		t.Fatal("client did not list uploaded file", listing)
	}
	var checksum string
	if e = c.a.DB.QueryRow(ctx, "SELECT sha256 FROM files WHERE transfer_id=$1 AND path='documents/договор.txt' AND state='ready'", id).Scan(&checksum); e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(data)
	if checksum != hex.EncodeToString(hash[:]) {
		t.Fatal("client upload changed bytes")
	}
	run("moveto", "dav:documents/договор.txt", "dav:documents/renamed.txt")
	run("deletefile", "dav:documents/renamed.txt")
	run("rmdir", "dav:documents")
	if listing = run("lsf", "dav:"); strings.TrimSpace(listing) != "" {
		t.Fatal("client deletion left entries", listing)
	}
	t.Log("rclone mkdir/copyto/lsf/moveto/deletefile/rmdir passed with a multi-chunk UTF-8 filename and matching server SHA-256")
}

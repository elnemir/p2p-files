package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Explicit opt-in: these run real OS clients, not HTTP fixtures pretending to be them.
func TestNativeWebDAVClients(t *testing.T) {
	client := os.Getenv("FILEX_NATIVE_WEBDAV_CLIENT")
	if client == "" {
		t.Skip("set FILEX_NATIVE_WEBDAV_CLIENT=macos or linux for the actual native client")
	}
	if client != "macos" && client != "linux" {
		t.Fatal("unsupported native test client")
	}
	c, _ := testClient(t)
	id := c.draft()
	cred := davCredential(t, c, id)
	davStatus(t, davRequest(c, cred, "PUT", "from-server.txt", "server fixture for native read\n", nil), 201)
	var mu sync.Mutex
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &davStatusWriter{ResponseWriter: w}
		c.h.ServeHTTP(rec, r)
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.EscapedPath()+" "+httpStatusNumber(rec.status))
		mu.Unlock()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if _, e := c.a.DB.Exec(ctx, "UPDATE app_settings SET value=jsonb_set(value,'{publicOrigin}',to_jsonb($1::text)) WHERE active", server.URL); e != nil {
		t.Fatal(e)
	}
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	var cmd *exec.Cmd
	if client == "macos" {
		cmd = exec.CommandContext(ctx, "python3", filepath.Join(root, "scripts/webdav/macos.py"))
	} else {
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "--init", "-i", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true", "--mount", "type=bind,src="+filepath.Join(root, "scripts/webdav")+",dst=/checks,readonly", "filex-webdav-gvfs-test", "dbus-run-session", "--", "python3", "/checks/linux.py")
	}
	raw, e := json.Marshal(map[string]string{"url": server.URL + davPrefix + id + "/", "password": cred.Password})
	if e != nil {
		t.Fatal(e)
	}
	// Give filesystem clients time to unmount before a timed-out process is killed.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 20 * time.Second
	cmd.Stdin = strings.NewReader(string(raw))
	output, err := cmd.CombinedOutput()
	mu.Lock()
	defer mu.Unlock()
	for _, line := range requests {
		t.Log(line)
	}
	if err != nil {
		t.Fatalf("native %s: %v\n%s", client, err, strings.ReplaceAll(string(output), cred.Password, "[redacted]"))
	}
	t.Log(strings.ReplaceAll(string(output), cred.Password, "[redacted]"))
}

type davStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *davStatusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *davStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func httpStatusNumber(n int) string {
	if n == 0 {
		n = 200
	}
	return fmt.Sprint(n)
}

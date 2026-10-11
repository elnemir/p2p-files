package httpapi

import (
	"context"
	"filex.local/filex/internal/communications"
	"github.com/coder/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntegrationWebSocketOriginPrivacyAndRevocation(t *testing.T) {
	c, _ := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id := c.draft()
	c.upload(id, "a", "ws")
	c.publish(id)
	if e := (communications.Service{DB: c.a.DB, Keys: c.a.Keys}).Fanout(ctx); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.Host = "localhost:8090"; c.h.ServeHTTP(w, r) }))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/events"
	headers := http.Header{"Cookie": []string{c.cookie.String()}, "Origin": []string{"https://other.example.test"}}
	conn, response, e := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: headers})
	if e == nil {
		conn.CloseNow()
		t.Fatal("cross-origin WS accepted")
	}
	if response == nil || response.StatusCode != 403 {
		t.Fatal("origin status", e)
	}
	headers.Set("Origin", "http://localhost:8090")
	conn, _, e = websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: headers})
	if e != nil {
		t.Fatal(e)
	}
	defer conn.CloseNow()
	_, b, e := conn.Read(ctx)
	if e != nil || !strings.Contains(string(b), "transfer.published") || strings.Contains(string(b), "recipientEmail") {
		t.Fatal("owner event", e, string(b))
	}
	if _, e = c.a.DB.Exec(ctx, "UPDATE web_sessions SET revoked_at=clock_timestamp()"); e != nil {
		t.Fatal(e)
	}
	_, _, e = conn.Read(ctx)
	if e == nil {
		t.Fatal("revoked WS retained")
	}
}

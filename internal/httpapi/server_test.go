package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type pinger struct{ err error }

func (p pinger) Ping(context.Context) error { return p.err }
func TestStatusPrivacyAndInternalIsolation(t *testing.T) {
	for _, tc := range []struct {
		err    error
		code   int
		status string
	}{{nil, 200, "operational"}, {errors.New("postgres://secret@internal:5432"), 503, "unavailable"}} {
		w := httptest.NewRecorder()
		Public(pinger{tc.err}).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/status?secret=foo", nil))
		if w.Code != tc.code || w.Body.String() != `{"status":"`+tc.status+`"}`+"\n" {
			t.Fatalf("bad response %d %s", w.Code, w.Body.String())
		}
		for k, v := range map[string]string{"Cache-Control": "no-store", "Referrer-Policy": "no-referrer", "X-Content-Type-Options": "nosniff"} {
			if w.Header().Get(k) != v {
				t.Fatal("missing privacy header", k)
			}
		}
	}
	for _, path := range []string{"/health", "/ready", "/metrics", "/api/v1/transfers"} {
		w := httptest.NewRecorder()
		Public(pinger{}).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatal("unimplemented/internal route exposed", path)
		}
	}
	w := httptest.NewRecorder()
	Public(pinger{}).ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/status", strings.NewReader("secret")))
	if w.Code != 405 {
		t.Fatal("POST status accepted")
	}
	w = httptest.NewRecorder()
	Internal(pinger{errors.New("down")}).ServeHTTP(w, httptest.NewRequest("GET", "/ready", nil))
	if w.Code != 503 {
		t.Fatal("ready with database down")
	}
}

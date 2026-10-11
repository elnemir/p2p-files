package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Pinger interface{ Ping(context.Context) error }

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Static response values are serializable; write errors cannot safely change an already sent status.
	_ = json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{value})
}
func Public(db Pinger) http.Handler {
	mux := http.NewServeMux()
	status := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if db.Ping(ctx) != nil {
			writeStatus(w, 503, "unavailable")
			return
		}
		writeStatus(w, 200, "operational")
	}
	mux.HandleFunc("GET /api/v1/status", status)
	return headers(mux)
}
func Internal(db Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { writeStatus(w, 200, "alive") })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if db.Ping(ctx) != nil {
			writeStatus(w, 503, "not_ready")
			return
		}
		writeStatus(w, 200, "ready")
	})
	return headers(mux)
}

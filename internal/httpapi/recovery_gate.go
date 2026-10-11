package httpapi

import (
	"context"
	"filex.local/filex/internal/maintenance"
	"net/http"
	"strings"
)

func (a *App) Ping(ctx context.Context) error {
	if e := a.DB.Ping(ctx); e != nil {
		return e
	}
	state, e := a.Operations.State(ctx)
	if e != nil {
		return e
	}
	if state.Generation != a.Gate.Generation {
		return maintenance.ErrRestart
	}
	if state.Mode != "open" {
		return maintenance.ErrClosed
	}
	return nil
}
func (a *App) recoveryGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status" {
			if e := a.Ping(r.Context()); e != nil {
				writeStatus(w, 503, "unavailable")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		control := strings.HasPrefix(r.URL.Path, "/api/v1/admin/") || r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/me" || r.URL.Path == "/api/v1/auth/logout" || r.URL.Path == "/api/v1/auth/reauth"
		ctx, release, e := a.Gate.Acquire(r.Context(), control)
		if e != nil {
			failure(w, e)
			return
		}
		defer release()
		next.ServeHTTP(guardedResponse{ResponseWriter: w, ctx: ctx}, r.WithContext(ctx))
	})
}

type guardedResponse struct {
	http.ResponseWriter
	ctx context.Context
}

func (w guardedResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w guardedResponse) Write(b []byte) (int, error) {
	if e := w.ctx.Err(); e != nil {
		return 0, e
	}
	return w.ResponseWriter.Write(b)
}

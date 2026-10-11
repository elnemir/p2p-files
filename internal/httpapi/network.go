package httpapi

import (
	"context"
	"errors"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/networkpolicy"
	"net/http"
)

type clientAddressKey struct{}

func (a *App) networkGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, e := database.Settings(r.Context(), a.DB)
		if e != nil {
			failure(w, e)
			return
		}
		client, e := networkpolicy.Resolve(r, s)
		if e != nil {
			if errors.Is(e, networkpolicy.ErrMalformed) {
				failure(w, auth.ErrInvalid)
			} else {
				failure(w, auth.ErrForbidden)
			}
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientAddressKey{}, client.String())))
	})
}

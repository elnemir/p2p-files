package httpapi

import (
	"encoding/json"
	"filex.local/filex/internal/auth"
	"io"
	"net/http"
	"strings"
)

// Native form downloads stream directly to the browser download manager. CSRF is
// supplied as a hidden form field, then verified by the same protected handler.
func forensicForm(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			r.Body = http.MaxBytesReader(w, r.Body, 65536)
			if r.ParseForm() != nil {
				failure(w, auth.ErrInvalid)
				return
			}
			r.Header.Set("X-CSRF-Token", r.PostForm.Get("_csrf"))
			v := map[string]any{"reason": r.PostForm.Get("reason")}
			if !strings.HasSuffix(r.URL.Path, "/forensic-zip") {
				v["rawCiphertext"] = r.PostForm.Get("rawCiphertext") == "true"
				v["objectKey"] = r.PostForm.Get("objectKey")
			}
			b, e := json.Marshal(v)
			if e != nil {
				failure(w, e)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(b)))
			r.Header.Set("Content-Type", "application/json")
		}
		next(w, r)
	}
}

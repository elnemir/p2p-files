package httpapi

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/database"
	"net/http"
	"reflect"
	"testing"
)

func TestIntegrationTransferEditorCompatibilityAndOwnerIsolation(t *testing.T) {
	c, _ := testClient(t)
	ctx := context.Background()
	id := c.draft()
	path := "/api/v1/transfers/" + id
	if w := c.req("PATCH", path, `{"title":"Editor context","message":"Keep the existing message"}`, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
	var original, editor map[string]json.RawMessage
	w := c.req("GET", path, "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	decode(t, w, &original)
	w = c.req("GET", path+"/editor", "", nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	decode(t, w, &editor)
	settings, e := database.Settings(ctx, c.a.DB)
	if e != nil {
		t.Fatal(e)
	}
	var message string
	var limit int64
	if e = json.Unmarshal(editor["message"], &message); e != nil || message != "Keep the existing message" {
		t.Fatal("editor lost message", e)
	}
	if e = json.Unmarshal(editor["maxTtlSeconds"], &limit); e != nil || limit != settings.TransferMaxTTLSeconds {
		t.Fatal("editor policy differs from active settings", e)
	}
	delete(editor, "message")
	delete(editor, "maxTtlSeconds")
	if !reflect.DeepEqual(original, editor) || len(original) != 7 {
		t.Fatal("legacy DTO changed or read mutated transfer state")
	}
	public := c
	public.cookie = nil
	public.csrf = ""
	if w = public.req("GET", path+"/editor", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatal("anonymous editor access", w.Code)
	}
	for scope, status := range map[string]int{"account:read": 403, "transfers:read": 200} {
		w = c.req("POST", "/api/v1/auth/tokens", `{"scopes":["`+scope+`"],"ttlSeconds":600}`, nil)
		if w.Code != 201 {
			t.Fatal(w.Code)
		}
		var token struct{ Token string }
		decode(t, w, &token)
		if w = public.req("GET", path+"/editor", "", map[string]string{"Authorization": "Bearer " + token.Token}); w.Code != status {
			t.Fatal("editor scope boundary", scope, w.Code)
		}
	}
	if _, e = c.a.Auth.CreateUser(ctx, "host_cli", "other-editor@example.test", "strong-password-0123", "admin"); e != nil {
		t.Fatal(e)
	}
	w = public.req("POST", "/api/v1/auth/login", `{"email":"other-editor@example.test","password":"strong-password-0123"}`, nil)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	public.cookie = w.Result().Cookies()[0]
	if w = public.req("GET", path+"/editor", "", nil); w.Code != http.StatusNotFound {
		t.Fatal("another administrator can read owner editor context", w.Code)
	}
}

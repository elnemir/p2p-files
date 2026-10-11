package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/transfer"
	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"time"
)

func (a *App) communicationsRoutes(m *http.ServeMux) {
	m.HandleFunc("PUT /api/v1/admin/smtp", a.protect("admin:communications", true, a.smtpConfig))
	m.HandleFunc("POST /api/v1/webhooks", a.protect("webhooks:write", false, a.webhookConfig))
	m.HandleFunc("DELETE /api/v1/webhooks/{id}", a.protect("webhooks:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		_, e := a.DB.Exec(r.Context(), "UPDATE communication_configs SET enabled=false WHERE id::text=$1 AND owner_id=$2 AND kind='webhook'", r.PathValue("id"), p.ID)
		if e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("GET /api/v1/notifications", a.protect("notifications:read", false, a.notifications))
	m.HandleFunc("POST /api/v1/notifications/{id}/read", a.protect("notifications:write", false, func(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
		_, e := a.DB.Exec(r.Context(), "UPDATE notifications SET read_at=clock_timestamp() WHERE id::text=$1 AND user_id=$2", r.PathValue("id"), p.ID)
		if e != nil {
			return e
		}
		w.WriteHeader(204)
		return nil
	}))
	m.HandleFunc("GET /api/v1/notification-preferences", a.protect("account:read", false, a.getPreferences))
	m.HandleFunc("PUT /api/v1/notification-preferences", a.protect("account:write", false, a.putPreferences))
	m.HandleFunc("GET /api/v1/events", a.protect("notifications:read", false, a.events))
	m.HandleFunc("GET /api/v1/deliveries", a.protect("notifications:read", false, a.deliveries))
	m.HandleFunc("POST /api/v1/deliveries/{id}/retry", a.protect("notifications:write", false, a.retryDelivery))
}
func (a *App) smtpConfig(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v communications.SMTPConfig
	if e := body(w, r, &v); e != nil {
		return e
	}
	if e := v.Validate(); e != nil {
		return auth.ErrInvalid
	}
	kid, key, e := a.Keys.Active(r.Context(), "configuration")
	if e != nil {
		return e
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(724185003)"); e != nil {
		return e
	}
	id := domain.NewID()
	var existing *string
	if e = tx.QueryRow(r.Context(), "SELECT min(id::text) FROM communication_configs WHERE kind='smtp'").Scan(&existing); e != nil {
		return e
	}
	if existing != nil {
		id = *existing
	}
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	box, e := a.Keys.Seal(r.Context(), kid, key, "filex/communications/v1/"+id, raw)
	if e != nil {
		return e
	}
	if e = keyring.RequireActive(r.Context(), tx, kid); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO communication_configs(id,kind,key_id,envelope) VALUES($1,'smtp',$2,$3) ON CONFLICT(id) DO UPDATE SET key_id=excluded.key_id,envelope=excluded.envelope,enabled=true", id, kid, box); e != nil {
		return e
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "smtp.configured", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 200, map[string]string{"id": id, "status": "configured"})
}
func (a *App) webhookConfig(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v communications.WebhookConfig
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.Secret == "" {
		v.Secret = auth.Token()
	}
	if e := v.Validate(); e != nil {
		return auth.ErrInvalid
	}
	kid, key, e := a.Keys.Active(r.Context(), "configuration")
	if e != nil {
		return e
	}
	tx, e := a.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(r.Context(), "SELECT id FROM users WHERE id=$1 FOR UPDATE", p.ID); e != nil {
		return e
	}
	var n int
	if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM communication_configs WHERE owner_id=$1 AND enabled", p.ID).Scan(&n); e != nil {
		return e
	}
	if n >= 10 {
		return auth.ErrInvalid
	}
	id := domain.NewID()
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	box, e := a.Keys.Seal(r.Context(), kid, key, "filex/communications/v1/"+id, raw)
	if e != nil {
		return e
	}
	if e = keyring.RequireActive(r.Context(), tx, kid); e != nil {
		return e
	}
	if _, e = tx.Exec(r.Context(), "INSERT INTO communication_configs(id,owner_id,kind,key_id,envelope) VALUES($1,$2,'webhook',$3,$4)", id, p.ID, kid, box); e != nil {
		return e
	}
	if e = audit.Append(r.Context(), tx, a.Key, p.ID, "webhook.created", id); e != nil {
		return e
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return reply(w, 201, map[string]string{"id": id, "secret": v.Secret})
}

type preference struct {
	Locale     string `json:"locale"`
	Timezone   string `json:"timezone"`
	Digest     string `json:"digest"`
	QuietStart *int   `json:"quietStart"`
	QuietEnd   *int   `json:"quietEnd"`
	Browser    bool   `json:"browserEnabled"`
}

func (a *App) getPreferences(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	v := preference{Locale: p.Locale, Timezone: "UTC", Digest: "immediate"}
	var raw []byte
	e := a.DB.QueryRow(r.Context(), "SELECT jsonb_build_object('locale',locale,'timezone',timezone,'digest',digest,'quietStart',quiet_start,'quietEnd',quiet_end,'browserEnabled',browser_enabled) FROM notification_preferences WHERE user_id=$1", p.ID).Scan(&raw)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if len(raw) > 0 {
		if e = json.Unmarshal(raw, &v); e != nil {
			return e
		}
	}
	return reply(w, 200, v)
}
func (a *App) putPreferences(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	var v preference
	if e := body(w, r, &v); e != nil {
		return e
	}
	if v.Locale != "ru" && v.Locale != "en" {
		return auth.ErrInvalid
	}
	if _, e := time.LoadLocation(v.Timezone); e != nil || v.Timezone == "" {
		return auth.ErrInvalid
	}
	if v.Digest != "immediate" && v.Digest != "hourly" && v.Digest != "daily" && v.Digest != "weekly" && v.Digest != "off" {
		return auth.ErrInvalid
	}
	for _, h := range []*int{v.QuietStart, v.QuietEnd} {
		if h != nil && (*h < 0 || *h > 23) {
			return auth.ErrInvalid
		}
	}
	if (v.QuietStart == nil) != (v.QuietEnd == nil) {
		return auth.ErrInvalid
	}
	_, e := a.DB.Exec(r.Context(), "INSERT INTO notification_preferences(user_id,locale,timezone,digest,quiet_start,quiet_end,browser_enabled) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(user_id) DO UPDATE SET locale=excluded.locale,timezone=excluded.timezone,digest=excluded.digest,quiet_start=excluded.quiet_start,quiet_end=excluded.quiet_end,browser_enabled=excluded.browser_enabled", p.ID, v.Locale, v.Timezone, v.Digest, v.QuietStart, v.QuietEnd, v.Browser)
	if e != nil {
		return e
	}
	w.WriteHeader(204)
	return nil
}
func (a *App) notificationList(ctx context.Context, user string, after int64) ([]map[string]any, error) {
	rows, e := a.DB.Query(ctx, "SELECT id,kind,transfer_id::text,created_at,read_at FROM notifications WHERE user_id=$1 AND id>$2 ORDER BY id LIMIT 100", user, after)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind string
		var transferID *string
		var at time.Time
		var read *time.Time
		if e = rows.Scan(&id, &kind, &transferID, &at, &read); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"schemaVersion": 1, "id": id, "kind": kind, "transferId": transferID, "createdAt": at, "readAt": read})
	}
	return out, rows.Err()
}
func (a *App) notifications(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	out, e := a.notificationList(r.Context(), p.ID, max(0, after))
	if e != nil {
		return e
	}
	return reply(w, 200, out)
}
func (a *App) events(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	if p.Bearer || r.Header.Get("Origin") != c.PublicOrigin {
		return auth.ErrForbidden
	}
	conn, e := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
	if e != nil {
		return nil
	}
	defer conn.CloseNow()
	ctx := conn.CloseRead(r.Context())
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		var valid bool
		e = a.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM web_sessions s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND s.revoked_at IS NULL AND s.auth_epoch=u.auth_epoch AND NOT u.disabled AND s.expires_at>clock_timestamp())", p.SessionID).Scan(&valid)
		if e != nil || !valid {
			return nil
		}
		out, e := a.notificationList(ctx, p.ID, after)
		if e != nil {
			return nil
		}
		for _, item := range out {
			b, e := json.Marshal(item)
			if e != nil {
				return nil
			}
			send, cancel := context.WithTimeout(ctx, 3*time.Second)
			e = conn.Write(send, websocket.MessageText, b)
			cancel()
			if e != nil {
				return nil
			}
			after = item["id"].(int64)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (a *App) deliveries(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	rows, e := a.DB.Query(r.Context(), "SELECT id::text,event_type,kind,state,attempts,last_code,created_at FROM deliveries WHERE owner_id=$1 ORDER BY created_at DESC LIMIT 100", p.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, event, kind, state string
		var attempts int
		var code *string
		var at time.Time
		if e = rows.Scan(&id, &event, &kind, &state, &attempts, &code, &at); e != nil {
			return e
		}
		out = append(out, map[string]any{"id": id, "eventType": event, "kind": kind, "state": state, "attempts": attempts, "lastCode": code, "createdAt": at})
	}
	if e = rows.Err(); e != nil {
		return e
	}
	return reply(w, 200, out)
}
func (a *App) retryDelivery(w http.ResponseWriter, r *http.Request, p auth.Principal, c config.Settings) error {
	tag, e := a.DB.Exec(r.Context(), "UPDATE deliveries SET state='pending',available_at=clock_timestamp(),attempts=0,fence=fence+1 WHERE id::text=$1 AND owner_id=$2 AND state IN ('dead','delivered')", r.PathValue("id"), p.ID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return transfer.ErrConflict
	}
	w.WriteHeader(204)
	return nil
}

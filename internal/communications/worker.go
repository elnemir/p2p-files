package communications

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Service struct {
	DB   *pgxpool.Pool
	Keys keyring.Ring
	Mail func(context.Context, SMTPConfig, string, string, string, string) error
	Hook func(context.Context, WebhookConfig, string, []byte) error
}

func (s Service) Fanout(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var event int64
	var transferID, kind, owner, email, locale string
	e = tx.QueryRow(ctx, "SELECT o.id,o.aggregate_id::text,o.event_type,t.owner_id::text,u.email,u.locale FROM outbox o JOIN transfers t ON t.id=o.aggregate_id JOIN users u ON u.id=t.owner_id WHERE o.delivered_at IS NULL ORDER BY o.id FOR UPDATE OF o SKIP LOCKED LIMIT 1").Scan(&event, &transferID, &kind, &owner, &email, &locale)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	eventKey := "event:" + formatID(event)
	if _, e = tx.Exec(ctx, "INSERT INTO notifications(user_id,event_key,kind,transfer_id) VALUES($1,$2,$3,$4) ON CONFLICT(event_key) DO NOTHING", owner, eventKey, kind, transferID); e != nil {
		return e
	}
	if kind == "transfer.published" {
		if _, e = tx.Exec(ctx, `INSERT INTO deliveries(id,event_key,kind,owner_id,link_id,transfer_id,event_type,recipient,system_email,locale) SELECT gen_random_uuid(),$1||':'||l.id::text,'smtp',$2,l.id,l.transfer_id,$3,l.recipient_email,true,$4 FROM recipient_links l WHERE l.transfer_id=$5 AND l.recipient_email IS NOT NULL AND l.revoked_at IS NULL ON CONFLICT(event_key) DO NOTHING`, eventKey, owner, kind, locale, transferID); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO deliveries(id,event_key,kind,owner_id,config_id,transfer_id,event_type,payload) SELECT gen_random_uuid(),$1||':'||c.id::text,'webhook',$2,c.id,$3,$4,jsonb_build_object('eventId',$1::text,'type',$4::text,'transferId',$3::uuid::text,'schemaVersion',1) FROM communication_configs c WHERE c.kind='webhook' AND c.enabled AND c.owner_id=$2 ON CONFLICT(event_key) DO NOTHING`, eventKey, owner, transferID, kind); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE outbox SET delivered_at=clock_timestamp() WHERE id=$1", event); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func formatID(n int64) string { b, _ := json.Marshal(n); return string(b) }
func (s Service) Schedule(ctx context.Context) error {
	if _, e := s.DB.Exec(ctx, `INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) SELECT id,'transfer.expiring','expiring:'||id::text,'{}'::jsonb FROM transfers WHERE state='active' AND NOT frozen AND expires_at>clock_timestamp() AND expires_at<clock_timestamp()+interval '1 hour' ON CONFLICT(deduplication_key) DO NOTHING`); e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185002)"); e != nil {
		return e
	}
	var owner, email, locale, digest, timezone string
	var start, end *int
	var oldest time.Time
	e = tx.QueryRow(ctx, `SELECT u.id::text,u.email,COALESCE(p.locale,u.locale),COALESCE(p.digest,'immediate'),COALESCE(p.timezone,'UTC'),p.quiet_start,p.quiet_end,min(n.created_at) FROM notifications n JOIN users u ON u.id=n.user_id LEFT JOIN notification_preferences p ON p.user_id=u.id WHERE n.emailed_at IS NULL AND NOT u.disabled AND COALESCE(p.digest,'immediate')<>'off' GROUP BY u.id,p.locale,p.digest,p.timezone,p.quiet_start,p.quiet_end HAVING min(n.created_at)<=clock_timestamp()-CASE COALESCE(p.digest,'immediate') WHEN 'hourly' THEN interval '1 hour' WHEN 'daily' THEN interval '1 day' WHEN 'weekly' THEN interval '7 days' ELSE interval '0 seconds' END ORDER BY min(n.created_at) LIMIT 1`).Scan(&owner, &email, &locale, &digest, &timezone, &start, &end, &oldest)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	rows, e := tx.Query(ctx, "SELECT id,kind,transfer_id::text FROM notifications WHERE user_id=$1 AND emailed_at IS NULL ORDER BY id LIMIT 1000 FOR UPDATE", owner)
	if e != nil {
		return e
	}
	var ids []int64
	var kind string
	var transferID *string
	for rows.Next() {
		var id int64
		if e = rows.Scan(&id, &kind, &transferID); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 1 || digest != "immediate" {
		kind = "digest"
		transferID = nil
	}
	payload, _ := json.Marshal(map[string]int{"count": len(ids)})
	id := domain.NewID()
	if _, e = tx.Exec(ctx, "INSERT INTO deliveries(id,event_key,kind,owner_id,transfer_id,event_type,locale,payload,recipient) VALUES($1,$2,'smtp',$3,$4,$5,$6,$7,$8)", id, "digest:"+owner+":"+formatID(ids[0])+":"+formatID(ids[len(ids)-1]), owner, transferID, kind, locale, payload, email); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE notifications SET emailed_at=clock_timestamp() WHERE id=ANY($1)", ids); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) Config(ctx context.Context, id string, out any) error {
	var keyID string
	var box []byte
	if e := s.DB.QueryRow(ctx, "SELECT key_id::text,envelope FROM communication_configs WHERE id::text=$1 AND enabled", id).Scan(&keyID, &box); e != nil {
		return e
	}
	key, e := s.Keys.Get(ctx, keyID)
	if e != nil {
		return e
	}
	raw, e := cryptobox.Open(key, "filex/communications/v1/"+id, box)
	if e != nil {
		return e
	}
	return json.Unmarshal(raw, out)
}
func (s Service) Deliver(ctx context.Context, c config.Settings) error {
	var id, kind, locale, event, state string
	var owner, configID, linkID, to *string
	var payload []byte
	var attempts int
	var fence int64
	var system bool
	e := s.DB.QueryRow(ctx, `WITH next AS(SELECT id FROM deliveries WHERE (state='pending' OR (state='sending' AND lease_until<clock_timestamp())) AND available_at<=clock_timestamp() ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE deliveries d SET state='sending',lease_until=clock_timestamp()+interval '60 seconds',fence=fence+1 FROM next WHERE d.id=next.id RETURNING d.id::text,d.kind,d.owner_id::text,d.config_id::text,d.link_id::text,d.recipient,d.event_type,d.locale,d.payload,d.attempts,d.fence,d.system_email,d.state`).Scan(&id, &kind, &owner, &configID, &linkID, &to, &event, &locale, &payload, &attempts, &fence, &system, &state)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	deferState := func(code string, delay time.Duration) error {
		_, e := s.DB.Exec(ctx, "UPDATE deliveries SET state='pending',lease_until=NULL,last_code=$3,available_at=clock_timestamp()+make_interval(secs=>$4) WHERE id=$1 AND fence=$2", id, fence, code, delay.Seconds())
		return e
	}
	if kind == "smtp" && !system && owner != nil {
		var zone string
		var start, end *int
		var digest string
		e = s.DB.QueryRow(ctx, "SELECT timezone,quiet_start,quiet_end,digest FROM notification_preferences WHERE user_id=$1", *owner).Scan(&zone, &start, &end, &digest)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if digest == "off" {
			_, e = s.DB.Exec(ctx, "UPDATE deliveries SET state='cancelled',lease_until=NULL WHERE id=$1 AND fence=$2", id, fence)
			return e
		}
		if Quiet(time.Now(), zone, start, end) {
			return deferState("quiet_hours", 15*time.Minute)
		}
	}
	if kind == "smtp" {
		var smtpID string
		if e = s.DB.QueryRow(ctx, "SELECT id::text FROM communication_configs WHERE kind='smtp' AND enabled").Scan(&smtpID); errors.Is(e, pgx.ErrNoRows) {
			return deferState("smtp_unconfigured", time.Minute)
		}
		if e != nil {
			return e
		}
		var cfg SMTPConfig
		if e = s.Config(ctx, smtpID, &cfg); e != nil {
			return e
		}
		link := ""
		if linkID != nil {
			var box []byte
			var keyID string
			if e = s.DB.QueryRow(ctx, "SELECT l.token_envelope,l.key_id::text FROM recipient_links l JOIN transfers t ON t.id=l.transfer_id WHERE l.id=$1 AND l.revoked_at IS NULL AND t.state='active' AND t.expires_at>clock_timestamp() AND t.absolute_expires_at>clock_timestamp()", *linkID).Scan(&box, &keyID); errors.Is(e, pgx.ErrNoRows) {
				_, e = s.DB.Exec(ctx, "UPDATE deliveries SET state='cancelled',lease_until=NULL WHERE id=$1 AND fence=$2", id, fence)
				return e
			}
			if e != nil {
				return e
			}
			key, e := s.Keys.Get(ctx, keyID)
			if e != nil {
				return e
			}
			raw, e := cryptobox.Open(key, "filex/link/v1/"+*linkID, box)
			if e != nil {
				return e
			}
			link = c.PublicOrigin + "/receive#token=" + string(raw)
		}
		var p struct {
			Count int `json:"count"`
		}
		if e = json.Unmarshal(payload, &p); e != nil {
			return e
		}
		subject, message := Template(event, locale, link, p.Count)
		send := s.Mail
		if send == nil {
			send = SendMail
		}
		if to == nil {
			return errors.New("delivery lacks recipient")
		}
		e = send(ctx, cfg, id, *to, subject, message)
	} else {
		if configID == nil {
			return errors.New("delivery lacks configuration")
		}
		var cfg WebhookConfig
		e = s.Config(ctx, *configID, &cfg)
		if errors.Is(e, pgx.ErrNoRows) {
			_, e = s.DB.Exec(ctx, "UPDATE deliveries SET state='cancelled',lease_until=NULL WHERE id=$1 AND fence=$2", id, fence)
			return e
		}
		if e == nil {
			send := s.Hook
			if send == nil {
				send = SendWebhook
			}
			var event map[string]any
			if e = json.Unmarshal(payload, &event); e != nil {
				return e
			}
			event["eventId"] = id
			payload, e = json.Marshal(event)
			if e != nil {
				return e
			}
			e = send(ctx, cfg, id, payload)
		}
	}
	if e == nil {
		_, e = s.DB.Exec(ctx, "UPDATE deliveries SET state='delivered',delivered_at=clock_timestamp(),lease_until=NULL,attempts=attempts+1,last_code=NULL WHERE id=$1 AND fence=$2", id, fence)
		return e
	}
	attempts++
	state = "pending"
	if attempts >= 8 {
		state = "dead"
	}
	delay := min(3600, 5*(1<<min(attempts, 10)))
	_, e = s.DB.Exec(ctx, "UPDATE deliveries SET state=$3,attempts=$4,lease_until=NULL,last_code='delivery_failed',available_at=clock_timestamp()+make_interval(secs=>$5) WHERE id=$1 AND fence=$2", id, fence, state, attempts, delay)
	return e
}

package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type event struct {
	Actor   string `json:"actor"`
	Action  string `json:"action"`
	Subject string `json:"subject"`
	Time    string `json:"time"`
}

func Append(ctx context.Context, tx pgx.Tx, key []byte, actor, action, subject string) error {
	if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184995)"); e != nil {
		return e
	}
	var previous string
	e := tx.QueryRow(ctx, "SELECT hash FROM audit_events ORDER BY id DESC LIMIT 1").Scan(&previous)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return e
	}
	payload, e := json.Marshal(event{actor, action, subject, now.UTC().Format(time.RFC3339Nano)})
	if e != nil {
		return e
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("filex/audit/v1/" + previous))
	mac.Write(payload)
	hash := hex.EncodeToString(mac.Sum(nil))
	_, e = tx.Exec(ctx, "INSERT INTO audit_events(actor_id,action,subject_id,occurred_at,payload,previous_hash,hash) VALUES($1,$2,$3,$4,$5,$6,$7)", actor, action, subject, now, payload, previous, hash)
	return e
}
func Verify(ctx context.Context, p *pgxpool.Pool, key []byte) error {
	rows, e := p.Query(ctx, "SELECT actor_id,action,subject_id,occurred_at,payload,previous_hash,hash FROM audit_events ORDER BY id")
	if e != nil {
		return e
	}
	defer rows.Close()
	previous := ""
	for rows.Next() {
		var actor, action, subject, prev, hash string
		var at time.Time
		var payload []byte
		if e = rows.Scan(&actor, &action, &subject, &at, &payload, &prev, &hash); e != nil {
			return e
		}
		expected, _ := json.Marshal(event{actor, action, subject, at.UTC().Format(time.RFC3339Nano)})
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte("filex/audit/v1/" + previous))
		mac.Write(expected)
		if prev != previous || !hmac.Equal(expected, payload) || hex.EncodeToString(mac.Sum(nil)) != hash {
			return errors.New("audit integrity failed")
		}
		previous = hash
	}
	return rows.Err()
}

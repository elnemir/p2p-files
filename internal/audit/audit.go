package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	var payload []byte
	var signature, id, eventHash, evidenceHash, reason string
	var eventID int64
	e := p.QueryRow(ctx, "SELECT id::text,event_id,event_hash,evidence_hash,reason,payload,signature FROM audit_checkpoints WHERE kind='recovery' ORDER BY event_id DESC,created_at DESC,id DESC LIMIT 1").Scan(&id, &eventID, &eventHash, &evidenceHash, &reason, &payload, &signature)
	if errors.Is(e, pgx.ErrNoRows) {
		return VerifyFull(ctx, p, key)
	}
	if e != nil {
		return e
	}
	var anchor Checkpoint
	if e = json.Unmarshal(payload, &anchor); e != nil || !CheckSignature(key, payload, signature) || anchor.Kind != "recovery" || anchor.ID != id || anchor.EventID != eventID || anchor.EventHash != eventHash || anchor.EvidenceHash != evidenceHash || anchor.Reason != reason {
		return errors.New("invalid audit checkpoint")
	}
	var hash string
	if e = p.QueryRow(ctx, "SELECT hash FROM audit_events WHERE id=$1", anchor.EventID).Scan(&hash); e != nil || hash != anchor.EventHash {
		return errors.New("audit checkpoint boundary missing")
	}
	evidence, e := Evidence(ctx, p, "audit_events", anchor.EventID)
	if e != nil || evidence != anchor.EvidenceHash {
		return errors.New("audit checkpoint history changed")
	}
	return verifyAfter(ctx, p, key, anchor.EventID, anchor.EventHash)
}

type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// Evidence binds the exact preserved rows, including already damaged history.
// A negative boundary means the whole current journal; zero means no rows.
func Evidence(ctx context.Context, p Querier, table string, through int64) (string, error) {
	if table != "audit_events" && table != "ops_events" {
		return "", errors.New("invalid audit table")
	}
	rows, e := p.Query(ctx, "SELECT row_to_json(e)::text FROM "+table+" e WHERE $1<0 OR id<=$1 ORDER BY id", through)
	if e != nil {
		return "", e
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return "", e
		}
		h.Write([]byte(raw))
		h.Write([]byte{'\n'})
	}
	return fmt.Sprintf("%x", h.Sum(nil)), rows.Err()
}
func VerifyFull(ctx context.Context, p *pgxpool.Pool, key []byte) error {
	return verifyAfter(ctx, p, key, 0, "")
}

type Checkpoint struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	EventID      int64     `json:"eventId"`
	EventHash    string    `json:"eventHash"`
	EvidenceHash string    `json:"evidenceHash"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
}

func SignCheckpoint(key, payload []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("filex/audit/checkpoint/v1/"))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
func CheckSignature(key, payload []byte, signature string) bool {
	return hmac.Equal([]byte(SignCheckpoint(key, payload)), []byte(signature))
}
func verifyAfter(ctx context.Context, p *pgxpool.Pool, key []byte, after int64, previous string) error {
	rows, e := p.Query(ctx, "SELECT actor_id,action,subject_id,occurred_at,payload,previous_hash,hash FROM audit_events WHERE id>$1 ORDER BY id", after)
	if e != nil {
		return e
	}
	defer rows.Close()
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

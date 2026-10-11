package communications

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"time"
)

// AcceptWebhook is a receiver reference implementation. Signature, signed event ID,
// deduplication receipt and consumer mutation belong to one local transaction.
// Delivery is at least once: an already accepted event returns false, nil.
func (s Service) AcceptWebhook(ctx context.Context, secret, id string, at int64, payload []byte, signature string, apply func(pgx.Tx) error) (bool, error) {
	if len(payload) > 65536 || !domain.ValidID(id) || !VerifySignature(secret, at, payload, signature, time.Now()) {
		return false, errors.New("webhook authentication failed")
	}
	var event struct {
		ID      string `json:"eventId"`
		Version int    `json:"schemaVersion"`
	}
	if json.Unmarshal(payload, &event) != nil || event.ID != id || event.Version != 1 {
		return false, errors.New("invalid webhook event")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	tag, e := tx.Exec(ctx, "INSERT INTO webhook_receipts(event_id) VALUES($1) ON CONFLICT(event_id) DO NOTHING", id)
	if e != nil {
		return false, e
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if e = apply(tx); e != nil {
		return false, e
	}
	if e = tx.Commit(ctx); e != nil {
		return false, e
	}
	return true, nil
}

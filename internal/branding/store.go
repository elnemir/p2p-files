package branding

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
)

type Reader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Revision struct {
	Revision int64  `json:"revision"`
	Value    Config `json:"value"`
}
type Store struct {
	DB  *pgxpool.Pool
	Key []byte
}

func Current(ctx context.Context, db Reader) (Revision, error) {
	var rev Revision
	var raw []byte
	e := db.QueryRow(ctx, "SELECT revision,value FROM branding_revisions ORDER BY revision DESC LIMIT 1").Scan(&rev.Revision, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return Revision{Value: Default()}, nil
	}
	if e != nil {
		return rev, e
	}
	e = json.Unmarshal(raw, &rev.Value)
	return rev, e
}
func (s Store) Apply(ctx context.Context, p auth.Principal, expected int64, c Config) (Revision, error) {
	if expected < 0 {
		return Revision{}, ErrInvalid
	}
	if p.Bearer {
		return Revision{}, auth.ErrForbidden
	}
	c, e := Normalize(c)
	if e != nil {
		return Revision{}, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Revision{}, e
	}
	defer tx.Rollback(context.Background())
	// Serialize publication, and recheck current authority after validation work.
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185020)"); e != nil {
		return Revision{}, e
	}
	var actor string
	e = tx.QueryRow(ctx, `SELECT u.id::text FROM users u JOIN web_sessions s ON s.user_id=u.id WHERE u.id::text=$1 AND s.id::text=$2 AND u.role='admin' AND NOT u.disabled AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.reauth_at>clock_timestamp()-interval '5 minutes' AND s.auth_epoch=u.auth_epoch AND u.auth_epoch=$3 FOR SHARE OF u,s`, p.ID, p.SessionID, p.Epoch).Scan(&actor)
	if errors.Is(e, pgx.ErrNoRows) {
		return Revision{}, auth.ErrForbidden
	}
	if e != nil {
		return Revision{}, e
	}
	if _, e = licensing.Guard(ctx, tx, "white_label"); e != nil {
		return Revision{}, e
	}
	current, e := Current(ctx, tx)
	if e != nil {
		return Revision{}, e
	}
	if current.Revision != expected {
		return Revision{}, ErrConflict
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return Revision{}, e
	}
	result := Revision{Value: c}
	if e = tx.QueryRow(ctx, "INSERT INTO branding_revisions(created_by,value) VALUES($1,$2) RETURNING revision", actor, raw).Scan(&result.Revision); e != nil {
		return Revision{}, e
	}
	if e = audit.Append(ctx, tx, s.Key, actor, "branding.applied", strconv.FormatInt(result.Revision, 10)); e != nil {
		return Revision{}, e
	}
	return result, tx.Commit(ctx)
}

// Snapshot binds presentation to the same transaction as the report revision.
func Snapshot(ctx context.Context, db Reader) ([]byte, error) {
	r, e := Current(ctx, db)
	if e != nil {
		return nil, e
	}
	return json.Marshal(r.Value)
}
func DecodeSnapshot(raw []byte) (Config, error) {
	if len(raw) == 0 {
		return Default(), nil
	}
	var c Config
	e := json.Unmarshal(raw, &c)
	return c, e
}

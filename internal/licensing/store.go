package licensing

import (
	"context"
	"crypto/sha256"
	"filex.local/filex/internal/audit"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var started = time.Now()

type Status struct {
	InstallationID string  `json:"installationId"`
	Epoch          int64   `json:"epoch"`
	State          string  `json:"state"`
	Claims         *Claims `json:"claims"`
	Pending        *Claims `json:"pending"`
	EnabledUsers   int64   `json:"enabledUsers"`
	OverLimit      bool    `json:"overLimit"`
}
type Store struct {
	DB       *pgxpool.Pool
	AuditKey []byte
}

// Every grant holds the same license row through the domain commit. Import
// cannot revoke an epoch while a quota reservation is being committed.
func snapshot(ctx context.Context, tx pgx.Tx) (Status, error) {
	var s Status
	var raw, pending string
	var observed, wall time.Time
	e := tx.QueryRow(ctx, `SELECT installation_id::text,epoch,envelope,pending_envelope,max_observed_at,clock_timestamp() FROM ops_license WHERE singleton FOR UPDATE`).Scan(&s.InstallationID, &s.Epoch, &raw, &pending, &observed, &wall)
	if e != nil {
		return s, e
	}
	now := wall
	floor := started.Add(time.Since(started))
	if floor.After(now) {
		now = floor
	}
	if observed.After(now) {
		now = observed
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_license SET max_observed_at=GREATEST(max_observed_at,$1) WHERE singleton", now); e != nil {
		return s, e
	}
	s.State = "missing"
	keys, e := ReleaseKeys()
	if e != nil {
		return s, e
	}
	if pending != "" {
		c, err := Verify(pending, keys)
		if err != nil || c.InstallationID != s.InstallationID {
			s.State = "invalid"
			return s, nil
		}
		s.Pending = &c
		if !now.Before(c.NotBefore) {
			raw = pending
			if _, e = tx.Exec(ctx, "UPDATE ops_license SET envelope=pending_envelope,pending_envelope='',epoch=epoch+1 WHERE singleton"); e != nil {
				return s, e
			}
			s.Epoch++
			s.Pending = nil
		}
	}
	if raw != "" {
		c, err := Verify(raw, keys)
		if err != nil || c.InstallationID != s.InstallationID {
			s.State = "invalid"
		} else {
			s.Claims = &c
			s.State = c.State(now)
			if !UpdateEntitled(c) {
				s.State = "update_not_entitled"
			}
		}
	}
	if wall.Before(observed.Add(-5*time.Minute)) || wall.Before(floor.Add(-5*time.Minute)) {
		s.State = "clock_suspect"
	}
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM users WHERE NOT disabled").Scan(&s.EnabledUsers); e != nil {
		return s, e
	}
	s.OverLimit = s.Claims != nil && s.Claims.Limits.Users != nil && s.EnabledUsers > *s.Claims.Limits.Users
	return s, nil
}
func (s Store) Status(ctx context.Context) (Status, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Status{}, e
	}
	defer tx.Rollback(context.Background())
	v, e := snapshot(ctx, tx)
	if e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}
func Guard(ctx context.Context, tx pgx.Tx, feature string) (Claims, error) {
	v, e := snapshot(ctx, tx)
	if e != nil {
		return Claims{}, e
	}
	if v.State != "active" || v.Claims == nil || !v.Claims.Allows(feature) {
		return Claims{}, ErrDenied
	}
	return *v.Claims, nil
}
func UserSlot(ctx context.Context, tx pgx.Tx, bootstrap bool) error {
	v, e := snapshot(ctx, tx)
	if e != nil {
		return e
	}
	if bootstrap {
		var total int64
		if e = tx.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&total); e != nil {
			return e
		}
		if total == 0 {
			return nil
		}
	}
	if v.State != "active" || v.Claims == nil || v.Claims.Limits.Users != nil && v.EnabledUsers >= *v.Claims.Limits.Users {
		return ErrDenied
	}
	return nil
}
func TransferQuota(ctx context.Context, tx pgx.Tx, id, path string, bytes int64) error {
	c, e := Guard(ctx, tx, "")
	if e != nil {
		return e
	}
	if bytes < 0 {
		return ErrInvalid
	}
	if c.Limits.Bytes == nil {
		return nil
	}
	// Candidates replacing one logical path reserve its maximum, not both copies.
	var used int64
	e = tx.QueryRow(ctx, `SELECT COALESCE(sum(n),0)::bigint FROM (SELECT license_path_key,max(license_bytes) n FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted') AND license_path_key<>$2 GROUP BY license_path_key) q`, id, path).Scan(&used)
	if e != nil {
		return e
	}
	var current int64
	if path != "" {
		e = tx.QueryRow(ctx, `SELECT COALESCE(max(license_bytes),0) FROM files WHERE transfer_id=$1 AND license_path_key=$2 AND state NOT IN ('cancelled','deleted')`, id, path).Scan(&current)
		if e != nil {
			return e
		}
	}
	if used > *c.Limits.Bytes || max(bytes, current) > *c.Limits.Bytes-used {
		return ErrDenied
	}
	return nil
}
func AppendQuota(ctx context.Context, tx pgx.Tx, file string, next int64) error {
	var transfer, path string
	var reserved int64
	if e := tx.QueryRow(ctx, "SELECT transfer_id::text,license_path_key,license_bytes FROM files WHERE id=$1", file).Scan(&transfer, &path, &reserved); e != nil {
		return e
	}
	if e := TransferQuota(ctx, tx, transfer, path, max(reserved, next)); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, "UPDATE files SET license_bytes=GREATEST(license_bytes,$2) WHERE id=$1", file, next)
	return e
}

type streamingKey struct{}

func WithStreaming(ctx context.Context) context.Context {
	return context.WithValue(ctx, streamingKey{}, true)
}
func ReservedBytes(ctx context.Context, size int64) int64 {
	if yes, _ := ctx.Value(streamingKey{}).(bool); yes {
		return 0
	}
	return size
}
func (s Store) Preview(ctx context.Context, raw string) (Claims, error) {
	keys, e := ReleaseKeys()
	if e != nil {
		return Claims{}, e
	}
	c, e := Verify(raw, keys)
	if e != nil {
		return c, e
	}
	v, e := s.Status(ctx)
	if e != nil {
		return c, e
	}
	if c.InstallationID != v.InstallationID {
		return c, ErrInvalid
	}
	return c, nil
}
func (s Store) Apply(ctx context.Context, raw string, epoch int64, actor, session string, authEpoch int64) (Status, error) {
	keys, e := ReleaseKeys()
	if e != nil {
		return Status{}, e
	}
	c, e := Verify(raw, keys)
	if e != nil {
		return Status{}, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Status{}, e
	}
	defer tx.Rollback(context.Background())
	if actor != "host_cli" {
		var ok bool
		e = tx.QueryRow(ctx, `SELECT true FROM users u JOIN web_sessions s ON s.user_id=u.id WHERE u.id::text=$1 AND s.id::text=$2 AND u.role='admin' AND NOT u.disabled AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp() AND s.reauth_at>clock_timestamp()-interval '5 minutes' AND s.auth_epoch=u.auth_epoch AND u.auth_epoch=$3 FOR SHARE OF u,s`, actor, session, authEpoch).Scan(&ok)
		if e != nil {
			return Status{}, e
		}
		if !ok {
			return Status{}, ErrDenied
		}
	}
	v, e := snapshot(ctx, tx)
	if e != nil {
		return v, e
	}
	if v.InstallationID != c.InstallationID {
		return v, ErrInvalid
	}
	if epoch != v.Epoch {
		return v, ErrConflict
	}
	var entitlement, current, pending string
	var revision int64
	var observed time.Time
	e = tx.QueryRow(ctx, "SELECT entitlement_id,high_revision,envelope,pending_envelope,max_observed_at FROM ops_license WHERE singleton").Scan(&entitlement, &revision, &current, &pending, &observed)
	if e != nil {
		return v, e
	}
	if raw == current || raw == pending {
		return v, tx.Commit(ctx)
	}
	if entitlement != "" && entitlement != c.EntitlementID || c.Revision <= revision {
		return v, ErrConflict
	}
	if c.State(observed) == "expired" {
		return v, ErrInvalid
	}
	if c.NotBefore.After(observed) {
		pending = raw
	} else {
		current = raw
		pending = ""
	}
	_, e = tx.Exec(ctx, "UPDATE ops_license SET envelope=$1,pending_envelope=$2,entitlement_id=$3,high_revision=$4,epoch=epoch+1 WHERE singleton", current, pending, c.EntitlementID, c.Revision)
	if e != nil {
		return v, e
	}
	digest := sha256.Sum256([]byte(raw))
	if e = audit.Append(ctx, tx, s.AuditKey, actor, "license.applied", fmt.Sprintf("%s:%d:%x", c.EntitlementID, c.Revision, digest)); e != nil {
		return v, e
	}
	v, e = snapshot(ctx, tx)
	if e != nil {
		return v, e
	}
	return v, tx.Commit(ctx)
}

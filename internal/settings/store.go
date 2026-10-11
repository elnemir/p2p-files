package settings

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"unicode/utf8"
)

type Store struct {
	DB  *pgxpool.Pool
	Key []byte
}
type Revision struct {
	Revision  int64           `json:"revision"`
	Active    bool            `json:"active"`
	Validated bool            `json:"validated"`
	Value     json.RawMessage `json:"value"`
}

func (s Store) Draft(ctx context.Context, actor string, raw []byte) (int64, error) {
	if _, e := config.ParseSettings(raw); e != nil {
		return 0, e
	}
	var id int64
	e := s.DB.QueryRow(ctx, "INSERT INTO app_settings(created_by,value) VALUES(NULLIF($1,'host_cli')::uuid,$2) RETURNING revision", actor, raw).Scan(&id)
	return id, e
}
func (s Store) Validate(ctx context.Context, id int64) error {
	var raw []byte
	if e := s.DB.QueryRow(ctx, "SELECT value FROM app_settings WHERE revision=$1", id).Scan(&raw); e != nil {
		return e
	}
	if _, e := config.ParseSettings(raw); e != nil {
		return e
	}
	_, e := s.DB.Exec(ctx, "UPDATE app_settings SET validated=true WHERE revision=$1", id)
	return e
}
func (s Store) Apply(ctx context.Context, actor string, id, expected int64) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184996)"); e != nil {
		return e
	}
	var active int64
	var previous []byte
	if e = tx.QueryRow(ctx, "SELECT revision,value FROM app_settings WHERE active FOR UPDATE").Scan(&active, &previous); e != nil {
		return e
	}
	if active != expected {
		return errors.New("settings revision conflict")
	}
	var raw []byte
	var validated bool
	if e = tx.QueryRow(ctx, "SELECT value,validated FROM app_settings WHERE revision=$1 FOR UPDATE", id).Scan(&raw, &validated); e != nil {
		return e
	}
	if !validated {
		return errors.New("settings draft not validated")
	}
	next, e := config.ParseSettings(raw)
	if e != nil {
		return e
	}
	prior, e := config.ParseSettings(previous)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=false WHERE active"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=true WHERE revision=$1", id); e != nil {
		return e
	}
	if !prior.NetworkPolicy.Equal(next.NetworkPolicy) || prior.PublicOrigin != next.PublicOrigin {
		if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='open' THEN 'maintenance' ELSE mode END,reason='Network policy changed',actor=$1,reconciliation_id=NULL,generation=gen_random_uuid(),changed_at=clock_timestamp() WHERE singleton", actor); e != nil {
			return e
		}
	}
	if e = audit.Append(ctx, tx, s.Key, actor, "settings.applied", "settings"); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s Store) Active(ctx context.Context) (Revision, error) {
	var r Revision
	e := s.DB.QueryRow(ctx, "SELECT revision,active,validated,value FROM app_settings WHERE active").Scan(&r.Revision, &r.Active, &r.Validated, &r.Value)
	return r, e
}

// ResetNetworkAllowlist preserves transport trust while repairing operator lockout.
func (s Store) ResetNetworkAllowlist(ctx context.Context, reason string) (int64, error) {
	if utf8.RuneCountInString(strings.TrimSpace(reason)) < 5 || utf8.RuneCountInString(reason) > 2000 {
		return 0, errors.New("invalid network reset reason")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184996)"); e != nil {
		return 0, e
	}
	var raw []byte
	if e = tx.QueryRow(ctx, "SELECT value FROM app_settings WHERE active FOR UPDATE").Scan(&raw); e != nil {
		return 0, e
	}
	value, e := config.ParseSettings(raw)
	if e != nil {
		return 0, e
	}
	value.NetworkPolicy.AllowedClientCIDRs = nil
	raw, e = json.Marshal(value)
	if e != nil {
		return 0, e
	}
	var revision int64
	if e = tx.QueryRow(ctx, "INSERT INTO app_settings(value,validated) VALUES($1,true) RETURNING revision", raw).Scan(&revision); e != nil {
		return 0, e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=false WHERE active"); e != nil {
		return 0, e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=true WHERE revision=$1", revision); e != nil {
		return 0, e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='open' THEN 'maintenance' ELSE mode END,reason=$1,actor='host_cli',reconciliation_id=NULL,generation=gen_random_uuid(),changed_at=clock_timestamp() WHERE singleton", reason); e != nil {
		return 0, e
	}
	if e = audit.Append(ctx, tx, s.Key, "host_cli", "network.allowlist_reset", reason); e != nil {
		return 0, e
	}
	return revision, tx.Commit(ctx)
}
func (s Store) List(ctx context.Context) ([]Revision, error) {
	rows, e := s.DB.Query(ctx, "SELECT revision,active,validated,value FROM app_settings ORDER BY revision DESC LIMIT 100")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Revision{}
	for rows.Next() {
		var r Revision
		if e = rows.Scan(&r.Revision, &r.Active, &r.Validated, &r.Value); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

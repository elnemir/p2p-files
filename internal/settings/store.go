package settings

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
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
	e := s.DB.QueryRow(ctx, "INSERT INTO app_settings(created_by,value) VALUES($1,$2) RETURNING revision", actor, raw).Scan(&id)
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
	if e = tx.QueryRow(ctx, "SELECT revision FROM app_settings WHERE active FOR UPDATE").Scan(&active); e != nil {
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
	if _, e = config.ParseSettings(raw); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=false WHERE active"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE app_settings SET active=true WHERE revision=$1", id); e != nil {
		return e
	}
	if e = audit.Append(ctx, tx, s.Key, actor, "settings.applied", "settings"); e != nil {
		return e
	}
	return tx.Commit(ctx)
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

package operations

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/domain"
	"time"
)

type Schedule struct {
	Kind     string    `json:"kind"`
	Interval int       `json:"intervalSeconds"`
	Next     time.Time `json:"nextAt"`
	Provider *string   `json:"providerId"`
}

func (s Service) Schedules(ctx context.Context) ([]Schedule, error) {
	rows, e := s.DB.Query(ctx, "SELECT kind,interval_seconds,next_at,provider_id::text FROM ops_schedules ORDER BY kind")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Schedule{}
	for rows.Next() {
		var v Schedule
		if e = rows.Scan(&v.Kind, &v.Interval, &v.Next, &v.Provider); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Service) SetSchedule(ctx context.Context, actor, kind string, interval int, provider string) error {
	if (kind != "backup" && kind != "audit" && kind != "storage") || interval < 0 || interval > 2592000 || (interval > 0 && interval < 60) {
		return ErrInvalid
	}
	if kind == "backup" && interval > 0 {
		var valid bool
		if !domain.ValidID(provider) {
			return ErrInvalid
		}
		if e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM storage_providers WHERE id=$1 AND purpose='backups' AND enabled AND NOT draining)", provider).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return ErrInvalid
		}
	} else {
		provider = ""
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "UPDATE ops_schedules SET interval_seconds=$2::int,next_at=clock_timestamp()+make_interval(secs=>$2::int),provider_id=NULLIF($3,'')::uuid WHERE kind=$1", kind, interval, provider); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "schedule.updated", kind, "interval configured"); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) Schedule(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var mode string
	if e = tx.QueryRow(ctx, "SELECT mode FROM ops_control WHERE singleton FOR SHARE").Scan(&mode); e != nil {
		return e
	}
	if mode != "open" {
		return nil
	}
	rows, e := tx.Query(ctx, "SELECT kind,interval_seconds,next_at,provider_id::text FROM ops_schedules WHERE interval_seconds>0 AND next_at<=clock_timestamp() ORDER BY kind FOR UPDATE SKIP LOCKED")
	if e != nil {
		return e
	}
	due := []Schedule{}
	for rows.Next() {
		var v Schedule
		if e = rows.Scan(&v.Kind, &v.Interval, &v.Next, &v.Provider); e != nil {
			rows.Close()
			return e
		}
		due = append(due, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range due {
		var pending bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_jobs WHERE kind=$1 AND state IN ('queued','running'))", v.Kind).Scan(&pending); e != nil {
			return e
		}
		if pending {
			continue
		}
		id := domain.NewID()
		var box []byte
		if v.Kind == "backup" {
			if v.Provider == nil {
				return ErrInvalid
			}
			var valid bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM storage_providers WHERE id=$1 AND purpose='backups' AND enabled AND NOT draining)", *v.Provider).Scan(&valid); e != nil {
				return e
			}
			if !valid {
				continue
			}
			d, e := s.Storage.Describe(ctx, *v.Provider)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(d)
			if e != nil {
				return e
			}
			box, e = s.seal("filex/ops_provider/v1/"+id, raw)
			if e != nil {
				return e
			}
		}
		if _, e = tx.Exec(ctx, "INSERT INTO ops_jobs(id,kind,actor,reason,provider_id,provider_envelope) VALUES($1,$2,'scheduler','Scheduled operational check',$3,$4)", id, v.Kind, v.Provider, box); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE ops_schedules SET next_at=clock_timestamp()+make_interval(secs=>interval_seconds) WHERE kind=$1", v.Kind); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

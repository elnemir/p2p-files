package operations

import (
	"context"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/maintenance"
)

type PreparedUpgrade struct {
	Generation      string `json:"generation"`
	Mode            string `json:"mode"`
	Drained         bool   `json:"drained"`
	RequiresRestart bool   `json:"requiresRestart"`
}

// PrepareUpgrade fences the old runtime before waiting for external effects.
// The caller must stop all instances; this is not a persistent deployment lock.
func (s Service) PrepareUpgrade(ctx context.Context, actor, reason string) (PreparedUpgrade, error) {
	out := PreparedUpgrade{}
	if !validReason(reason) {
		return out, ErrInvalid
	}
	unlock, e := s.runnerLock(ctx)
	if e != nil {
		return out, e
	}
	defer unlock()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(context.Background())
	var generation string
	if e = tx.QueryRow(ctx, "SELECT generation::text FROM ops_control WHERE singleton FOR UPDATE").Scan(&generation); e != nil {
		return out, e
	}
	if generation != s.Generation {
		return out, maintenance.ErrRestart
	}
	var pending bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_jobs WHERE state IN ('queued','running'))").Scan(&pending); e != nil {
		return out, e
	}
	if pending {
		return out, ErrConflict
	}
	out.Generation = domain.NewID()
	if e = tx.QueryRow(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='open' THEN 'maintenance' ELSE mode END,generation=$1,reason=$2,actor=$3,reconciliation_id=NULL,changed_at=clock_timestamp() WHERE singleton RETURNING mode", out.Generation, reason, actor).Scan(&out.Mode); e != nil {
		return out, e
	}
	if e = s.event(ctx, tx, actor, "upgrade.preparing", out.Generation, reason); e != nil {
		return out, e
	}
	if e = tx.Commit(ctx); e != nil {
		return out, e
	}
	// Never roll back the closed gate when cancellation or lock timeout occurs.
	drain, e := maintenance.Exclusive(ctx, s.DB)
	if e != nil {
		return out, e
	}
	defer drain()
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(context.Background())
	if e = tx.QueryRow(ctx, "SELECT generation::text,mode FROM ops_control WHERE singleton FOR UPDATE").Scan(&generation, &out.Mode); e != nil {
		return out, e
	}
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_jobs WHERE state IN ('queued','running'))").Scan(&pending); e != nil {
		return out, e
	}
	if generation != out.Generation || out.Mode == "open" || pending {
		return out, ErrConflict
	}
	if e = s.event(ctx, tx, actor, "upgrade.prepared", out.Generation, reason); e != nil {
		return out, e
	}
	if e = tx.Commit(ctx); e != nil {
		return out, e
	}
	out.Drained, out.RequiresRestart = true, true
	return out, nil
}

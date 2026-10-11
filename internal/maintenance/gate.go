// Package maintenance coordinates live handlers with destructive recovery.
package maintenance

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"sync"
	"time"
)

var ErrClosed = errors.New("maintenance gate closed")
var ErrRestart = errors.New("runtime restart required")

const lockSQL = "hashtextextended(current_schema()||'/filex/runtime/v1',0)"

type State struct {
	Mode             string    `json:"mode"`
	Generation       string    `json:"generation"`
	Reason           string    `json:"reason"`
	RestoreID        *string   `json:"restoreId"`
	ReconciliationID *string   `json:"reconciliationId"`
	ChangedAt        time.Time `json:"changedAt"`
}

// Gate has a separate pool so a streaming handler cannot starve its own SQL.
type Gate struct {
	pool       *pgxpool.Pool
	Generation string
}

func Read(ctx context.Context, p *pgxpool.Pool) (State, error) {
	var s State
	e := p.QueryRow(ctx, "SELECT mode,generation::text,reason,restore_id::text,reconciliation_id::text,changed_at FROM ops_control WHERE singleton").Scan(&s.Mode, &s.Generation, &s.Reason, &s.RestoreID, &s.ReconciliationID, &s.ChangedAt)
	return s, e
}
func New(ctx context.Context, db *pgxpool.Pool) (*Gate, error) {
	s, e := Read(ctx, db)
	if e != nil {
		return nil, e
	}
	c := db.Config().Copy()
	c.MaxConns = 24
	c.MinConns = 0
	p, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, e
	}
	return &Gate{pool: p, Generation: s.Generation}, nil
}
func (g *Gate) Close() { g.pool.Close() }

// StartupShared keeps the key snapshot and runtime generation from straddling
// a restore commit. It does not require an open data plane.
func StartupShared(ctx context.Context, db *pgxpool.Pool) (func(), error) {
	c, e := db.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = c.Exec(ctx, "SELECT pg_advisory_lock_shared("+lockSQL+")"); e != nil {
		c.Release()
		return nil, e
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, e := c.Exec(cleanup, "SELECT pg_advisory_unlock_shared("+lockSQL+")"); e != nil {
			_ = c.Conn().Close(cleanup)
		}
		c.Release()
	}, nil
}
func (g *Gate) Acquire(ctx context.Context, control bool) (context.Context, func(), error) {
	c, e := g.pool.Acquire(ctx)
	if e != nil {
		return ctx, nil, e
	}
	var locked bool
	if e = c.QueryRow(ctx, "SELECT pg_try_advisory_lock_shared("+lockSQL+")").Scan(&locked); e != nil || !locked {
		c.Release()
		if e != nil {
			return ctx, nil, e
		}
		return ctx, nil, ErrClosed
	}
	unlock := func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var ok bool
		if err := c.QueryRow(cleanup, "SELECT pg_advisory_unlock_shared("+lockSQL+")").Scan(&ok); err != nil || !ok {
			_ = c.Conn().Close(cleanup)
		}
		c.Release()
	}
	check := func() error {
		var mode, generation string
		if e := c.QueryRow(ctx, "SELECT mode,generation::text FROM ops_control WHERE singleton").Scan(&mode, &generation); e != nil {
			return e
		}
		if generation != g.Generation {
			return ErrRestart
		}
		if mode != "open" && !control {
			return ErrClosed
		}
		return nil
	}
	if e = check(); e != nil {
		unlock()
		return ctx, nil, e
	}
	work, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(done)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-work.Done():
				return
			case <-ticker.C:
				if check() != nil {
					cancel()
					return
				}
			}
		}
	}()
	return work, func() { once.Do(func() { cancel(); <-done; unlock() }) }, nil
}

// Exclusive is called only after the durable gate has been closed. Session
// locks also fence external storage effects until every old handler has left.
func Exclusive(ctx context.Context, db *pgxpool.Pool) (func(), error) {
	c, e := db.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	if _, e = c.Exec(ctx, "SELECT pg_advisory_lock("+lockSQL+")"); e != nil {
		c.Release()
		return nil, e
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var ok bool
		if e := c.QueryRow(cleanup, "SELECT pg_advisory_unlock("+lockSQL+")").Scan(&ok); e != nil || !ok {
			_ = c.Conn().Close(cleanup)
		}
		c.Release()
	}, nil
}

// ExclusiveTransaction fences pending DDL through commit or rollback. Callers
// must not hold a shared runtime lock while upgrading to this exclusive lock.
func ExclusiveTransaction(ctx context.Context, tx pgx.Tx) error {
	_, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock("+lockSQL+")")
	return e
}

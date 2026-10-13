package worker

import (
	"context"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/exports"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/operations"
	"filex.local/filex/internal/reports"
	"filex.local/filex/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	DB         *pgxpool.Pool
	Keys       keyring.Ring
	Files      files.Service
	AuditKey   []byte
	Gate       *maintenance.Gate
	Operations operations.Service
}

func New(db *pgxpool.Pool, kek []byte) (Worker, error) {
	unlock, e := maintenance.StartupShared(context.Background(), db)
	if e != nil {
		return Worker{}, e
	}
	defer unlock()
	k := keyring.Ring{DB: db, KEK: kek}
	auditKey, e := database.AuditKey(context.Background(), db, kek)
	if e != nil {
		return Worker{}, e
	}
	ops, e := operations.New(context.Background(), db, kek)
	if e != nil {
		return Worker{}, e
	}
	gate, e := maintenance.New(context.Background(), db)
	if e != nil {
		return Worker{}, e
	}
	return Worker{Gate: gate, Operations: ops, DB: db, Keys: k, Files: files.Service{DB: db, Keys: k, Storage: storage.Registry{DB: db, Keys: k}}, AuditKey: auditKey}, nil
}
func (w Worker) Close() { w.Gate.Close() }
func (w Worker) Tasks() map[string]func(context.Context) error {
	artifact := artifacts.Store{DB: w.DB, Keys: w.Keys, Storage: w.Files.Storage}
	report := reports.Service{DB: w.DB, Keys: w.Keys, Artifacts: artifact, AuditKey: w.AuditKey}
	export := exports.Service{DB: w.DB, Artifacts: artifact}
	tasks := map[string]func(context.Context) error{
		"cleanup": func(ctx context.Context) error {
			_, identityErr := w.DB.Exec(ctx, "DELETE FROM identity_flows WHERE expires_at<=clock_timestamp()")
			return errors.Join(identityErr, w.Files.Sweep(ctx), w.Files.Cleanup(ctx), artifact.Cleanup(ctx), export.Cleanup(ctx))
		},
		"maintenance": w.Files.Maintain,
		"rotation":    w.Rotate,
		"bulk":        w.Bulk,
		"reports":     report.RenderPending,
		"exports":     export.Run,
		"communications": func(ctx context.Context) error {
			c, e := database.Settings(ctx, w.DB)
			if e != nil {
				return e
			}
			s := communications.Service{DB: w.DB, Keys: w.Keys}
			return errors.Join(s.Fanout(ctx), s.Schedule(ctx), s.Deliver(ctx, c))
		},
	}
	for name, task := range tasks {
		tasks[name] = func(ctx context.Context) error {
			work, release, e := w.Gate.Acquire(ctx, false)
			if errors.Is(e, maintenance.ErrClosed) {
				return nil
			}
			if e != nil {
				return e
			}
			defer release()
			return task(work)
		}
	}
	tasks["operations"] = func(ctx context.Context) error {
		if e := w.Operations.Schedule(ctx); e != nil {
			return e
		}
		return w.Operations.Run(ctx)
	}
	return tasks
}
func (w Worker) Step(ctx context.Context) error {
	var errs []error
	for _, task := range w.Tasks() {
		errs = append(errs, task(ctx))
	}
	return errors.Join(errs...)
}

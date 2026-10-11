package worker

import (
	"context"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/communications"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/exports"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/reports"
	"filex.local/filex/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	DB       *pgxpool.Pool
	Keys     keyring.Ring
	Files    files.Service
	AuditKey []byte
}

func New(db *pgxpool.Pool, kek []byte) (Worker, error) {
	k := keyring.Ring{DB: db, KEK: kek}
	auditKey, e := hkdf.Key(sha256.New, kek, nil, "filex/audit/key/v1", 32)
	return Worker{DB: db, Keys: k, Files: files.Service{DB: db, Keys: k, Storage: storage.Registry{DB: db, Keys: k}}, AuditKey: auditKey}, e
}
func (w Worker) Tasks() map[string]func(context.Context) error {
	artifact := artifacts.Store{DB: w.DB, Keys: w.Keys, Storage: w.Files.Storage}
	report := reports.Service{DB: w.DB, Keys: w.Keys, Artifacts: artifact, AuditKey: w.AuditKey}
	export := exports.Service{DB: w.DB, Artifacts: artifact}
	return map[string]func(context.Context) error{
		"cleanup": func(ctx context.Context) error {
			return errors.Join(w.Files.Sweep(ctx), w.Files.Cleanup(ctx), artifact.Cleanup(ctx), export.Cleanup(ctx))
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
}
func (w Worker) Step(ctx context.Context) error {
	var errs []error
	for _, task := range w.Tasks() {
		errs = append(errs, task(ctx))
	}
	return errors.Join(errs...)
}

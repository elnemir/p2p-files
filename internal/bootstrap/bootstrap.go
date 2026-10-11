package bootstrap

import (
	"context"
	"errors"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"time"
)

func Open(ctx context.Context) (config.Bootstrap, *pgxpool.Pool, error) {
	c, err := config.Load(os.Getenv)
	if err != nil {
		return c, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, err := database.Connect(ctx, c.DatabaseURL)
	if err != nil {
		return c, nil, err
	}
	if err = database.Migrate(ctx, p); err != nil {
		p.Close()
		return c, nil, errors.New("database migration failed")
	}
	unlock, err := maintenance.StartupShared(ctx, p)
	if err != nil {
		p.Close()
		return c, nil, err
	}
	success := false
	defer func() {
		unlock()
		if !success {
			p.Close()
		}
	}()
	if err = database.CheckKEK(ctx, p, c.MasterKEK); err != nil {
		return c, nil, errors.New("critical key check failed")
	}
	if _, err = database.Settings(ctx, p); err != nil {
		return c, nil, errors.New("active settings invalid")
	}
	if c.Environment == "development" {
		for env, purpose := range map[string]string{"DEV_PAYLOAD_ROOT": "payload", "DEV_ARTIFACT_ROOT": "artifacts", "DEV_BACKUP_ROOT": "backups"} {
			if root := os.Getenv(env); root != "" {
				if _, err = (storage.Registry{DB: p}).Provision(ctx, root, purpose); err != nil {
					return c, nil, errors.New("development storage provisioning failed")
				}
			}
		}
	}
	success = true
	return c, p, nil
}

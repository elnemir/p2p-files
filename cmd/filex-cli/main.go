package main

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/httpapi"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/storage"
	"fmt"
	"io"
	"os"
	"slices"
	"time"
)

var jsonOutput bool

func emitLegacy(value any, text string) error {
	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(value)
	}
	fmt.Println(text)
	return nil
}
func run() error {
	jsonOutput = slices.Contains(os.Args, "--json")
	// Operational commands return JSON by default; accept the explicit flag too.
	os.Args = slices.DeleteFunc(os.Args, func(v string) bool { return v == "--json" })
	if handled, e := upgradeCLI(os.Args[1:]); handled {
		return e
	}
	if handled, e := settingsCLI(os.Args[1:]); handled {
		return e
	}
	if handled, e := operationalCLI(os.Args[1:]); handled {
		return e
	}
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		return emitLegacy(map[string]string{"version": "0.1.0-dev"}, "filex-cli 0.1.0-dev")
	}
	if len(os.Args) == 3 && os.Args[1] == "config" && os.Args[2] == "validate" {
		_, err := config.Load(os.Getenv)
		if err == nil {
			return emitLegacy(map[string]bool{"valid": true}, "Bootstrap configuration valid.")
		}
		return err
	}
	if len(os.Args) == 3 && os.Args[1] == "storage" && os.Args[2] == "s3" {
		var in struct {
			Purpose  string           `json:"purpose"`
			Priority int              `json:"priority"`
			S3       storage.S3Config `json:"s3"`
		}
		decoder := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		decoder.DisallowUnknownFields()
		if e := decoder.Decode(&in); e != nil {
			return fmt.Errorf("invalid storage JSON on stdin")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return fmt.Errorf("trailing storage input")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		c, db, e := bootstrap.Open(ctx)
		if e != nil {
			return e
		}
		defer db.Close()
		unlock, e := maintenance.StartupShared(ctx, db)
		if e != nil {
			return e
		}
		defer unlock()
		auditKey, e := database.AuditKey(ctx, db, c.MasterKEK)
		if e != nil {
			return fmt.Errorf("storage bootstrap failed")
		}
		registry := storage.Registry{DB: db, Keys: keyring.Ring{DB: db, KEK: c.MasterKEK}}
		id, e := registry.Register(ctx, storage.Descriptor{Kind: "s3", S3: &in.S3}, in.Purpose, in.Priority, auditKey, "host_cli")
		if e != nil {
			return fmt.Errorf("S3 provisioning failed")
		}
		return emitLegacy(map[string]string{"id": id}, id)
	}
	if len(os.Args) == 3 && os.Args[1] == "user" && os.Args[2] == "create" {
		var v struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		d := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		d.DisallowUnknownFields()
		if err := d.Decode(&v); err != nil {
			return fmt.Errorf("invalid user JSON on stdin")
		}
		if d.Decode(new(any)) != io.EOF {
			return fmt.Errorf("trailing user input")
		}
		c, p, err := bootstrap.Open(context.Background())
		if err != nil {
			return err
		}
		defer p.Close()
		unlock, err := maintenance.StartupShared(context.Background(), p)
		if err != nil {
			return err
		}
		defer unlock()
		app, err := httpapi.New(p, c.MasterKEK)
		if err != nil {
			return err
		}
		defer app.Close()
		id, err := app.Auth.CreateUser(context.Background(), "host_cli", v.Email, v.Password, v.Role)
		if err != nil {
			return fmt.Errorf("user provisioning failed")
		}
		return emitLegacy(map[string]string{"id": id}, id)
	}
	if len(os.Args) == 4 && os.Args[1] == "storage" && (os.Args[2] == "local" || os.Args[2] == "backup-local") {
		_, p, e := bootstrap.Open(context.Background())
		if e != nil {
			return e
		}
		defer p.Close()
		unlock, e := maintenance.StartupShared(context.Background(), p)
		if e != nil {
			return e
		}
		defer unlock()
		purpose := "payload"
		if os.Args[2] == "backup-local" {
			purpose = "backups"
		}
		id, e := (storage.Registry{DB: p}).Provision(context.Background(), os.Args[3], purpose)
		if e != nil {
			return fmt.Errorf("local storage provisioning failed")
		}
		return emitLegacy(map[string]string{"id": id}, id)
	}
	if len(os.Args) == 2 && os.Args[1] == "migrate" {
		_, p, err := bootstrap.Open(context.Background())
		if err != nil {
			return err
		}
		p.Close()
		return emitLegacy(map[string]string{"status": "verified"}, "Migrations, key check and settings verified.")
	}
	return fmt.Errorf("supported commands: --version, config validate|ingress-check, settings show|apply, network reset, upgrade check|prepare, migrate, user create, storage local|backup-local|s3, backup, recovery, audit, kek, operations, incidents, completion; see docs/cli.md")
}
func main() {
	if err := run(); err != nil {
		if jsonOutput {
			_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"status": "failed", "code": "command_failed"})
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(2)
	}
}

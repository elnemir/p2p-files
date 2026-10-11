package main

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/settings"
	"io"
	"os"
	"time"
)

func settingsCLI(args []string) (bool, error) {
	if len(args) != 2 {
		return false, nil
	}
	command := args[0] + " " + args[1]
	if command != "settings show" && command != "settings apply" && command != "network reset" && command != "config ingress-check" {
		return false, nil
	}
	var in struct {
		Expected int64           `json:"expectedRevision"`
		Value    json.RawMessage `json:"value"`
		Reason   string          `json:"reason"`
	}
	if command == "settings apply" || command == "network reset" {
		d := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		d.DisallowUnknownFields()
		if e := d.Decode(&in); e != nil || d.Decode(new(any)) != io.EOF {
			return true, errors.New("invalid settings JSON on stdin")
		}
		if command == "settings apply" {
			if in.Expected < 1 || len(in.Value) == 0 || in.Reason != "" {
				return true, errors.New("settings apply requires expectedRevision and value")
			}
		} else if in.Expected != 0 || len(in.Value) != 0 {
			return true, errors.New("network reset accepts reason only")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if command == "settings show" || command == "config ingress-check" {
		c, e := config.Load(os.Getenv)
		if e != nil {
			return true, e
		}
		db, e := database.Connect(ctx, c.DatabaseURL)
		if e != nil {
			return true, e
		}
		defer db.Close()
		active, e := (settings.Store{DB: db}).Active(ctx)
		if e != nil {
			return true, e
		}
		if command == "config ingress-check" {
			value, e := config.ParseSettings(active.Value)
			if e != nil {
				return true, e
			}
			if e = config.IngressReady(value); e != nil {
				return true, e
			}
			return true, json.NewEncoder(os.Stdout).Encode(map[string]any{"ingressConfigured": true, "revision": active.Revision, "productionReleaseApproved": false})
		}
		return true, json.NewEncoder(os.Stdout).Encode(active)
	}
	c, db, e := bootstrap.Open(ctx)
	if e != nil {
		return true, e
	}
	defer db.Close()
	unlock, e := maintenance.StartupShared(ctx, db)
	if e != nil {
		return true, e
	}
	defer unlock()
	key, e := database.AuditKey(ctx, db, c.MasterKEK)
	if e != nil {
		return true, errors.New("settings audit key unavailable")
	}
	s := settings.Store{DB: db, Key: key}
	switch command {
	case "network reset":
		id, e := s.ResetNetworkAllowlist(ctx, in.Reason)
		if e != nil {
			return true, e
		}
		return true, json.NewEncoder(os.Stdout).Encode(map[string]any{"revision": id, "gate": "closed"})
	default:
		id, e := s.Draft(ctx, "host_cli", in.Value)
		if e == nil {
			e = s.Validate(ctx, id)
		}
		if e == nil {
			e = s.Apply(ctx, "host_cli", id, in.Expected)
		}
		if e != nil {
			return true, e
		}
		return true, json.NewEncoder(os.Stdout).Encode(map[string]int64{"revision": id})
	}
}

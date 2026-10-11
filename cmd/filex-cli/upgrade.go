package main

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/operations"
	"io"
	"os"
	"time"
)

func upgradeCLI(args []string) (bool, error) {
	if len(args) == 0 || args[0] != "upgrade" {
		return false, nil
	}
	if len(args) != 2 || (args[1] != "check" && args[1] != "prepare") {
		return true, errors.New("usage: upgrade check|prepare")
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if args[1] == "prepare" {
		d := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		d.DisallowUnknownFields()
		if e := d.Decode(&in); e != nil {
			return true, errors.New("invalid upgrade JSON on stdin")
		}
		if d.Decode(new(any)) != io.EOF {
			return true, errors.New("trailing upgrade input")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, e := config.Load(os.Getenv)
	if e != nil {
		return true, e
	}
	db, e := database.Connect(ctx, c.DatabaseURL)
	if e != nil {
		return true, e
	}
	defer db.Close()
	plan, e := database.CheckUpgrade(ctx, db)
	if e != nil {
		return true, errors.New("upgrade compatibility check failed")
	}
	if args[1] == "check" {
		return true, json.NewEncoder(os.Stdout).Encode(plan)
	}
	if len(plan.Pending) != 0 {
		return true, errors.New("prepare requires the currently installed schema; use its matching binary")
	}
	// Do not call bootstrap.Open: preparing downtime must never apply DDL.
	s, e := operations.New(ctx, db, c.MasterKEK)
	if e != nil {
		return true, errors.New("upgrade preparation bootstrap failed")
	}
	out, e := s.PrepareUpgrade(ctx, "host_cli", in.Reason)
	if e != nil {
		return true, errors.New("upgrade preparation failed; inspect operations state; access may remain closed")
	}
	return true, json.NewEncoder(os.Stdout).Encode(out)
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/licensing"
	"filex.local/filex/internal/maintenance"
	"io"
	"os"
	"time"
)

func licenseCLI(args []string) (bool, error) {
	if len(args) == 0 || args[0] != "license" {
		return false, nil
	}
	if len(args) != 2 || (args[1] != "status" && args[1] != "apply") {
		return true, errors.New("use license status or license apply (JSON stdin)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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
		return true, e
	}
	store := licensing.Store{DB: db, AuditKey: key}
	var v licensing.Status
	if args[1] == "status" {
		v, e = store.Status(ctx)
	} else {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 32769))
		if err != nil {
			return true, err
		}
		var in struct {
			Envelope string `json:"envelope"`
			Epoch    *int64 `json:"expectedEpoch"`
		}
		if licensing.Decode(raw, &in) != nil || in.Epoch == nil {
			return true, licensing.ErrInvalid
		}
		v, e = store.Apply(ctx, in.Envelope, *in.Epoch, "host_cli", "", 0)
	}
	if e != nil {
		return true, e
	}
	return true, json.NewEncoder(os.Stdout).Encode(v)
}

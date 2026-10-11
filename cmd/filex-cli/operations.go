package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/operations"
	"fmt"
	"io"
	"os"
	"time"
)

type operationInput struct {
	Provider       string `json:"providerId"`
	Backup         string `json:"backupId"`
	Reason         string `json:"reason"`
	RecoveryKEK    string `json:"recoveryKek"`
	NewKEK         string `json:"newKek"`
	Reconciliation string `json:"reconciliationId"`
	Hash           string `json:"hash"`
	Ack            bool   `json:"acknowledgeCapabilities"`
	Kind           string `json:"kind"`
	Evidence       string `json:"evidenceHash"`
	ID             string `json:"id"`
	State          string `json:"state"`
	Interval       int    `json:"intervalSeconds"`
	Revision       int    `json:"revision"`
}

func operationalCLI(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if args[0] == "completion" {
		if len(args) != 2 || (args[1] != "bash" && args[1] != "zsh") {
			return true, errors.New("usage: completion bash|zsh")
		}
		if args[1] == "bash" {
			fmt.Println("complete -W 'config settings network migrate upgrade user storage backup recovery audit kek operations incidents completion' filex-cli")
		} else {
			fmt.Println("#compdef filex-cli\n_arguments '1:command:(config settings network migrate upgrade user storage backup recovery audit kek operations incidents completion)' '*:argument:'")
		}
		return true, nil
	}
	families := map[string]bool{"backup": true, "recovery": true, "audit": true, "kek": true, "operations": true, "incidents": true}
	if !families[args[0]] {
		return false, nil
	}
	if len(args) != 2 {
		return true, errors.New("operational commands require a subcommand; see wiki/cli.md")
	}
	command := args[0] + " " + args[1]
	var in operationInput
	noInput := command == "operations status" || command == "operations run" || command == "operations schedules" || command == "incidents list" || command == "audit verify" || command == "audit evidence"
	if !noInput {
		d := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
		d.DisallowUnknownFields()
		if e := d.Decode(&in); e != nil {
			return true, errors.New("invalid operation JSON on stdin")
		}
		if d.Decode(new(any)) != io.EOF {
			return true, errors.New("trailing operation input")
		}
	}
	var recoveryKey []byte
	if in.RecoveryKEK != "" {
		var e error
		recoveryKey, e = base64.StdEncoding.Strict().DecodeString(in.RecoveryKEK)
		in.RecoveryKEK = ""
		if e != nil || len(recoveryKey) != 32 {
			return true, errors.New("invalid recovery KEK")
		}
		defer clear(recoveryKey)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	c, db, e := bootstrap.Open(ctx)
	if e != nil {
		return true, e
	}
	defer db.Close()
	s, e := operations.New(ctx, db, c.MasterKEK)
	if e != nil {
		return true, errors.New("operational bootstrap failed")
	}
	if command != "backup create" && command != "backup restore" && command != "recovery reconcile" && command != "kek rewrap" && command != "operations run" {
		gate, err := maintenance.New(ctx, db)
		if err != nil {
			return true, err
		}
		defer gate.Close()
		if gate.Generation != s.Generation {
			return true, maintenance.ErrRestart
		}
		work, release, err := gate.Acquire(ctx, true)
		if err != nil {
			return true, err
		}
		defer release()
		ctx = work
	}
	output := any(map[string]string{"status": "completed"})
	switch command {
	case "operations status":
		state, err := s.State(ctx)
		if err != nil {
			e = err
			break
		}
		jobs, err := s.Jobs(ctx)
		e = err
		output = map[string]any{"state": state, "jobs": jobs}
	case "operations run":
		e = s.Run(ctx)
	case "operations schedules":
		output, e = s.Schedules(ctx)
	case "operations schedule":
		e = s.SetSchedule(ctx, "host_cli", in.Kind, in.Interval, in.Provider)
	case "backup create", "backup restore", "recovery reconcile":
		kind := "backup"
		if command == "backup restore" {
			kind = "restore"
		}
		if command == "recovery reconcile" {
			kind = "reconcile"
		}
		var id string
		id, e = s.QueueAndExecute(ctx, "host_cli", kind, in.Backup, in.Provider, in.Reason, recoveryKey)
		output = map[string]string{"id": id, "status": "completed"}
	case "backup verify":
		output, e = s.VerifyBackup(ctx, in.Backup, recoveryKey)
	case "backup import":
		e = s.ImportBackup(ctx, "host_cli", in.Backup, in.Provider, in.Reason, recoveryKey)
	case "recovery close":
		e = s.SetMaintenance(ctx, "host_cli", in.Reason)
	case "recovery report":
		output, e = s.Reconciliation(ctx, in.Reconciliation)
	case "recovery reopen":
		e = s.Reopen(ctx, "host_cli", in.Reconciliation, in.Hash, in.Reason, in.Ack)
	case "recovery revoke-capabilities":
		e = s.RevokeCapabilities(ctx, "host_cli", in.Reason)
	case "audit verify":
		e = s.CheckAudit(ctx)
	case "audit evidence":
		var digest string
		digest, e = s.AuditEvidence(ctx)
		output = map[string]string{"evidenceHash": digest}
	case "audit checkpoint":
		output, e = s.Checkpoint(ctx, "host_cli", in.Kind, in.Reason, in.Evidence)
	case "incidents list":
		output, e = s.Incidents(ctx)
	case "incidents update":
		e = s.UpdateIncident(ctx, in.ID, in.State, "host_cli", in.Reason)
	case "incidents report":
		var raw []byte
		var digest string
		raw, digest, e = s.IncidentReport(ctx, in.ID, in.Revision)
		output = map[string]any{"sha256": digest, "report": json.RawMessage(raw)}
	case "kek rewrap":
		var key []byte
		key, e = base64.StdEncoding.Strict().DecodeString(in.NewKEK)
		in.NewKEK = ""
		if e == nil && len(key) == 32 {
			e = s.Rewrap(ctx, key, in.Provider, "host_cli", in.Reason)
		} else {
			e = operations.ErrInvalid
		}
		clear(key)
		output = map[string]any{"status": "completed", "requiresRestart": true}
	default:
		return true, errors.New("unsupported operational command; see wiki/cli.md")
	}
	if e != nil {
		return true, errors.New("operation failed; inspect the protected operations dashboard and retry after resolving the cause")
	}
	return true, json.NewEncoder(os.Stdout).Encode(output)
}

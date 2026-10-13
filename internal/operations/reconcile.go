package operations

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/files"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/storage"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

type Finding struct {
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Severity string `json:"severity"`
}
type Reconciliation struct {
	ID         string    `json:"id"`
	Generation string    `json:"generation"`
	CreatedAt  time.Time `json:"createdAt"`
	Files      int       `json:"files"`
	Artifacts  int       `json:"artifacts"`
	Objects    int       `json:"objects"`
	Links      int       `json:"links"`
	Tokens     int       `json:"tokens"`
	Findings   []Finding `json:"findings"`
	Passed     bool      `json:"passed"`
	Hash       string    `json:"hash"`
}

func (s Service) checkCrypto(ctx context.Context) error {
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var box []byte
	if e = tx.QueryRow(ctx, "SELECT envelope FROM bootstrap_secrets WHERE name='key_check'").Scan(&box); e != nil {
		return e
	}
	sentinel, e := cryptobox.Open(s.Keys.KEK, "filex/bootstrap/v1/key_check", box)
	if e != nil || !bytes.Equal(sentinel, []byte("filex bootstrap key verification v1")) {
		return ErrBackup
	}
	rows, e := tx.Query(ctx, "SELECT id::text,wrapped FROM key_versions")
	if e != nil {
		return e
	}
	keys := map[string][]byte{}
	for rows.Next() {
		var id string
		var box []byte
		if e = rows.Scan(&id, &box); e != nil {
			rows.Close()
			return e
		}
		key, err := cryptobox.Open(s.Keys.KEK, "filex/keys/v1/"+id, box)
		if err != nil || len(key) != 32 {
			rows.Close()
			return ErrBackup
		}
		keys[id] = key
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	refs := []struct{ table, id, key, box, context, where string }{
		{"recipient_links", "id", "key_id", "token_envelope", "link", "true"},
		{"storage_providers", "id", "secret_key_id", "secret_envelope", "storage", "secret_key_id IS NOT NULL"},
		{"communication_configs", "id", "key_id", "envelope", "communications", "true"},
		{"identity_providers", "id", "secret_key_id", "secret_box", "identity_provider", "true"},
		{"identity_flows", "state_hash", "key_id", "box", "identity_flow", "true"},
		{"reports", "id", "key_id", "snapshot_envelope", "report", "state<>'purged'"},
		{"security_incident_reports", "id", "key_id", "envelope", "security_report", "true"},
	}
	for _, ref := range refs {
		rows, e := tx.Query(ctx, "SELECT "+ref.id+"::text,"+ref.key+"::text,"+ref.box+" FROM "+ref.table+" WHERE "+ref.where)
		if e != nil {
			return e
		}
		for rows.Next() {
			var id, kid string
			var box []byte
			if e = rows.Scan(&id, &kid, &box); e != nil {
				rows.Close()
				return e
			}
			if _, e = cryptobox.Open(keys[kid], "filex/"+ref.context+"/v1/"+id, box); e != nil {
				rows.Close()
				return ErrBackup
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
	}
	rows, e = tx.Query(ctx, "SELECT seed_key_id::text,public_key FROM signing_keys")
	if e != nil {
		return e
	}
	for rows.Next() {
		var id string
		var public []byte
		if e = rows.Scan(&id, &public); e != nil {
			rows.Close()
			return e
		}
		if len(keys[id]) != 32 || !bytes.Equal(ed25519.NewKeyFromSeed(keys[id]).Public().(ed25519.PublicKey), public) {
			rows.Close()
			return ErrBackup
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) CheckCrypto(ctx context.Context) error {
	if e := s.checkCrypto(ctx); e != nil {
		_, err := s.Incident(ctx, "crypto_integrity", "keys", "critical", map[string]string{"code": "crypto_check_failed"})
		return errors.Join(e, err)
	}
	return nil
}
func (s Service) scanFiles(ctx context.Context, report *Reconciliation, limit int) error {
	svc := files.Service{DB: s.DB, Keys: s.Keys, Storage: s.Storage}
	cursor := ""
	for {
		rows, e := s.DB.Query(ctx, "SELECT f.id::text,t.owner_id::text FROM files f JOIN transfers t ON t.id=f.transfer_id WHERE f.id::text>$1 AND f.state IN ('ready','leased','storage_missing','integrity_failed') ORDER BY f.id LIMIT 64", cursor)
		if e != nil {
			return e
		}
		type pair struct{ id, owner string }
		batch := []pair{}
		for rows.Next() {
			var p pair
			if e = rows.Scan(&p.id, &p.owner); e != nil {
				rows.Close()
				return e
			}
			batch = append(batch, p)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(batch) == 0 {
			break
		}
		for _, p := range batch {
			if e = ctx.Err(); e != nil {
				return e
			}
			f, e := svc.Get(ctx, p.owner, p.id)
			if e != nil {
				return e
			}
			h := sha256.New()
			e = svc.Read(ctx, f, 0, f.Size, h)
			report.Files++
			if e == nil && (f.SHA256 == nil || hex.EncodeToString(h.Sum(nil)) != *f.SHA256) {
				if e = svc.MarkFailure(ctx, f, "integrity_failed"); e != nil {
					return e
				}
				e = ErrBackup
			}
			if e != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				var unchanged bool
				if err := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM files WHERE id=$1 AND provider_id::text=$2 AND key_id::text=$3 AND salt=$4 AND state IN ('ready','leased','storage_missing','integrity_failed'))", f.ID, f.ProviderID, f.KeyID, f.Salt).Scan(&unchanged); err != nil {
					return err
				}
				if !unchanged {
					cursor = p.id
					continue
				}
				var state string
				if err := s.DB.QueryRow(ctx, "SELECT state FROM files WHERE id=$1", f.ID).Scan(&state); err != nil {
					return err
				}
				kind, severity := "storage_unavailable", "critical"
				if errors.Is(e, storage.ErrMissing) {
					kind, severity = "storage_missing", "warning"
				} else if state == "integrity_failed" {
					kind, severity = "integrity_failed", "warning"
				}
				report.Findings = append(report.Findings, Finding{kind, f.ID, severity})
				if _, err := s.Incident(ctx, kind, f.ID, severity, map[string]string{"fileId": f.ID}); err != nil {
					return err
				}
			}
			cursor = p.id
			if limit > 0 && report.Files >= limit {
				return nil
			}
		}
	}
	return nil
}
func (s Service) scanArtifacts(ctx context.Context, report *Reconciliation) error {
	store := artifacts.Store{DB: s.DB, Keys: s.Keys, Storage: s.Storage}
	cursor := ""
	for {
		rows, e := s.DB.Query(ctx, "SELECT id::text FROM artifacts WHERE id::text>$1 AND state='ready' AND expires_at>clock_timestamp() ORDER BY id LIMIT 64", cursor)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			a, e := store.Get(ctx, id)
			if e != nil {
				return e
			}
			h := sha256.New()
			e = store.Read(ctx, a, 0, a.Size, h)
			report.Artifacts++
			cursor = id
			if e != nil || a.SHA256 == "" || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if _, err := s.DB.Exec(ctx, "UPDATE artifacts SET expires_at=LEAST(expires_at,clock_timestamp()) WHERE id=$1", id); err != nil {
					return err
				}
				report.Findings = append(report.Findings, Finding{"artifact_unavailable", id, "warning"})
				if _, err := s.Incident(ctx, "artifact_unavailable", id, "warning", map[string]string{"artifactId": id}); err != nil {
					return err
				}
			}
		}
	}
}
func (s Service) inventory(ctx context.Context, report *Reconciliation) error {
	if e := s.rememberProviders(ctx); e != nil {
		return e
	}
	rows, e := s.DB.Query(ctx, "SELECT id::text,envelope FROM ops_provider_history ORDER BY id")
	if e != nil {
		return e
	}
	type item struct {
		id  string
		box []byte
	}
	all := []item{}
	for rows.Next() {
		var p item
		if e = rows.Scan(&p.id, &p.box); e != nil {
			rows.Close()
			return e
		}
		all = append(all, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range all {
		raw, e := s.open("filex/ops_inventory/v1/"+v.id, v.box)
		if e != nil {
			return e
		}
		var d storage.Descriptor
		if e = json.Unmarshal(raw, &d); e != nil {
			return e
		}
		provider, e := d.Open()
		if e != nil {
			return e
		}
		lister, ok := provider.(storage.Inventory)
		if !ok {
			return ErrBackup
		}
		cursor := ""
		for {
			keys, next, err := lister.List(ctx, cursor, 1000)
			if err != nil {
				report.Findings = append(report.Findings, Finding{"inventory_unavailable", v.id, "critical"})
				break
			}
			report.Objects += len(keys)
			if report.Objects > 1000000 {
				return ErrBackup
			}
			for _, key := range keys {
				var known bool
				e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM file_chunks c JOIN files f ON f.id=c.file_id WHERE f.provider_id=$1 AND c.object_key=$2 UNION ALL SELECT 1 FROM artifact_chunks c JOIN artifacts a ON a.id=c.artifact_id WHERE a.provider_id=$1 AND c.object_key=$2 UNION ALL SELECT 1 FROM deletion_jobs WHERE provider_id=$1 AND object_key=$2 AND completed_at IS NULL)`, v.id, key).Scan(&known)
				if e != nil {
					return e
				}
				if !known {
					report.Findings = append(report.Findings, Finding{"orphan_object", v.id + ":" + key, "warning"})
				}
				if len(report.Findings) > 10000 {
					return ErrBackup
				}
			}
			if next == "" {
				break
			}
			if next == cursor {
				return ErrBackup
			}
			cursor = next
		}
	}
	return nil
}
func (s Service) Reconcile(ctx context.Context, actor, reason string) (Reconciliation, error) {
	out := Reconciliation{ID: domain.NewID(), CreatedAt: time.Now().UTC(), Findings: []Finding{}, Passed: true}
	if !validReason(reason) {
		return out, ErrInvalid
	}
	state, e := s.State(ctx)
	if e != nil {
		return out, e
	}
	if state.Mode == "open" {
		return out, ErrConflict
	}
	out.Generation = state.Generation
	unlock, e := maintenance.Exclusive(ctx, s.DB)
	if e != nil {
		return out, e
	}
	defer unlock()
	if e = s.CheckAudit(ctx); e != nil {
		out.Findings = append(out.Findings, Finding{"audit_integrity", "audit", "critical"})
	}
	if e = s.CheckCrypto(ctx); e != nil {
		out.Findings = append(out.Findings, Finding{"crypto_integrity", "keys", "critical"})
	}
	if e = s.scanFiles(ctx, &out, 0); e != nil {
		return out, e
	}
	if e = s.scanArtifacts(ctx, &out); e != nil {
		return out, e
	}
	if e = s.inventory(ctx, &out); e != nil {
		return out, e
	}
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM recipient_links WHERE revoked_at IS NULL").Scan(&out.Links); e != nil {
		return out, e
	}
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM api_tokens WHERE revoked_at IS NULL AND expires_at>clock_timestamp()").Scan(&out.Tokens); e != nil {
		return out, e
	}
	var critical int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM security_incidents WHERE severity='critical' AND state<>'resolved'").Scan(&critical); e != nil {
		return out, e
	}
	if critical > 0 {
		out.Findings = append(out.Findings, Finding{"unresolved_critical_incidents", fmt.Sprint(critical), "critical"})
	}
	for _, f := range out.Findings {
		if f.Severity == "critical" {
			out.Passed = false
		}
	}
	raw, e := json.Marshal(out)
	if e != nil {
		return out, e
	}
	out.Hash = hash(raw)
	raw, e = json.Marshal(out)
	if e != nil {
		return out, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "INSERT INTO ops_reconciliations(id,generation,report,hash,passed) VALUES($1,$2,$3,$4,$5)", out.ID, out.Generation, raw, out.Hash, out.Passed); e != nil {
		return out, e
	}
	tag, e := tx.Exec(ctx, "UPDATE ops_control SET reconciliation_id=$1,mode=CASE WHEN $3 THEN 'reconcile' ELSE 'blocked' END,changed_at=clock_timestamp() WHERE generation=$2 AND mode<>'open'", out.ID, out.Generation, out.Passed)
	if e != nil {
		return out, e
	}
	if tag.RowsAffected() != 1 {
		return out, ErrConflict
	}
	if e = s.event(ctx, tx, actor, "reconciliation.completed", out.ID, reason); e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}
func (s Service) Reopen(ctx context.Context, actor, id, digest, reason string, ack bool) error {
	if !validReason(reason) || !domain.ValidID(id) || !ack {
		return ErrInvalid
	}
	unlock, e := s.runnerLock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var mode, generation string
	var current *string
	if e = tx.QueryRow(ctx, "SELECT mode,generation::text,reconciliation_id::text FROM ops_control WHERE singleton FOR UPDATE").Scan(&mode, &generation, &current); e != nil {
		return e
	}
	if mode == "open" || current == nil || *current != id {
		return ErrConflict
	}
	var valid bool
	e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_reconciliations WHERE id=$1 AND generation=$2 AND hash=$3 AND passed AND created_at>clock_timestamp()-interval '15 minutes') AND NOT EXISTS(SELECT 1 FROM security_incidents WHERE severity='critical' AND state<>'resolved') AND NOT EXISTS(SELECT 1 FROM ops_jobs WHERE state IN ('queued','running'))", id, generation, digest).Scan(&valid)
	if e != nil {
		return e
	}
	if !valid {
		return ErrConflict
	}
	if e = s.event(ctx, tx, actor, "maintenance.opened", id, reason); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode='open',reason=$1,actor=$2,changed_at=clock_timestamp()", reason, actor); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) CheckStorage(ctx context.Context) error {
	r := Reconciliation{Findings: []Finding{}}
	// Periodic checks verify payload cryptography without an inventory sweep or
	// reopening the recovery gate. An explicit reconciliation scans all objects.
	return s.scanFiles(ctx, &r, 0)
}
func (s Service) Reconciliation(ctx context.Context, id string) (json.RawMessage, error) {
	var raw json.RawMessage
	e := s.DB.QueryRow(ctx, "SELECT report FROM ops_reconciliations WHERE id::text=$1", id).Scan(&raw)
	return raw, e
}
func (s Service) RevokeCapabilities(ctx context.Context, actor, reason string) error {
	if !validReason(reason) {
		return ErrInvalid
	}
	state, e := s.State(ctx)
	if e != nil {
		return e
	}
	if state.Mode == "open" {
		return ErrConflict
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	for _, q := range []string{"UPDATE recipient_links SET revoked_at=clock_timestamp(),epoch=epoch+1 WHERE revoked_at IS NULL", "UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE revoked_at IS NULL", "UPDATE ops_control SET reconciliation_id=NULL"} {
		if _, e = tx.Exec(ctx, q); e != nil {
			return e
		}
	}
	if e = s.event(ctx, tx, actor, "capabilities.revoked", "system", reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

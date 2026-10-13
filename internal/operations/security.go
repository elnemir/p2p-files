package operations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/branding"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"github.com/jackc/pgx/v5"
	"time"
)

func boundedCleanup() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

type Incident struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Subject   string    `json:"subject"`
	Severity  string    `json:"severity"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s Service) Incidents(ctx context.Context) ([]Incident, error) {
	rows, e := s.DB.Query(ctx, "SELECT id::text,kind,subject,severity,state,created_at FROM security_incidents ORDER BY created_at DESC,id LIMIT 100")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Incident{}
	for rows.Next() {
		var v Incident
		if e = rows.Scan(&v.ID, &v.Kind, &v.Subject, &v.Severity, &v.State, &v.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Service) Incident(ctx context.Context, kind, subject, severity string, evidence any) (string, error) {
	// Containment must survive failure to decrypt the report encryption key.
	if severity == "critical" {
		if _, e := s.DB.Exec(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='restoring' THEN mode ELSE 'blocked' END,reason='security_incident',reconciliation_id=NULL,changed_at=clock_timestamp()"); e != nil {
			return "", e
		}
	}
	id := domain.NewID()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185022)"); e != nil {
		return "", e
	}
	var old string
	e = tx.QueryRow(ctx, "SELECT id::text FROM security_incidents WHERE kind=$1 AND subject=$2 AND state<>'resolved'", kind, subject).Scan(&old)
	if e == nil {
		return old, tx.Commit(ctx)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO security_incidents(id,kind,subject,severity) VALUES($1,$2,$3,$4)", id, kind, subject, severity); e != nil {
		return "", e
	}
	if e = s.incidentReport(ctx, tx, id, "detected", evidence); e != nil {
		return "", e
	}
	if severity == "critical" {
		if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode='blocked',reason='security_incident',reconciliation_id=NULL,changed_at=clock_timestamp() WHERE mode<>'restoring'"); e != nil {
			return "", e
		}
	}
	if e = s.event(ctx, tx, "scheduler", "incident.created", id, kind); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}
func (s Service) incidentReport(ctx context.Context, tx pgx.Tx, id, reason string, evidence any) error {
	var kind, subject, severity, state string
	if e := tx.QueryRow(ctx, "SELECT kind,subject,severity,state FROM security_incidents WHERE id=$1", id).Scan(&kind, &subject, &severity, &state); e != nil {
		return e
	}
	var revision int
	if e := tx.QueryRow(ctx, "SELECT COALESCE(max(revision),0)+1 FROM security_incident_reports WHERE incident_id=$1", id).Scan(&revision); e != nil {
		return e
	}
	presentation, e := branding.Snapshot(ctx, tx)
	if e != nil {
		return e
	}
	reportID := domain.NewID()
	raw, e := json.Marshal(map[string]any{"schemaVersion": 1, "id": reportID, "incidentId": id, "revision": revision, "kind": kind, "subject": subject, "severity": severity, "state": state, "reason": reason, "evidence": evidence, "createdAt": time.Now().UTC()})
	if e != nil {
		return e
	}
	kid, key, e := s.Keys.Active(ctx, "configuration")
	if e != nil {
		return e
	}
	box, e := s.Keys.Seal(ctx, kid, key, "filex/security_report/v1/"+reportID, raw)
	if e != nil {
		return e
	}
	if e = keyring.RequireActive(ctx, tx, kid); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "INSERT INTO security_incident_reports(id,incident_id,revision,key_id,envelope,sha256,branding) VALUES($1,$2,$3,$4,$5,$6,$7)", reportID, id, revision, kid, box, hash(raw), presentation)
	return e
}
func (s Service) IncidentReport(ctx context.Context, id string, revision int) ([]byte, string, error) {
	var reportID, kid, digest string
	var box []byte
	e := s.DB.QueryRow(ctx, "SELECT id::text,key_id::text,envelope,sha256 FROM security_incident_reports WHERE incident_id::text=$1 AND ($2=0 OR revision=$2) ORDER BY revision DESC LIMIT 1", id, revision).Scan(&reportID, &kid, &box, &digest)
	if e != nil {
		return nil, "", e
	}
	key, e := s.Keys.Get(ctx, kid)
	if e != nil {
		return nil, "", e
	}
	raw, e := cryptobox.Open(key, "filex/security_report/v1/"+reportID, box)
	if e != nil || hash(raw) != digest {
		return nil, "", ErrBackup
	}
	return raw, digest, nil
}
func (s Service) UpdateIncident(ctx context.Context, id, state, actor, reason string) error {
	if !validReason(reason) || (state != "acknowledged" && state != "resolved") {
		return ErrInvalid
	}
	if state == "resolved" {
		var kind string
		if e := s.DB.QueryRow(ctx, "SELECT kind FROM security_incidents WHERE id::text=$1", id).Scan(&kind); e != nil {
			return e
		}
		if kind == "audit_integrity" {
			if e := audit.Verify(ctx, s.DB, s.AuditKey); e != nil {
				return ErrConflict
			}
		}
		if kind == "crypto_integrity" {
			if e := s.checkCrypto(ctx); e != nil {
				return ErrConflict
			}
		}
		if kind == "operations_audit_integrity" {
			if e := s.verifyLedger(ctx); e != nil {
				return ErrConflict
			}
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var locked string
	if e = tx.QueryRow(ctx, "SELECT id::text FROM security_incidents WHERE id::text=$1 AND state<>'resolved' FOR UPDATE", id).Scan(&locked); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE security_incidents SET state=$2,updated_at=clock_timestamp() WHERE id=$1", id, state); e != nil {
		return e
	}
	if e = s.incidentReport(ctx, tx, id, reason, map[string]string{"actor": actor}); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET reconciliation_id=NULL WHERE mode<>'open'"); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "incident."+state, id, reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) AuditEvidence(ctx context.Context) (string, error) {
	app, e := audit.Evidence(ctx, s.DB, "audit_events", -1)
	if e != nil {
		return "", e
	}
	ops, e := audit.Evidence(ctx, s.DB, "ops_events", -1)
	if e != nil {
		return "", e
	}
	return hash([]byte("filex/recovery/evidence/v1/" + app + "/" + ops)), nil
}
func (s Service) CheckAudit(ctx context.Context) error {
	if e := audit.Verify(ctx, s.DB, s.AuditKey); e != nil {
		evidence, err := s.AuditEvidence(ctx)
		if err != nil {
			return err
		}
		_, err = s.Incident(ctx, "audit_integrity", "audit", "critical", map[string]string{"evidenceHash": evidence})
		return errors.Join(e, err)
	}
	if e := s.verifyLedger(ctx); e != nil {
		_, err := s.Incident(ctx, "operations_audit_integrity", "operations_ledger", "critical", map[string]string{"code": "ledger_check_failed"})
		return errors.Join(e, err)
	}
	return nil
}
func (s Service) Checkpoint(ctx context.Context, actor, kind, reason, expected string) (map[string]any, error) {
	if !validReason(reason) || (kind != "regular" && kind != "recovery") {
		return nil, ErrInvalid
	}
	state, e := s.State(ctx)
	if e != nil {
		return nil, e
	}
	if kind == "recovery" && state.Mode == "open" {
		return nil, ErrConflict
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(context.Background())
	var mode, generation string
	if e = tx.QueryRow(ctx, "SELECT mode,generation::text FROM ops_control FOR UPDATE").Scan(&mode, &generation); e != nil {
		return nil, e
	}
	if generation != s.Generation || (kind == "recovery" && mode == "open") {
		return nil, ErrConflict
	}
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724184995),pg_advisory_xact_lock(724185021)"); e != nil {
		return nil, e
	}
	evidence, e := s.AuditEvidence(ctx)
	if e != nil {
		return nil, e
	}
	if kind == "recovery" && (expected == "" || expected != evidence) {
		return nil, ErrConflict
	}
	if kind == "regular" {
		if e = errors.Join(audit.Verify(ctx, s.DB, s.AuditKey), s.verifyLedger(ctx)); e != nil {
			return nil, ErrConflict
		}
	}
	appEvidence, e := audit.Evidence(ctx, tx, "audit_events", -1)
	if e != nil {
		return nil, e
	}
	opsEvidence, e := audit.Evidence(ctx, tx, "ops_events", -1)
	if e != nil {
		return nil, e
	}
	cp := audit.Checkpoint{ID: domain.NewID(), Kind: kind, EvidenceHash: appEvidence, Reason: reason, CreatedAt: time.Now().UTC()}
	e = tx.QueryRow(ctx, "SELECT id,hash FROM audit_events ORDER BY id DESC LIMIT 1").Scan(&cp.EventID, &cp.EventHash)
	if errors.Is(e, pgx.ErrNoRows) {
		if kind == "recovery" {
			return nil, ErrConflict
		}
	} else if e != nil {
		return nil, e
	}
	raw, e := json.Marshal(cp)
	if e != nil {
		return nil, e
	}
	signature := audit.SignCheckpoint(s.AuditKey, raw)
	if _, e = tx.Exec(ctx, "INSERT INTO audit_checkpoints(id,kind,event_id,event_hash,evidence_hash,payload,signature,reason) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", cp.ID, kind, cp.EventID, cp.EventHash, appEvidence, raw, signature, reason); e != nil {
		return nil, e
	}
	ocp := audit.Checkpoint{ID: cp.ID, Kind: kind, EvidenceHash: opsEvidence, Reason: reason, CreatedAt: cp.CreatedAt}
	e = tx.QueryRow(ctx, "SELECT id,hash FROM ops_events ORDER BY id DESC LIMIT 1").Scan(&ocp.EventID, &ocp.EventHash)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	oraw, e := json.Marshal(ocp)
	if e != nil {
		return nil, e
	}
	osig := audit.SignCheckpoint(s.OpsKey, oraw)
	if _, e = tx.Exec(ctx, "INSERT INTO ops_checkpoints(id,event_id,payload,signature) VALUES($1,$2,$3,$4)", ocp.ID, ocp.EventID, oraw, osig); e != nil {
		return nil, e
	}
	if e = s.event(ctx, tx, actor, "audit.checkpoint", cp.ID, reason); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET reconciliation_id=NULL WHERE mode<>'open'"); e != nil {
		return nil, e
	}
	return map[string]any{"checkpoint": cp, "signature": signature, "operationsCheckpoint": ocp, "operationsSignature": osig}, tx.Commit(ctx)
}
func (s Service) Checkpoints(ctx context.Context) ([]map[string]any, error) {
	rows, e := s.DB.Query(ctx, "SELECT a.payload,a.signature,o.payload,o.signature FROM audit_checkpoints a LEFT JOIN ops_checkpoints o ON o.id=a.id ORDER BY a.created_at DESC,a.id LIMIT 100")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var p json.RawMessage
		var sig string
		var op json.RawMessage
		var osig *string
		if e = rows.Scan(&p, &sig, &op, &osig); e != nil {
			return nil, e
		}
		out = append(out, map[string]any{"checkpoint": p, "signature": sig, "operationsCheckpoint": op, "operationsSignature": osig})
	}
	return out, rows.Err()
}
func (s Service) verifyLedger(ctx context.Context) error {
	var after int64
	var prev string
	rows, e := s.DB.Query(ctx, "SELECT id::text,event_id,payload,signature FROM ops_checkpoints ORDER BY event_id DESC,created_at DESC,id DESC")
	if e != nil {
		return e
	}
	for rows.Next() {
		var id, sig string
		var boundary int64
		var raw []byte
		var cp audit.Checkpoint
		if e = rows.Scan(&id, &boundary, &raw, &sig); e != nil {
			rows.Close()
			return e
		}
		if e = json.Unmarshal(raw, &cp); e != nil || !audit.CheckSignature(s.OpsKey, raw, sig) || cp.ID != id || cp.EventID != boundary {
			rows.Close()
			return ErrBackup
		}
		if cp.Kind == "recovery" {
			rows.Close()
			evidence, err := audit.Evidence(ctx, s.DB, "ops_events", cp.EventID)
			if err != nil || evidence != cp.EvidenceHash {
				return ErrBackup
			}
			if cp.EventID > 0 {
				var digest string
				if err = s.DB.QueryRow(ctx, "SELECT hash FROM ops_events WHERE id=$1", cp.EventID).Scan(&digest); err != nil || digest != cp.EventHash {
					return ErrBackup
				}
			}
			after, prev = cp.EventID, cp.EventHash
			break
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = s.DB.Query(ctx, "SELECT payload,previous_hash,hash FROM ops_events WHERE id>$1 ORDER BY id", after)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var parent, digest string
		if e = rows.Scan(&raw, &parent, &digest); e != nil {
			return e
		}
		mac := hmac.New(sha256.New, s.OpsKey)
		mac.Write([]byte("filex/ops/audit/v1/" + prev))
		mac.Write(raw)
		if parent != prev || hex.EncodeToString(mac.Sum(nil)) != digest {
			return ErrBackup
		}
		prev = digest
	}
	return rows.Err()
}

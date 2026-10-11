// Package operations owns recovery state independently of restored data.
package operations

import (
	"context"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/maintenance"
	"filex.local/filex/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrInvalid = errors.New("invalid operation")
var ErrConflict = errors.New("operation state conflict")
var ErrBackup = errors.New("backup verification failed")

type Service struct {
	DB               *pgxpool.Pool
	Keys             keyring.Ring
	Storage          storage.Registry
	AuditKey, OpsKey []byte
	Generation       string
}

func New(ctx context.Context, db *pgxpool.Pool, kek []byte) (Service, error) {
	s := Service{DB: db, Keys: keyring.Ring{DB: db, KEK: kek}}
	unlock, e := maintenance.StartupShared(ctx, db)
	if e != nil {
		return s, e
	}
	defer unlock()
	s.Storage = storage.Registry{DB: db, Keys: s.Keys}
	s.AuditKey, e = database.AuditKey(ctx, db, kek)
	if e != nil {
		return s, e
	}
	tx, e := db.Begin(ctx)
	if e != nil {
		return s, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185020)"); e != nil {
		return s, e
	}
	var box []byte
	e = tx.QueryRow(ctx, "SELECT envelope FROM ops_secrets WHERE name='ledger_key'").Scan(&box)
	if errors.Is(e, pgx.ErrNoRows) {
		tag, err := db.Exec(ctx, "UPDATE crypto_usage SET seal_count=seal_count+1 WHERE name='kek' AND seal_count<1048576")
		if err != nil {
			return s, err
		}
		if tag.RowsAffected() != 1 {
			return s, keyring.ErrUsage
		}
		key := make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return s, e
		}
		box, e = cryptobox.Seal(kek, "filex/ops/v1/ledger_key", key)
		if e != nil {
			return s, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO ops_secrets VALUES('ledger_key',$1)", box); e != nil {
			return s, e
		}
	} else if e != nil {
		return s, e
	}
	s.OpsKey, e = cryptobox.Open(kek, "filex/ops/v1/ledger_key", box)
	if e != nil {
		return s, e
	}
	if e = tx.Commit(ctx); e != nil {
		return s, e
	}
	state, e := maintenance.Read(ctx, db)
	s.Generation = state.Generation
	return s, e
}
func validReason(v string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(v)) >= 5 && utf8.RuneCountInString(v) <= 2000
}
func hash(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func (s Service) seal(aad string, plain []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, e := rand.Read(salt); e != nil {
		return nil, e
	}
	key, e := hkdf.Key(sha256.New, s.OpsKey, salt, aad, 32)
	if e != nil {
		return nil, e
	}
	box, e := cryptobox.Seal(key, aad, plain)
	return append(salt, box...), e
}
func (s Service) open(aad string, box []byte) ([]byte, error) {
	if len(box) < 61 {
		return nil, ErrBackup
	}
	key, e := hkdf.Key(sha256.New, s.OpsKey, box[:32], aad, 32)
	if e != nil {
		return nil, e
	}
	return cryptobox.Open(key, aad, box[32:])
}
func (s Service) event(ctx context.Context, tx pgx.Tx, actor, action, subject, reason string) error {
	if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185021)"); e != nil {
		return e
	}
	var prev string
	e := tx.QueryRow(ctx, "SELECT hash FROM ops_events ORDER BY id DESC LIMIT 1").Scan(&prev)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	b, e := json.Marshal(map[string]string{"actor": actor, "action": action, "subject": subject, "reason": reason, "at": time.Now().UTC().Format(time.RFC3339Nano)})
	if e != nil {
		return e
	}
	mac := hmac.New(sha256.New, s.OpsKey)
	mac.Write([]byte("filex/ops/audit/v1/" + prev))
	mac.Write(b)
	_, e = tx.Exec(ctx, "INSERT INTO ops_events(payload,previous_hash,hash) VALUES($1,$2,$3)", b, prev, hex.EncodeToString(mac.Sum(nil)))
	return e
}
func (s Service) SetMaintenance(ctx context.Context, actor, reason string) error {
	if !validReason(reason) {
		return ErrInvalid
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode=CASE WHEN mode='open' THEN 'maintenance' ELSE mode END,reason=$1,actor=$2,reconciliation_id=NULL,changed_at=clock_timestamp() WHERE singleton", reason, actor); e != nil {
		return e
	}
	if e = s.event(ctx, tx, actor, "maintenance.closed", "system", reason); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

type Job struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	State       string          `json:"state"`
	Reason      string          `json:"reason"`
	SourceID    *string         `json:"sourceId"`
	ProviderID  *string         `json:"providerId"`
	PreBackupID *string         `json:"preBackupId"`
	Progress    int64           `json:"progress"`
	ErrorCode   *string         `json:"errorCode"`
	Result      json.RawMessage `json:"result"`
	CreatedAt   time.Time       `json:"createdAt"`
	CompletedAt *time.Time      `json:"completedAt"`
}

func (s Service) Jobs(ctx context.Context) ([]Job, error) {
	rows, e := s.DB.Query(ctx, "SELECT id::text,kind,state,reason,source_id::text,provider_id::text,pre_backup_id::text,progress,error_code,result,created_at,completed_at FROM ops_jobs ORDER BY created_at DESC,id LIMIT 100")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Job{}
	for rows.Next() {
		var j Job
		if e = rows.Scan(&j.ID, &j.Kind, &j.State, &j.Reason, &j.SourceID, &j.ProviderID, &j.PreBackupID, &j.Progress, &j.ErrorCode, &j.Result, &j.CreatedAt, &j.CompletedAt); e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
func (s Service) Queue(ctx context.Context, actor, kind, source, provider, reason string) (string, error) {
	if !validReason(reason) {
		return "", ErrInvalid
	}
	if kind != "backup" && kind != "restore" && kind != "reconcile" && kind != "audit" && kind != "storage" {
		return "", ErrInvalid
	}
	id := domain.NewID()
	var box []byte
	if kind == "backup" || kind == "restore" {
		if !domain.ValidID(provider) {
			return "", ErrInvalid
		}
		var ok bool
		if e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM storage_providers WHERE id=$1 AND purpose='backups' AND enabled AND NOT draining)", provider).Scan(&ok); e != nil {
			return "", e
		}
		if !ok {
			return "", ErrInvalid
		}
		d, e := s.Storage.Describe(ctx, provider)
		if e != nil {
			return "", e
		}
		raw, e := json.Marshal(d)
		if e != nil {
			return "", e
		}
		box, e = s.seal("filex/ops_provider/v1/"+id, raw)
		if e != nil {
			return "", e
		}
	}
	if kind == "restore" && !domain.ValidID(source) {
		return "", ErrInvalid
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	var mode string
	if e = tx.QueryRow(ctx, "SELECT mode FROM ops_control WHERE singleton FOR UPDATE").Scan(&mode); e != nil {
		return "", e
	}
	var busy bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_jobs WHERE kind='restore' AND state IN ('queued','running'))").Scan(&busy); e != nil {
		return "", e
	}
	if busy {
		return "", ErrConflict
	}
	if kind == "restore" {
		var ok bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ops_jobs WHERE id=$1 AND kind='backup' AND state='completed')", source).Scan(&ok); e != nil {
			return "", e
		}
		if !ok {
			return "", ErrInvalid
		}
		if _, e = tx.Exec(ctx, "UPDATE ops_control SET mode='maintenance',reason=$1,actor=$2,reconciliation_id=NULL,changed_at=clock_timestamp()", reason, actor); e != nil {
			return "", e
		}
	}
	if kind == "reconcile" && mode == "open" {
		return "", ErrConflict
	}
	if _, e = tx.Exec(ctx, "INSERT INTO ops_jobs(id,kind,actor,reason,source_id,provider_id,provider_envelope) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,$7)", id, kind, actor, reason, source, provider, box); e != nil {
		return "", e
	}
	if e = s.event(ctx, tx, actor, kind+".queued", id, reason); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}
func (s Service) jobProvider(ctx context.Context, id string) (storage.Provider, error) {
	var box []byte
	if e := s.DB.QueryRow(ctx, "SELECT provider_envelope FROM ops_jobs WHERE id=$1", id).Scan(&box); e != nil {
		return nil, e
	}
	plain, e := s.open("filex/ops_provider/v1/"+id, box)
	if e != nil {
		return nil, e
	}
	var d storage.Descriptor
	if e = json.Unmarshal(plain, &d); e != nil {
		return nil, e
	}
	return d.Open()
}
func (s Service) State(ctx context.Context) (maintenance.State, error) {
	return maintenance.Read(ctx, s.DB)
}

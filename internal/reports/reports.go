package reports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"strings"
	"time"
)

type Service struct {
	DB        *pgxpool.Pool
	Keys      keyring.Ring
	Artifacts artifacts.Store
	AuditKey  []byte
}

func Canonical(v any) ([]byte, error) { return json.Marshal(v) }
func (s Service) Create(ctx context.Context, actor, transferID, kind, locale, timezone string) (string, string, error) {
	if kind != "user" && kind != "admin" {
		return "", "", domain.ErrInvalid
	}
	if locale != "ru" && locale != "en" {
		return "", "", domain.ErrInvalid
	}
	if _, e := time.LoadLocation(timezone); e != nil {
		return "", "", domain.ErrInvalid
	}
	kid, key, e := s.Keys.Active(ctx, "artifacts")
	if e != nil {
		return "", "", e
	}
	if e = s.Keys.Reserve(ctx, kid, 1); e != nil {
		return "", "", e
	}
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if e != nil {
		return "", "", e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='10s'"); e != nil {
		return "", "", e
	}
	var owner, title, state string
	var published *time.Time
	var manifest []byte
	var hash *string
	if e = tx.QueryRow(ctx, "SELECT owner_id::text,title,state,published_at,manifest,manifest_hash FROM transfers WHERE id::text=$1 AND ($2='admin' OR owner_id::text=$3) FOR UPDATE", transferID, kind, actor).Scan(&owner, &title, &state, &published, &manifest, &hash); e != nil {
		return "", "", e
	}
	var rev int
	if e = tx.QueryRow(ctx, "SELECT COALESCE(max(revision),0)+1 FROM reports WHERE transfer_id=$1 AND kind=$2", transferID, kind).Scan(&rev); e != nil {
		return "", "", e
	}
	id := domain.NewID()
	var now time.Time
	if e = tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&now); e != nil {
		return "", "", e
	}
	snapshot := map[string]any{"schemaVersion": 1, "reportId": id, "revision": rev, "kind": kind, "transferId": transferID, "title": title, "state": state, "publishedAt": published, "manifestHash": hash, "createdAt": now.UTC().Format(time.RFC3339Nano), "locale": locale, "timezone": timezone}
	if manifest != nil {
		var m any
		d := json.NewDecoder(strings.NewReader(string(manifest)))
		d.UseNumber()
		if e = d.Decode(&m); e != nil {
			return "", "", e
		}
		snapshot["manifest"] = m
	}
	rows, e := tx.Query(ctx, "SELECT path,size,sha256,state FROM files WHERE transfer_id=$1 ORDER BY path", transferID)
	if e != nil {
		return "", "", e
	}
	files := []map[string]any{}
	var bytes int
	for rows.Next() {
		var path, state string
		var size int64
		var sha *string
		if e = rows.Scan(&path, &size, &sha, &state); e != nil {
			rows.Close()
			return "", "", e
		}
		bytes += len(path) + 128
		if bytes > 12<<20 {
			rows.Close()
			return "", "", errors.New("report snapshot limit exceeded")
		}
		files = append(files, map[string]any{"path": path, "size": size, "sha256": sha, "state": state})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return "", "", e
	}
	snapshot["files"] = files
	rows, e = tx.Query(ctx, "SELECT occurred_at,file_count,total_bytes,status FROM download_journal WHERE transfer_id=$1 ORDER BY id", transferID)
	if e != nil {
		return "", "", e
	}
	journal := []map[string]any{}
	for rows.Next() {
		var at time.Time
		var count, bytes int64
		var status string
		if e = rows.Scan(&at, &count, &bytes, &status); e != nil {
			rows.Close()
			return "", "", e
		}
		journal = append(journal, map[string]any{"occurredAt": at.UTC().Format(time.RFC3339Nano), "fileCount": count, "totalBytes": bytes, "status": status})
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return "", "", e
	}
	snapshot["downloads"] = journal
	if kind == "admin" {
		rows, e = tx.Query(ctx, "SELECT d.id::text,l.recipient_email,d.kind,d.state,d.created_at,d.completed_at FROM downloads d JOIN recipient_sessions rs ON rs.id=d.recipient_session_id JOIN recipient_links l ON l.id=rs.link_id WHERE d.transfer_id=$1 ORDER BY d.created_at LIMIT 100001", transferID)
		if e != nil {
			return "", "", e
		}
		events := []map[string]any{}
		for rows.Next() {
			var id, kind, state string
			var email *string
			var at time.Time
			var completed *time.Time
			if e = rows.Scan(&id, &email, &kind, &state, &at, &completed); e != nil {
				rows.Close()
				return "", "", e
			}
			if len(events) >= 100000 {
				rows.Close()
				return "", "", errors.New("admin report limit exceeded")
			}
			events = append(events, map[string]any{"downloadId": id, "recipientEmail": email, "kind": kind, "state": state, "createdAt": at, "completedAt": completed})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return "", "", e
		}
		snapshot["administrativeEvents"] = events
	}
	content, e := Canonical(snapshot)
	if e != nil {
		return "", "", e
	}
	contentHash := sha256.Sum256(content)
	snapshot["documentContentHash"] = hex.EncodeToString(contentHash[:])
	raw, e := Canonical(snapshot)
	if e != nil {
		return "", "", e
	}
	if len(raw) > 16<<20 {
		return "", "", errors.New("report snapshot limit exceeded")
	}
	sum := sha256.Sum256(raw)
	canonicalHash := hex.EncodeToString(sum[:])
	box, e := cryptobox.Seal(key, "filex/report/v1/"+id, raw)
	if e != nil {
		return "", "", e
	}
	if e = keyring.RequireActive(ctx, tx, kid); e != nil {
		return "", "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO reports(id,owner_id,transfer_id,kind,revision,canonical_hash,key_id,snapshot_envelope,locale,timezone) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)", id, actor, transferID, kind, rev, canonicalHash, kid, box, locale, timezone); e != nil {
		return "", "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO report_history(report_id,actor_id,action) VALUES($1,$2,'created')", id, actor); e != nil {
		return "", "", e
	}
	if e = audit.Append(ctx, tx, s.AuditKey, actor, "report.created", id+":"+canonicalHash); e != nil {
		return "", "", e
	}
	return id, canonicalHash, tx.Commit(ctx)
}
func (s Service) Snapshot(ctx context.Context, id string) ([]byte, string, error) {
	var keyID, hash string
	var box []byte
	if e := s.DB.QueryRow(ctx, "SELECT key_id::text,snapshot_envelope,canonical_hash FROM reports WHERE id::text=$1 AND state<>'purged'", id).Scan(&keyID, &box, &hash); e != nil {
		return nil, "", e
	}
	key, e := s.Keys.Get(ctx, keyID)
	if e != nil {
		return nil, "", e
	}
	raw, e := cryptobox.Open(key, "filex/report/v1/"+id, box)
	if e != nil {
		return nil, "", e
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != hash {
		return nil, "", errors.New("report integrity failed")
	}
	return raw, hash, nil
}
func (s Service) RenderPending(ctx context.Context) error {
	var id, owner string
	var fence int64
	e := s.DB.QueryRow(ctx, `WITH next AS(SELECT id FROM reports WHERE state='active' AND (render_state='pending' OR (render_state='running' AND render_lease_until<clock_timestamp())) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE reports r SET render_state='running',render_fence=render_fence+1,render_lease_until=clock_timestamp()+interval '5 minutes' FROM next WHERE r.id=next.id RETURNING r.id::text,r.owner_id::text,r.render_fence`).Scan(&id, &owner, &fence)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	raw, hash, e := s.Snapshot(ctx, id)
	if e != nil {
		return e
	}
	formats := map[string]string{"json": "application/json", "csv": "text/csv; charset=utf-8", "pdf": "application/pdf"}
	created := map[string]string{}
	for format, media := range formats {
		artifact, err := s.Artifacts.Create(ctx, owner, "report-"+id+"."+format, media, func(w io.Writer) error { return Render(w, raw, hash, format) })
		if err != nil {
			_, update := s.DB.Exec(ctx, "UPDATE reports SET render_state='failed',render_error='render_failed' WHERE id=$1 AND render_fence=$2", id, fence)
			if update != nil {
				return update
			}
			return err
		}
		created[format] = artifact
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	tag, e := tx.Exec(ctx, "UPDATE reports SET render_state='ready',render_error=NULL WHERE id=$1 AND render_fence=$2 AND render_state='running' AND render_lease_until>clock_timestamp() AND state='active'", id, fence)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errors.New("stale report worker")
	}
	for format, artifact := range created {
		if _, e = tx.Exec(ctx, "INSERT INTO report_artifacts(report_id,artifact_id,format) VALUES($1,$2,$3) ON CONFLICT(report_id,format) DO UPDATE SET artifact_id=excluded.artifact_id", id, artifact, format); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}

package exports

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/artifacts"
	"filex.local/filex/internal/reports"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
)

type Service struct {
	DB        *pgxpool.Pool
	Artifacts artifacts.Store
}
type job struct {
	ID, Owner, Kind, Format, Compression, State string
	Fence                                       int64
	Bytes                                       int64
	Count                                       int
}

// Claim serializes the global/per-owner limits. Aging moves any five-minute-old
// USER job ahead of newly arriving ADMIN jobs, preventing starvation.
func (s Service) claim(ctx context.Context) (job, error) {
	var j job
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return j, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185010)"); e != nil {
		return j, e
	}
	if _, e = tx.Exec(ctx, "UPDATE export_jobs SET state=CASE WHEN snapshot_at IS NULL THEN 'queued' ELSE 'materialized' END,lease_until=NULL WHERE state IN ('snapshotting','rendering') AND lease_until<clock_timestamp()"); e != nil {
		return j, e
	}
	if _, e = tx.Exec(ctx, "UPDATE export_jobs e SET state='cancelled',fence=fence+1,lease_until=NULL,error_code='authorization_changed' FROM users u WHERE e.owner_id=u.id AND e.state IN ('queued','snapshotting','materialized','rendering') AND (u.disabled OR u.auth_epoch<>e.auth_epoch OR (e.kind='admin' AND u.role<>'admin'))"); e != nil {
		return j, e
	}
	var active int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM export_jobs WHERE state IN ('snapshotting','rendering') AND lease_until>clock_timestamp()").Scan(&active); e != nil {
		return j, e
	}
	if active >= 2 {
		if e = tx.Commit(ctx); e != nil {
			return j, e
		}
		return j, pgx.ErrNoRows
	}
	e = tx.QueryRow(ctx, `SELECT id::text,owner_id::text,kind,format,compression,state,COALESCE(snapshot_bytes,0),rows_count FROM export_jobs e WHERE state IN ('queued','materialized') AND NOT EXISTS(SELECT 1 FROM export_jobs running WHERE running.owner_id=e.owner_id AND running.state IN ('snapshotting','rendering') AND running.lease_until>clock_timestamp()) ORDER BY CASE WHEN created_at<clock_timestamp()-interval '5 minutes' THEN 0 WHEN kind='admin' THEN 1 ELSE 2 END,created_at,id FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&j.ID, &j.Owner, &j.Kind, &j.Format, &j.Compression, &j.State, &j.Bytes, &j.Count)
	if errors.Is(e, pgx.ErrNoRows) {
		if e = tx.Commit(ctx); e != nil {
			return j, e
		}
		return j, pgx.ErrNoRows
	}
	if e != nil {
		return j, e
	}
	state := "snapshotting"
	if j.State == "materialized" {
		state = "rendering"
	}
	if e = tx.QueryRow(ctx, "UPDATE export_jobs SET state=$2,fence=fence+1,lease_until=clock_timestamp()+interval '3 minutes',processed=0 WHERE id=$1 RETURNING fence", j.ID, state).Scan(&j.Fence); e != nil {
		return j, e
	}
	return j, tx.Commit(ctx)
}
func (s Service) snapshot(ctx context.Context, j job) error {
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='10s'"); e != nil {
		return e
	}
	var state string
	if e = tx.QueryRow(ctx, "SELECT state FROM export_jobs WHERE id=$1 AND fence=$2 AND lease_until>clock_timestamp() FOR UPDATE", j.ID, j.Fence).Scan(&state); e != nil {
		return e
	}
	if state != "snapshotting" {
		return errors.New("export cancelled")
	}
	if e = s.authorize(ctx, tx, j); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM export_rows WHERE job_id=$1", j.ID); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO export_rows(job_id,ordinal,value) SELECT $1,row_number() OVER(ORDER BY t.created_at,t.id),jsonb_build_object('id',t.id,'title',t.title,'state',t.state,'createdAt',t.created_at,'publishedAt',t.published_at,'expiresAt',t.expires_at,'manifestHash',t.manifest_hash,'fileCount',(SELECT count(*) FROM files f WHERE f.transfer_id=t.id),'totalBytes',(SELECT COALESCE(sum(size),0) FROM files f WHERE f.transfer_id=t.id)) FROM transfers t WHERE t.owner_id=$2 OR $3='admin' ORDER BY t.created_at,t.id LIMIT 50001`, j.ID, j.Owner, j.Kind); e != nil {
		return e
	}
	var count int
	var size int64
	if e = tx.QueryRow(ctx, "SELECT count(*),COALESCE(sum(octet_length(value::text)),0) FROM export_rows WHERE job_id=$1", j.ID).Scan(&count, &size); e != nil {
		return e
	}
	if count > 50000 || size > 16<<20 {
		return errors.New("snapshot_limit_exceeded")
	}
	if _, e = tx.Exec(ctx, "UPDATE export_jobs SET state='materialized',snapshot_at=transaction_timestamp(),snapshot_bytes=$3,rows_count=$4,lease_until=NULL WHERE id=$1 AND fence=$2", j.ID, j.Fence, size, count); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) Run(ctx context.Context) error {
	j, e := s.claim(ctx)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	if j.State == "queued" {
		e = s.snapshot(ctx, j)
	} else {
		e = s.render(ctx, j)
	}
	if e != nil {
		code := "export_failed"
		if e.Error() == "snapshot_limit_exceeded" {
			code = e.Error()
		}
		_, update := s.DB.Exec(ctx, "UPDATE export_jobs SET state='failed',lease_until=NULL,error_code=$3 WHERE id=$1 AND fence=$2 AND state IN ('snapshotting','rendering')", j.ID, j.Fence, code)
		if update != nil {
			return update
		}
		return e
	}
	return nil
}
func (s Service) render(ctx context.Context, j job) error {
	compression := j.Compression
	if compression == "auto" {
		compression = "none"
		if j.Bytes > 1<<20 {
			compression = "gzip"
		}
	}
	media := "application/json"
	if j.Format == "csv" {
		media = "text/csv; charset=utf-8"
	}
	filename := "export-" + j.ID + "." + j.Format
	if compression == "gzip" {
		filename += ".gz"
		media = "application/gzip"
	}
	if compression == "zip" {
		filename += ".zip"
		media = "application/zip"
	}
	artifact, e := s.Artifacts.Create(ctx, j.Owner, filename, media, func(w io.Writer) error {
		var close func() error
		switch compression {
		case "gzip":
			g := gzip.NewWriter(w)
			w = g
			close = g.Close
		case "zip":
			z := zip.NewWriter(w)
			part, e := z.CreateHeader(&zip.FileHeader{Name: "export." + j.Format, Method: zip.Deflate})
			if e != nil {
				return e
			}
			w = part
			close = z.Close
		}
		if e := s.writeRows(ctx, j, w); e != nil {
			if close != nil {
				_ = close()
			}
			return e
		}
		if close != nil {
			return close()
		}
		return nil
	})
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = s.authorize(ctx, tx, j); e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, "UPDATE export_jobs SET state='completed',processed=rows_count,artifact_id=$3,lease_until=NULL,error_code=NULL WHERE id=$1 AND fence=$2 AND state='rendering' AND lease_until>clock_timestamp()", j.ID, j.Fence, artifact)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errors.New("stale export worker")
	}
	if _, e = tx.Exec(ctx, "DELETE FROM export_rows WHERE job_id=$1", j.ID); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Service) writeRows(ctx context.Context, j job, w io.Writer) error {
	rows, e := s.DB.Query(ctx, "SELECT ordinal,value FROM export_rows WHERE job_id=$1 ORDER BY ordinal", j.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	var csvWriter *csv.Writer
	if j.Format == "csv" {
		csvWriter = csv.NewWriter(w)
		csvWriter.UseCRLF = true
		if e = csvWriter.Write([]string{"id", "title", "state", "createdAt", "publishedAt", "expiresAt", "manifestHash", "fileCount", "totalBytes"}); e != nil {
			return e
		}
	} else {
		if _, e = io.WriteString(w, "[\n"); e != nil {
			return e
		}
	}
	count := 0
	for rows.Next() {
		var ordinal int64
		var raw []byte
		if e = rows.Scan(&ordinal, &raw); e != nil {
			return e
		}
		if count%50 == 0 {
			tag, e := s.DB.Exec(ctx, "UPDATE export_jobs SET processed=$3,lease_until=clock_timestamp()+interval '3 minutes' WHERE id=$1 AND fence=$2 AND state='rendering' AND lease_until>clock_timestamp() AND EXISTS(SELECT 1 FROM users u WHERE u.id=export_jobs.owner_id AND NOT u.disabled AND u.auth_epoch=export_jobs.auth_epoch AND (export_jobs.kind='user' OR u.role='admin'))", j.ID, j.Fence, count)
			if e != nil {
				return e
			}
			if tag.RowsAffected() != 1 {
				return errors.New("export cancelled or stale")
			}
		}
		if csvWriter != nil {
			var v map[string]json.RawMessage
			if e = json.Unmarshal(raw, &v); e != nil {
				return e
			}
			record := []string{}
			for _, key := range []string{"id", "title", "state", "createdAt", "publishedAt", "expiresAt", "manifestHash", "fileCount", "totalBytes"} {
				var text string
				if string(v[key]) != "null" {
					if e = json.Unmarshal(v[key], &text); e != nil {
						text = string(v[key])
					}
				}
				record = append(record, reports.SafeCSV(text))
			}
			if e = csvWriter.Write(record); e != nil {
				return e
			}
		} else {
			if count > 0 {
				if _, e = io.WriteString(w, ",\n"); e != nil {
					return e
				}
			}
			if _, e = w.Write(raw); e != nil {
				return e
			}
		}
		count++
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if count != j.Count {
		return fmt.Errorf("snapshot rows incomplete")
	}
	if csvWriter != nil {
		csvWriter.Flush()
		return csvWriter.Error()
	}
	_, e = io.WriteString(w, "\n]\n")
	return e
}
func (s Service) Cleanup(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, "DELETE FROM export_rows WHERE job_id IN (SELECT id FROM export_jobs WHERE state='cancelled' OR (state='failed' AND created_at<clock_timestamp()-interval '1 day'))")
	return e
}

// Hold the current owner authorization through the snapshot/final commit.
func (s Service) authorize(ctx context.Context, tx pgx.Tx, j job) error {
	var ok string
	return tx.QueryRow(ctx, "SELECT u.id::text FROM users u JOIN export_jobs e ON e.owner_id=u.id WHERE e.id=$1 AND NOT u.disabled AND u.auth_epoch=e.auth_epoch AND (e.kind='user' OR u.role='admin') FOR SHARE OF u", j.ID).Scan(&ok)
}

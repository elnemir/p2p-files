package transfer

import (
	"context"
	"errors"
	"filex.local/filex/internal/licensing"
	"time"

	"filex.local/filex/internal/config"
	"filex.local/filex/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("state or revision conflict")
)

type Store struct {
	db       *pgxpool.Pool
	settings config.Settings
}

func New(db *pgxpool.Pool, settings config.Settings) *Store {
	return &Store{db: db, settings: settings}
}

type Draft struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}
type Upload struct {
	ID        string
	ObjectKey string
	Revision  int64
}
type Published struct {
	ID                string    `json:"id"`
	Revision          int64     `json:"revision"`
	PublishedAt       time.Time `json:"publishedAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
	AbsoluteExpiresAt time.Time `json:"absoluteExpiresAt"`
	Manifest          []byte    `json:"-"`
	ManifestHash      string    `json:"manifestHash"`
}

func (s *Store) CreateDraft(ctx context.Context, owner string) (Draft, error) {
	if !domain.ValidID(owner) {
		return Draft{}, ErrNotFound
	}
	d := Draft{ID: domain.NewID(), Revision: 1}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(context.Background())
	if _, err = licensing.Guard(ctx, tx, ""); err != nil {
		return Draft{}, err
	}
	_, err = tx.Exec(ctx, "INSERT INTO transfers(id,owner_id) VALUES($1,$2)", d.ID, owner)
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == "23503" {
		return Draft{}, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}

func lockDraft(ctx context.Context, tx pgx.Tx, owner, id string) (int64, error) {
	var state string
	var frozen bool
	var revision int64
	err := tx.QueryRow(ctx, "SELECT state,revision,frozen FROM transfers WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner).Scan(&state, &revision, &frozen)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if state != "draft" || frozen {
		return 0, ErrConflict
	}
	return revision, nil
}

func (s *Store) ReserveUpload(ctx context.Context, owner, id, path string, size int64) (Upload, error) {
	return s.ReserveUploadWith(ctx, owner, id, path, size, nil)
}
func (s *Store) ReserveUploadWith(ctx context.Context, owner, id, path string, size int64, hook func(pgx.Tx, string) error) (Upload, error) {
	return s.reserveUpload(ctx, owner, id, path, size, false, hook)
}
func (s *Store) ReserveStaged(ctx context.Context, owner, id, path string, size int64, hook func(pgx.Tx, string) error) (Upload, error) {
	return s.reserveUpload(ctx, owner, id, path, size, true, hook)
}
func (s *Store) reserveUpload(ctx context.Context, owner, id, path string, size int64, staged bool, hook func(pgx.Tx, string) error) (Upload, error) {
	if !domain.ValidID(owner) || !domain.ValidID(id) {
		return Upload{}, ErrNotFound
	}
	key, err := domain.PathKey(path, s.settings.MaxPathBytes)
	if err != nil || size < 0 {
		return Upload{}, domain.ErrInvalid
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Upload{}, err
	}
	defer tx.Rollback(context.Background())
	revision, err := lockDraft(ctx, tx, owner, id)
	if err != nil {
		return Upload{}, err
	}
	if err = CheckDAVLocks(ctx, tx, id, key, false); err != nil {
		return Upload{}, err
	}
	var existingTarget bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND path_key=$2 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted'))", id, key).Scan(&existingTarget); err != nil {
		return Upload{}, err
	}
	if !existingTarget {
		if err = CheckDAVMembership(ctx, tx, id, key); err != nil {
			return Upload{}, err
		}
	}
	var count int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM files WHERE transfer_id=$1 AND staged_until IS NULL AND state NOT IN ('cancelled','deleted')", id).Scan(&count); err != nil {
		return Upload{}, err
	}
	if count >= s.settings.MaxFiles && !staged {
		return Upload{}, domain.ErrInvalid
	}
	rows, err := tx.Query(ctx, "SELECT path_key FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted')", id)
	if err != nil {
		return Upload{}, err
	}
	for rows.Next() {
		var existing string
		if err = rows.Scan(&existing); err != nil {
			rows.Close()
			return Upload{}, err
		}
		if pathsConflict(key, existing) && !(staged && key == existing) {
			rows.Close()
			return Upload{}, ErrConflict
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Upload{}, err
	}
	if err = licensing.TransferQuota(ctx, tx, id, key, licensing.ReservedBytes(ctx, size)); err != nil {
		return Upload{}, err
	}
	licensePath := key
	u := Upload{ID: domain.NewID(), Revision: revision + 1}
	u.ObjectKey = u.ID
	if staged {
		key = "#staged:" + u.ID
	}
	if _, err = tx.Exec(ctx, "INSERT INTO files(id,transfer_id,path,path_key,size,object_key,license_bytes,license_path_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", u.ID, id, path, key, size, u.ObjectKey, licensing.ReservedBytes(ctx, size), licensePath); err != nil {
		return Upload{}, err
	}
	if staged {
		if _, err = tx.Exec(ctx, "UPDATE files SET staged_until=clock_timestamp()+interval '5 minutes' WHERE id=$1", u.ID); err != nil {
			return Upload{}, err
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", id); err != nil {
		return Upload{}, err
	}
	if hook != nil {
		if err = hook(tx, u.ID); err != nil {
			return Upload{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Upload{}, err
	}
	return u, nil
}

func pathsConflict(a, b string) bool {
	return a == b || (len(a) > len(b) && a[:len(b)] == b && a[len(b)] == '/') || (len(b) > len(a) && b[:len(a)] == a && b[len(a)] == '/')
}

// BeginFinalize records the boundary between durable upload and server-side validation.
func (s *Store) BeginFinalize(ctx context.Context, owner, id, fileID string) (int64, error) {
	return s.BeginFinalizeWith(ctx, owner, id, fileID, nil)
}

func (s *Store) BeginFinalizeWith(ctx context.Context, owner, id, fileID string, guard func(context.Context, pgx.Tx) error) (int64, error) {
	if !domain.ValidID(owner) || !domain.ValidID(id) || !domain.ValidID(fileID) {
		return 0, ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())
	revision, err := lockDraft(ctx, tx, owner, id)
	if err != nil {
		return 0, err
	}
	if _, err = licensing.Guard(ctx, tx, ""); err != nil {
		return 0, err
	}

	var filePath string
	if err = tx.QueryRow(ctx, "SELECT path FROM files WHERE id=$1 AND transfer_id=$2", fileID, id).Scan(&filePath); err != nil {
		return 0, err
	}
	fileKey, err := domain.PathKey(filePath, s.settings.MaxPathBytes)
	if err != nil {
		return 0, err
	}
	if err = CheckDAVLocks(ctx, tx, id, fileKey, false); err != nil {
		return 0, err
	}
	if guard != nil {
		if err = guard(ctx, tx); err != nil {
			return 0, err
		}
	}
	var state string
	err = tx.QueryRow(ctx, "SELECT state FROM files WHERE id=$1 AND transfer_id=$2 FOR UPDATE", fileID, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if state == "finalizing" {
		return revision, tx.Commit(ctx)
	}
	if state != "uploading" {
		return 0, ErrConflict
	}
	if _, err = tx.Exec(ctx, "UPDATE files SET state='finalizing' WHERE id=$1", fileID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", id); err != nil {
		return 0, err
	}
	return revision + 1, tx.Commit(ctx)
}

// Finalize is a trusted worker boundary; no client-facing endpoint may set a checksum or ready state.
func (s *Store) Finalize(ctx context.Context, owner, id, fileID, sha256 string) (int64, error) {
	return s.FinalizeWith(ctx, owner, id, fileID, sha256, nil)
}

func (s *Store) FinalizeWith(ctx context.Context, owner, id, fileID, sha256 string, guard func(context.Context, pgx.Tx) error) (int64, error) {
	if !domain.ValidID(owner) || !domain.ValidID(id) || !domain.ValidID(fileID) {
		return 0, ErrNotFound
	}
	if !domain.ValidChecksum(sha256) {
		return 0, domain.ErrInvalid
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())
	revision, err := lockDraft(ctx, tx, owner, id)
	if err != nil {
		return 0, err
	}
	if _, err = licensing.Guard(ctx, tx, ""); err != nil {
		return 0, err
	}

	var filePath string
	if err = tx.QueryRow(ctx, "SELECT path FROM files WHERE id=$1 AND transfer_id=$2", fileID, id).Scan(&filePath); err != nil {
		return 0, err
	}
	fileKey, err := domain.PathKey(filePath, s.settings.MaxPathBytes)
	if err != nil {
		return 0, err
	}
	if err = CheckDAVLocks(ctx, tx, id, fileKey, false); err != nil {
		return 0, err
	}
	if guard != nil {
		if err = guard(ctx, tx); err != nil {
			return 0, err
		}
	}
	var state string
	var checksum *string
	err = tx.QueryRow(ctx, "SELECT state,sha256 FROM files WHERE id=$1 AND transfer_id=$2 FOR UPDATE", fileID, id).Scan(&state, &checksum)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if state == "ready" && checksum != nil && *checksum == sha256 {
		return revision, tx.Commit(ctx)
	}
	if state != "finalizing" {
		return 0, ErrConflict
	}
	if _, err = tx.Exec(ctx, "UPDATE files SET state='ready',sha256=$1 WHERE id=$2", sha256, fileID); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, "UPDATE transfers SET revision=revision+1 WHERE id=$1", id); err != nil {
		return 0, err
	}
	return revision + 1, tx.Commit(ctx)
}

func (s *Store) Publish(ctx context.Context, owner, id string, expectedRevision, ttlSeconds int64) (Published, error) {
	return s.PublishWith(ctx, owner, id, expectedRevision, ttlSeconds, nil)
}
func (s *Store) PublishWith(ctx context.Context, owner, id string, expectedRevision, ttlSeconds int64, hook func(pgx.Tx) error) (Published, error) {
	return s.PublishWithRequest(ctx, owner, id, expectedRevision, ttlSeconds, "", hook)
}
func (s *Store) PublishWithRequest(ctx context.Context, owner, id string, expectedRevision, ttlSeconds int64, fingerprint string, hook func(pgx.Tx) error) (Published, error) {
	if !domain.ValidID(owner) || !domain.ValidID(id) {
		return Published{}, ErrNotFound
	}
	if expectedRevision < 1 {
		return Published{}, domain.ErrInvalid
	}
	if _, _, err := domain.Expiry(time.Time{}, ttlSeconds, s.settings.TransferMaxTTLSeconds); err != nil {
		return Published{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Published{}, err
	}
	defer tx.Rollback(context.Background())
	var state string
	var revision int64
	var publishedRevision, publishedTTL *int64
	var publishedRequest string
	err = tx.QueryRow(ctx, "SELECT state,revision,published_revision,published_ttl_seconds,COALESCE(publish_request_hash,'') FROM transfers WHERE id=$1 AND owner_id=$2 FOR UPDATE", id, owner).Scan(&state, &revision, &publishedRevision, &publishedTTL, &publishedRequest)
	if errors.Is(err, pgx.ErrNoRows) {
		return Published{}, ErrNotFound
	}
	if err != nil {
		return Published{}, err
	}
	if state == "active" && publishedRequest == fingerprint && publishedRevision != nil && *publishedRevision == expectedRevision && publishedTTL != nil && *publishedTTL == ttlSeconds {
		p, err := readPublished(ctx, tx, id)
		if err != nil {
			return Published{}, err
		}
		return p, tx.Commit(ctx)
	}
	if state != "draft" || revision != expectedRevision {
		return Published{}, ErrConflict
	}
	if err = CheckDAVLocks(ctx, tx, id, "", true); err != nil {
		return Published{}, err
	}
	if err = licensing.TransferQuota(ctx, tx, id, "", 0); err != nil {
		return Published{}, err
	}
	var staging bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM files WHERE transfer_id=$1 AND staged_until IS NOT NULL AND state NOT IN ('cancelled','deleted'))", id).Scan(&staging); err != nil {
		return Published{}, err
	}
	if staging {
		return Published{}, ErrConflict
	}
	rows, err := tx.Query(ctx, "SELECT id::text,path,size,sha256,state FROM files WHERE transfer_id=$1 AND state NOT IN ('cancelled','deleted') ORDER BY id", id)
	if err != nil {
		return Published{}, err
	}
	files := []domain.File{}
	for rows.Next() {
		var f domain.File
		var hash *string
		var fileState string
		if err = rows.Scan(&f.ID, &f.Path, &f.Size, &hash, &fileState); err != nil {
			rows.Close()
			return Published{}, err
		}
		if fileState != "ready" || hash == nil {
			rows.Close()
			return Published{}, ErrConflict
		}
		f.SHA256 = *hash
		files = append(files, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return Published{}, err
	}
	manifest, hash, err := domain.BuildManifest(files, s.settings.MaxFiles, s.settings.MaxPathBytes)
	if err != nil {
		return Published{}, err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return Published{}, err
	}
	expires, absolute, err := domain.Expiry(now, ttlSeconds, s.settings.TransferMaxTTLSeconds)
	if err != nil {
		return Published{}, err
	}
	_, err = tx.Exec(ctx, "UPDATE transfers SET state='active',revision=revision+1,published_revision=$2,published_ttl_seconds=$3,published_at=$4,expires_at=$5,absolute_expires_at=$6,manifest=$7,manifest_hash=$8,publish_request_hash=NULLIF($9,'') WHERE id=$1", id, expectedRevision, ttlSeconds, now, expires, absolute, manifest, hash, fingerprint)
	if err != nil {
		return Published{}, err
	}
	if hook != nil {
		if err = hook(tx); err != nil {
			return Published{}, err
		}
	}
	// The deduplication key and business state share one transaction; delivery is a later slice.
	_, err = tx.Exec(ctx, "INSERT INTO outbox(aggregate_id,event_type,deduplication_key,payload) VALUES($1,'transfer.published',$2,jsonb_build_object('transferId',$3::text,'manifestHash',$4::text))", id, "transfer.published:"+id, id, hash)
	if err != nil {
		return Published{}, err
	}
	p, err := readPublished(ctx, tx, id)
	if err != nil {
		return Published{}, err
	}
	return p, tx.Commit(ctx)
}
func readPublished(ctx context.Context, tx pgx.Tx, id string) (Published, error) {
	p := Published{ID: id}
	err := tx.QueryRow(ctx, "SELECT revision,published_at,expires_at,absolute_expires_at,manifest,manifest_hash FROM transfers WHERE id=$1", id).Scan(&p.Revision, &p.PublishedAt, &p.ExpiresAt, &p.AbsoluteExpiresAt, &p.Manifest, &p.ManifestHash)
	return p, err
}

// Package issuer belongs exclusively to the vendor deployment, never customer runtime.
package issuer

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

//go:embed schema.sql
var schema string

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185040)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, schema); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func digest(v []byte) string { h := sha256.Sum256(v); return hex.EncodeToString(h[:]) }

type Input struct {
	IdempotencyKey string           `json:"idempotencyKey"`
	Claims         licensing.Claims `json:"claims"`
}
type Issuance struct {
	ID        string           `json:"id"`
	State     string           `json:"state"`
	Claims    licensing.Claims `json:"claims"`
	Hash      string           `json:"sha256"`
	CreatedAt time.Time        `json:"createdAt"`
	KeyID     string           `json:"keyId"`
}

func Issue(ctx context.Context, db *pgxpool.Pool, actor string, in Input) (Issuance, error) {
	out := Issuance{}
	if !domain.ValidID(in.IdempotencyKey) || in.Claims.Validate() != nil || in.Claims.IssuedAt.After(time.Now().Add(time.Minute)) {
		return out, licensing.ErrInvalid
	}
	raw, e := json.Marshal(in.Claims)
	if e != nil {
		return out, e
	}
	hash := digest(raw)
	tx, e := db.Begin(ctx)
	if e != nil {
		return out, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185041)"); e != nil {
		return out, e
	}
	var previousHash string
	e = tx.QueryRow(ctx, "SELECT id::text,request_hash FROM issuer_issuances WHERE idempotency_key=$1", in.IdempotencyKey).Scan(&out.ID, &previousHash)
	if e == nil {
		if previousHash != hash {
			return out, licensing.ErrConflict
		}
		return read(ctx, tx, out.ID)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	var rev int64
	var install, customer string
	e = tx.QueryRow(ctx, "SELECT revision,installation_id::text,customer_id FROM issuer_issuances WHERE entitlement_id=$1 ORDER BY revision DESC LIMIT 1", in.Claims.EntitlementID).Scan(&rev, &install, &customer)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return out, e
	}
	if in.Claims.Revision != rev+1 || (rev > 0 && (install != in.Claims.InstallationID || customer != in.Claims.CustomerID)) {
		return out, licensing.ErrConflict
	}
	out.ID = in.Claims.LicenseID
	_, e = tx.Exec(ctx, `INSERT INTO issuer_issuances(id,idempotency_key,request_hash,entitlement_id,revision,installation_id,customer_id,claims,actor) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, out.ID, in.IdempotencyKey, hash, in.Claims.EntitlementID, in.Claims.Revision, in.Claims.InstallationID, in.Claims.CustomerID, raw, actor)
	if e != nil {
		var duplicate *pgconn.PgError
		if errors.As(e, &duplicate) && duplicate.Code == "23505" {
			return out, licensing.ErrConflict
		}
		return out, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO issuer_outbox(issuance_id) VALUES($1)", out.ID); e != nil {
		return out, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO issuer_audit(actor,action,issuance_id) VALUES($1,'approved',$2)", actor, out.ID); e != nil {
		return out, e
	}
	out, e = read(ctx, tx, out.ID)
	if e != nil {
		return out, e
	}
	return out, tx.Commit(ctx)
}

type reader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func read(ctx context.Context, db reader, id string) (Issuance, error) {
	var out Issuance
	var raw []byte
	e := db.QueryRow(ctx, "SELECT id::text,state,claims,artifact_hash,created_at,key_id FROM issuer_issuances WHERE id=$1", id).Scan(&out.ID, &out.State, &raw, &out.Hash, &out.CreatedAt, &out.KeyID)
	if e != nil {
		return out, e
	}
	e = licensing.Decode(raw, &out.Claims)
	return out, e
}
func List(ctx context.Context, db *pgxpool.Pool) ([]Issuance, error) {
	rows, e := db.Query(ctx, "SELECT id::text,state,claims,artifact_hash,created_at,key_id FROM issuer_issuances ORDER BY created_at DESC,id DESC LIMIT 200")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Issuance{}
	for rows.Next() {
		var v Issuance
		var raw []byte
		if e = rows.Scan(&v.ID, &v.State, &raw, &v.Hash, &v.CreatedAt, &v.KeyID); e != nil {
			return nil, e
		}
		if e = licensing.Decode(raw, &v.Claims); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// SignNext holds the queue claim and artifact commit in one short transaction.
// No signature leaves the process before commit; a crash replays identical bytes.
func SignNext(ctx context.Context, db *pgxpool.Pool, kid string, key ed25519.PrivateKey) (bool, error) {
	tx, e := db.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(context.Background())
	var id string
	var raw []byte
	e = tx.QueryRow(ctx, `SELECT i.id::text,i.claims FROM issuer_outbox o JOIN issuer_issuances i ON i.id=o.issuance_id WHERE o.completed_at IS NULL ORDER BY i.created_at FOR UPDATE OF o,i SKIP LOCKED LIMIT 1`).Scan(&id, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var c licensing.Claims
	if e = licensing.Decode(raw, &c); e != nil {
		return false, e
	}
	signed, e := licensing.Sign(c, kid, key)
	if e != nil {
		return false, e
	}
	if _, e = licensing.Verify(signed, map[string]ed25519.PublicKey{kid: key.Public().(ed25519.PublicKey)}); e != nil {
		return false, e
	}
	_, e = tx.Exec(ctx, "UPDATE issuer_issuances SET state='issued',envelope=$2,artifact_hash=$3,key_id=$4,issued_at=clock_timestamp() WHERE id=$1 AND state='approved'", id, signed, digest([]byte(signed)), kid)
	if e != nil {
		return false, e
	}
	if _, e = tx.Exec(ctx, "UPDATE issuer_outbox SET completed_at=clock_timestamp() WHERE issuance_id=$1", id); e != nil {
		return false, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO issuer_audit(actor,action,issuance_id) VALUES('isolated_signer','issued',$1)", id); e != nil {
		return false, e
	}
	return true, tx.Commit(ctx)
}

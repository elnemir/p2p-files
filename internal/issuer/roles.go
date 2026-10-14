package issuer

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProvisionRoles is an explicit owner-only migration operation. The web and
// signer processes run with distinct, non-DDL PostgreSQL roles afterwards.
func ProvisionRoles(ctx context.Context, db *pgxpool.Pool, appPassword, signerPassword string) error {
	if len(appPassword) < 24 || len(signerPassword) < 24 {
		return errors.New("issuer database passwords must be at least 24 characters")
	}
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	for role, password := range map[string]string{"filex_issuer_app": appPassword, "filex_issuer_signer": signerPassword} {
		var exists bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)", role).Scan(&exists); e != nil {
			return e
		}
		verb := "CREATE"
		if exists {
			verb = "ALTER"
		}
		var sql string
		if e = tx.QueryRow(ctx, "SELECT format('%s ROLE %I LOGIN PASSWORD %L', $1::text,$2::text,$3::text)", verb, role, password).Scan(&sql); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, sql); e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC;
 GRANT USAGE ON SCHEMA public TO filex_issuer_app,filex_issuer_signer;
 GRANT SELECT,INSERT,DELETE ON issuer_flows,issuer_sessions TO filex_issuer_app;
 GRANT SELECT,INSERT ON issuer_setup TO filex_issuer_app;
 GRANT SELECT,INSERT ON issuer_issuances,issuer_outbox TO filex_issuer_app;
 GRANT SELECT ON issuer_issuances,issuer_outbox TO filex_issuer_signer;
 GRANT UPDATE(state,envelope,artifact_hash,key_id,issued_at) ON issuer_issuances TO filex_issuer_signer;
 GRANT UPDATE(completed_at) ON issuer_outbox TO filex_issuer_signer;
 GRANT SELECT,INSERT ON issuer_audit TO filex_issuer_app,filex_issuer_signer;
 GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA public TO filex_issuer_app,filex_issuer_signer;`)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

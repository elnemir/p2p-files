package identity

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/auth"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/keyring"
	"filex.local/filex/internal/transfer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB       *pgxpool.Pool
	Keys     keyring.Ring
	AuditKey []byte
}
type reader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s Store) provider(ctx context.Context, db reader, id string, secret bool) (Provider, error) {
	var p Provider
	var raw, box []byte
	var keyID string
	e := db.QueryRow(ctx, "SELECT id::text,revision,config,secret_key_id::text,secret_box FROM identity_providers WHERE id=$1", id).Scan(&p.ID, &p.Revision, &raw, &keyID, &box)
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(raw, &p.Config); e != nil {
		return p, e
	}
	if secret {
		key, e := s.Keys.Get(ctx, keyID)
		if e != nil {
			return p, e
		}
		plain, e := cryptobox.Open(key, "filex/identity_provider/v1/"+id, box)
		if e != nil {
			return p, e
		}
		p.secret = string(plain)
		p.HasSecret = p.secret != ""
	}
	return p, nil
}
func (s Store) Get(ctx context.Context, id string) (Provider, error) {
	if !domain.ValidID(id) {
		return Provider{}, auth.ErrInvalid
	}
	return s.provider(ctx, s.DB, id, true)
}
func (s Store) List(ctx context.Context) ([]Provider, error) {
	rows, e := s.DB.Query(ctx, "SELECT id::text FROM identity_providers ORDER BY created_at,id")
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	result := []Provider{}
	for _, id := range ids {
		p, e := s.Get(ctx, id)
		if e != nil {
			return nil, e
		}
		result = append(result, p)
	}
	return result, nil
}
func authority(ctx context.Context, tx pgx.Tx, p auth.Principal, fresh, admin bool) error {
	if p.Bearer || p.SessionID == "" {
		return auth.ErrForbidden
	}
	var id string
	e := tx.QueryRow(ctx, `SELECT u.id::text FROM users u JOIN web_sessions w ON w.user_id=u.id WHERE u.id::text=$1 AND w.id::text=$2 AND u.auth_epoch=$3 AND w.auth_epoch=u.auth_epoch AND NOT u.disabled AND w.revoked_at IS NULL AND w.expires_at>clock_timestamp() AND (NOT $4 OR w.reauth_at>clock_timestamp()-interval '5 minutes') AND (NOT $5 OR u.role='admin') FOR SHARE OF u,w`, p.ID, p.SessionID, p.Epoch, fresh, admin).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return auth.ErrForbidden
	}
	return e
}
func (s Store) Apply(ctx context.Context, p auth.Principal, id string, in Input) (Provider, error) {
	if in.Config.AllowedEmailDomains == nil {
		in.Config.AllowedEmailDomains = []string{}
	}
	if in.Config.PrivateCIDRs == nil {
		in.Config.PrivateCIDRs = []string{}
	}
	if in.Config.ExtraOrigins == nil {
		in.Config.ExtraOrigins = []string{}
	}
	create := id == ""
	if create {
		id = domain.NewID()
	} else if !domain.ValidID(id) {
		return Provider{}, auth.ErrInvalid
	}
	secret := ""
	if !create {
		prev, e := s.Get(ctx, id)
		if e != nil {
			return Provider{}, e
		}
		secret = prev.secret
	}
	if in.Secret != nil {
		secret = *in.Secret
	}
	if e := in.Config.Validate(secret); e != nil {
		return Provider{}, e
	}
	keyID, key, e := s.Keys.Active(ctx, "configuration")
	if e != nil {
		return Provider{}, e
	}
	box, e := s.Keys.Seal(ctx, keyID, key, "filex/identity_provider/v1/"+id, []byte(secret))
	if e != nil {
		return Provider{}, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Provider{}, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185021)"); e != nil {
		return Provider{}, e
	}
	if e = authority(ctx, tx, p, true, true); e != nil {
		return Provider{}, e
	}
	if e = keyring.RequireActive(ctx, tx, keyID); e != nil {
		return Provider{}, e
	}
	revision := int64(1)
	if create {
		if in.ExpectedRevision != 0 {
			return Provider{}, transfer.ErrConflict
		}
		var n int
		if e = tx.QueryRow(ctx, "SELECT count(*) FROM identity_providers").Scan(&n); e != nil {
			return Provider{}, e
		}
		if n >= 32 {
			return Provider{}, auth.ErrInvalid
		}
	} else {
		prev, e := s.provider(ctx, tx, id, false)
		if e != nil {
			return Provider{}, e
		}
		if prev.Revision != in.ExpectedRevision {
			return Provider{}, transfer.ErrConflict
		}
		if !prev.Config.SameAuthority(in.Config) {
			return Provider{}, auth.ErrInvalid
		}
		revision = prev.Revision + 1
	}
	raw, e := json.Marshal(in.Config)
	if e != nil {
		return Provider{}, e
	}
	if create {
		_, e = tx.Exec(ctx, "INSERT INTO identity_providers(id,config,secret_key_id,secret_box) VALUES($1,$2,$3,$4)", id, raw, keyID, box)
	} else {
		_, e = tx.Exec(ctx, "UPDATE identity_providers SET revision=$2,config=$3,secret_key_id=$4,secret_box=$5 WHERE id=$1", id, revision, raw, keyID, box)
	}
	if e != nil {
		return Provider{}, e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM identity_flows WHERE provider_id=$1", id); e != nil {
		return Provider{}, e
	}
	if _, e = tx.Exec(ctx, "UPDATE web_sessions SET revoked_at=clock_timestamp() WHERE identity_provider_id=$1 AND revoked_at IS NULL", id); e != nil {
		return Provider{}, e
	}
	if e = audit.Append(ctx, tx, s.AuditKey, p.ID, "identity.provider_updated", id); e != nil {
		return Provider{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Provider{}, e
	}
	return Provider{ID: id, Revision: revision, Config: in.Config, HasSecret: secret != ""}, nil
}

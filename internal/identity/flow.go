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
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5"
	"time"
)

type Flow struct {
	ProviderID string `json:"providerId"`
	Revision   int64  `json:"revision"`
	Origin     string `json:"origin"`
	Purpose    string `json:"purpose"`
	ActorID    string `json:"actorId"`
	SessionID  string `json:"sessionId"`
	Epoch      int64  `json:"epoch"`
	Nonce      string `json:"nonce"`
	Verifier   string `json:"verifier"`
	RequestID  string `json:"requestId"`
	Proof      Proof  `json:"proof"`
}

func NewFlow(p Provider, origin, purpose string, actor auth.Principal) (Flow, error) {
	if purpose != "login" && purpose != "link" && purpose != "reauth" {
		return Flow{}, auth.ErrInvalid
	}
	if purpose != "login" && (actor.Bearer || actor.SessionID == "") {
		return Flow{}, auth.ErrCredentials
	}
	return Flow{ProviderID: p.ID, Revision: p.Revision, Origin: origin, Purpose: purpose, ActorID: actor.ID, SessionID: actor.SessionID, Epoch: actor.Epoch, Nonce: auth.Token(), Verifier: auth.Token()}, nil
}
func (f Flow) actor() auth.Principal {
	return auth.Principal{ID: f.ActorID, SessionID: f.SessionID, Epoch: f.Epoch}
}
func (s Store) sealFlow(ctx context.Context, state string, f Flow) (string, []byte, error) {
	id, key, e := s.Keys.Active(ctx, "configuration")
	if e != nil {
		return "", nil, e
	}
	raw, e := json.Marshal(f)
	if e != nil {
		return "", nil, e
	}
	box, e := s.Keys.Seal(ctx, id, key, "filex/identity_flow/v1/"+auth.Digest(state), raw)
	return id, box, e
}
func (s Store) SaveFlow(ctx context.Context, state, binding string, f Flow) error {
	id, box, e := s.sealFlow(ctx, state, f)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185021)"); e != nil {
		return e
	}
	p, e := s.provider(ctx, tx, f.ProviderID, false)
	if e != nil {
		return e
	}
	if !p.Config.Enabled || p.Revision != f.Revision {
		return auth.ErrCredentials
	}
	if f.Purpose != "login" {
		if e = authority(ctx, tx, f.actor(), f.Purpose == "link", false); e != nil {
			return e
		}
	}
	if e = keyring.RequireActive(ctx, tx, id); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "DELETE FROM identity_flows WHERE expires_at<=clock_timestamp() OR binding_hash=$1", auth.Digest(binding)); e != nil {
		return e
	}
	var n int
	if e = tx.QueryRow(ctx, "SELECT count(*) FROM identity_flows").Scan(&n); e != nil {
		return e
	}
	if n >= 10000 {
		return auth.ErrRate
	}
	if _, e = tx.Exec(ctx, "INSERT INTO identity_flows(state_hash,binding_hash,provider_id,revision,key_id,box) VALUES($1,$2,$3,$4,$5,$6)", auth.Digest(state), auth.Digest(binding), f.ProviderID, f.Revision, id, box); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s Store) Flow(ctx context.Context, state string) (Flow, error) {
	if len(state) != 43 {
		return Flow{}, auth.ErrCredentials
	}
	var box []byte
	var id string
	e := s.DB.QueryRow(ctx, "SELECT key_id::text,box FROM identity_flows WHERE state_hash=$1 AND expires_at>clock_timestamp() AND NOT verified", auth.Digest(state)).Scan(&id, &box)
	if errors.Is(e, pgx.ErrNoRows) {
		return Flow{}, auth.ErrCredentials
	}
	if e != nil {
		return Flow{}, e
	}
	return s.openFlow(ctx, s.DB, state, id, box)
}
func (s Store) openFlow(ctx context.Context, db reader, state, id string, box []byte) (Flow, error) {
	var f Flow
	var wrapped []byte
	e := db.QueryRow(ctx, "SELECT wrapped FROM key_versions WHERE id=$1", id).Scan(&wrapped)
	if e != nil {
		return f, e
	}
	key, e := cryptobox.Open(s.Keys.KEK, "filex/keys/v1/"+id, wrapped)
	if e != nil {
		return f, e
	}
	raw, e := cryptobox.Open(key, "filex/identity_flow/v1/"+auth.Digest(state), box)
	if e != nil {
		return f, e
	}
	e = json.Unmarshal(raw, &f)
	return f, e
}
func (s Store) VerifyFlow(ctx context.Context, state string, f Flow) error {
	id, box, e := s.sealFlow(ctx, state, f)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = keyring.RequireActive(ctx, tx, id); e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, "UPDATE identity_flows SET key_id=$2,box=$3,verified=true WHERE state_hash=$1 AND expires_at>clock_timestamp() AND NOT verified", auth.Digest(state), id, box)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return auth.ErrCredentials
	}
	return tx.Commit(ctx)
}
func (s Store) Complete(ctx context.Context, state, binding, origin string, ttl int64) (auth.Session, error) {
	if len(state) != 43 || len(binding) != 43 {
		return auth.Session{}, auth.ErrCredentials
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return auth.Session{}, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185021)"); e != nil {
		return auth.Session{}, e
	}
	var box []byte
	var id string
	e = tx.QueryRow(ctx, "DELETE FROM identity_flows WHERE state_hash=$1 AND binding_hash=$2 AND verified AND expires_at>clock_timestamp() RETURNING key_id::text,box", auth.Digest(state), auth.Digest(binding)).Scan(&id, &box)
	if errors.Is(e, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrCredentials
	}
	if e != nil {
		return auth.Session{}, e
	}
	f, e := s.openFlow(ctx, tx, state, id, box)
	if e != nil {
		return auth.Session{}, e
	}
	if f.Origin != origin {
		return auth.Session{}, auth.ErrCredentials
	}
	result, e := s.finish(ctx, tx, f, ttl)
	if e != nil {
		return auth.Session{}, e
	}
	return result, tx.Commit(ctx)
}
func (s Store) LDAPSession(ctx context.Context, f Flow, ttl int64) (auth.Session, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return auth.Session{}, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185021)"); e != nil {
		return auth.Session{}, e
	}
	result, e := s.finish(ctx, tx, f, ttl)
	if e != nil {
		return auth.Session{}, e
	}
	return result, tx.Commit(ctx)
}
func (s Store) finish(ctx context.Context, tx pgx.Tx, f Flow, ttl int64) (auth.Session, error) {
	if _, e := licensing.Guard(ctx, tx, "federated_auth"); e != nil {
		return auth.Session{}, e
	}
	p, e := s.provider(ctx, tx, f.ProviderID, false)
	if e != nil {
		return auth.Session{}, e
	}
	if !p.Config.Enabled || p.Revision != f.Revision {
		return auth.Session{}, auth.ErrCredentials
	}
	proof, e := p.Config.CheckProof(f.Proof)
	if e != nil {
		return auth.Session{}, e
	}
	if f.Purpose == "reauth" && (proof.AuthenticatedAt < time.Now().Add(-5*time.Minute).Unix() || proof.AuthenticatedAt > time.Now().Add(time.Minute).Unix()) {
		return auth.Session{}, auth.ErrCredentials
	}
	if f.Purpose != "login" {
		if e = authority(ctx, tx, f.actor(), f.Purpose == "link", false); e != nil {
			return auth.Session{}, e
		}
	}
	var userID string
	e = tx.QueryRow(ctx, "SELECT user_id::text FROM external_identities WHERE provider_id=$1 AND subject=$2", p.ID, proof.Subject).Scan(&userID)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return auth.Session{}, e
	}
	if f.Purpose == "link" {
		if userID != "" && userID != f.ActorID {
			return auth.Session{}, auth.ErrCredentials
		}
		if userID == "" {
			if _, e = tx.Exec(ctx, "INSERT INTO external_identities(provider_id,subject,user_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", p.ID, proof.Subject, f.ActorID); e != nil {
				return auth.Session{}, e
			}
			if e = tx.QueryRow(ctx, "SELECT user_id::text FROM external_identities WHERE provider_id=$1 AND subject=$2", p.ID, proof.Subject).Scan(&userID); e != nil || userID != f.ActorID {
				return auth.Session{}, auth.ErrCredentials
			}
			if e = audit.Append(ctx, tx, s.AuditKey, f.ActorID, "identity.linked", p.ID); e != nil {
				return auth.Session{}, e
			}
		}
	} else if f.Purpose == "reauth" {
		if userID == "" || userID != f.ActorID {
			return auth.Session{}, auth.ErrCredentials
		}
	} else if f.Purpose == "login" {
		if userID == "" {
			if !p.Config.AllowRegistration {
				return auth.Session{}, auth.ErrCredentials
			}
			if e = licensing.UserSlot(ctx, tx, false); e != nil {
				return auth.Session{}, e
			}
			userID = domain.NewID()
			tag, e := tx.Exec(ctx, "INSERT INTO users(id,email,role) VALUES($1,$2,'user') ON CONFLICT DO NOTHING", userID, proof.Email)
			if e != nil {
				return auth.Session{}, e
			}
			if tag.RowsAffected() != 1 {
				return auth.Session{}, auth.ErrCredentials
			}
			if _, e = tx.Exec(ctx, "INSERT INTO external_identities(provider_id,subject,user_id) VALUES($1,$2,$3)", p.ID, proof.Subject, userID); e != nil {
				return auth.Session{}, e
			}
			if e = audit.Append(ctx, tx, s.AuditKey, userID, "identity.registered", p.ID); e != nil {
				return auth.Session{}, e
			}
		}
	} else {
		return auth.Session{}, auth.ErrInvalid
	}
	var user auth.Principal
	e = tx.QueryRow(ctx, "SELECT id::text,email,role,locale,auth_epoch FROM users WHERE id=$1 AND NOT disabled FOR UPDATE", userID).Scan(&user.ID, &user.Email, &user.Role, &user.Locale, &user.Epoch)
	if errors.Is(e, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrCredentials
	}
	if e != nil {
		return auth.Session{}, e
	}
	// Account authority is local; claims never assign or upgrade privileges.
	if f.Purpose != "login" && user.Epoch != f.Epoch {
		return auth.Session{}, auth.ErrCredentials
	}
	user.SessionID = domain.NewID()
	result := auth.Session{Token: auth.Token(), CSRF: auth.Token(), User: user}
	e = tx.QueryRow(ctx, "INSERT INTO web_sessions(id,user_id,token_hash,csrf_hash,auth_epoch,expires_at,identity_provider_id,reauth_at) VALUES($1,$2,$3,$4,$5,clock_timestamp()+make_interval(secs=>$6),$7,LEAST(clock_timestamp(),to_timestamp($8))) RETURNING expires_at", user.SessionID, user.ID, auth.Digest(result.Token), auth.Digest(result.CSRF), user.Epoch, ttl, p.ID, proof.AuthenticatedAt).Scan(&result.Expires)
	if e != nil {
		return auth.Session{}, e
	}
	if f.Purpose != "login" {
		if _, e = tx.Exec(ctx, "UPDATE web_sessions SET revoked_at=clock_timestamp() WHERE id=$1", f.SessionID); e != nil {
			return auth.Session{}, e
		}
	}
	if _, e = tx.Exec(ctx, "INSERT INTO login_history(user_id,success) VALUES($1,true)", user.ID); e != nil {
		return auth.Session{}, e
	}
	if e = audit.Append(ctx, tx, s.AuditKey, user.ID, "session.external_created", user.SessionID); e != nil {
		return auth.Session{}, e
	}
	return result, nil
}

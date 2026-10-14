package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"filex.local/filex/internal/audit"
	"filex.local/filex/internal/domain"
	"filex.local/filex/internal/licensing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"slices"
	"time"
)

type Store struct {
	DB    *pgxpool.Pool
	Key   []byte
	dummy string
}
type Principal struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Locale    string    `json:"locale"`
	SessionID string    `json:"sessionId,omitempty"`
	Epoch     int64     `json:"-"`
	CSRFHash  string    `json:"-"`
	Reauth    time.Time `json:"-"`
	Scopes    []string  `json:"-"`
	Bearer    bool      `json:"-"`
}
type Session struct {
	Token   string    `json:"-"`
	CSRF    string    `json:"csrfToken"`
	User    Principal `json:"user"`
	Expires time.Time `json:"expiresAt"`
}

func New(db *pgxpool.Pool, key []byte) (*Store, error) {
	dummy, e := Hash("unavailable-account-" + Token())
	return &Store{db, key, dummy}, e
}
func Token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func Digest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func (s *Store) CreateUser(ctx context.Context, actor, email, password, role string) (string, error) {
	return s.createUser(ctx, actor, email, password, role, false)
}

func (s *Store) SetupAdmin(ctx context.Context, email, password string) (string, error) {
	return s.createUser(ctx, "host_setup", email, password, "admin", true)
}

func (s *Store) createUser(ctx context.Context, actor, email, password, role string, first bool) (string, error) {
	email, e := Email(email)
	if e != nil {
		return "", e
	}
	if role != "user" && role != "admin" {
		return "", ErrInvalid
	}
	hash, e := Hash(password)
	if e != nil {
		return "", e
	}
	id := domain.NewID()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(context.Background())
	var completed bool
	if e = tx.QueryRow(ctx, "SELECT completed FROM ops_setup WHERE singleton FOR UPDATE").Scan(&completed); e != nil {
		return "", e
	}
	if first {
		var exists bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users)").Scan(&exists); e != nil {
			return "", e
		}
		if completed || exists {
			return "", ErrConflict
		}
	}
	if e = licensing.UserSlot(ctx, tx, (actor == "host_cli" || first) && role == "admin"); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO users(id,email,role,password_hash) VALUES($1,$2,$3,$4)", id, email, role, hash); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO password_history(user_id,hash) VALUES($1,$2)", id, hash); e != nil {
		return "", e
	}
	if e = audit.Append(ctx, tx, s.Key, actor, "user.created", id); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "UPDATE ops_setup SET completed=true WHERE singleton"); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}
func (s *Store) throttle(ctx context.Context, key string) error {
	var n int
	e := s.DB.QueryRow(ctx, `INSERT INTO auth_limits(key,window_start,count) VALUES($1,clock_timestamp(),1) ON CONFLICT(key) DO UPDATE SET count=CASE WHEN auth_limits.window_start<clock_timestamp()-interval '1 minute' THEN 1 ELSE auth_limits.count+1 END,window_start=CASE WHEN auth_limits.window_start<clock_timestamp()-interval '1 minute' THEN clock_timestamp() ELSE auth_limits.window_start END RETURNING count`, Digest(key)).Scan(&n)
	if e != nil {
		return e
	}
	if n > 20 {
		return ErrRate
	}
	return nil
}
func (s *Store) Login(ctx context.Context, email, password, remote string, ttl int64) (Session, error) {
	if e := s.throttle(ctx, "ip:"+remote); e != nil {
		return Session{}, e
	}
	normalized, e := Email(email)
	if e != nil {
		normalized = "invalid@example.invalid"
	}
	if e = s.throttle(ctx, "email:"+normalized); e != nil {
		return Session{}, e
	}
	var p Principal
	var hash *string
	var disabled bool
	e = s.DB.QueryRow(ctx, "SELECT id::text,email,role,locale,password_hash,disabled,auth_epoch FROM users WHERE lower(email)=$1", normalized).Scan(&p.ID, &p.Email, &p.Role, &p.Locale, &hash, &disabled, &p.Epoch)
	encoded := s.dummy
	if e == nil && hash != nil {
		encoded = *hash
	}
	ok := Verify(encoded, password)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return Session{}, e
	}
	if !ok || hash == nil || disabled || p.ID == "" {
		if p.ID != "" {
			if _, err := s.DB.Exec(ctx, "INSERT INTO login_history(user_id,success) VALUES($1,false)", p.ID); err != nil {
				return Session{}, err
			}
		}
		return Session{}, ErrCredentials
	}
	result := Session{Token: Token(), CSRF: Token(), User: p}
	result.User.SessionID = domain.NewID()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return Session{}, e
	}
	defer tx.Rollback(context.Background())
	var epoch int64
	var activeHash string
	if e = tx.QueryRow(ctx, "SELECT auth_epoch,password_hash FROM users WHERE id=$1 AND NOT disabled FOR UPDATE", p.ID).Scan(&epoch, &activeHash); e != nil {
		return Session{}, ErrCredentials
	}
	if epoch != p.Epoch || activeHash != encoded {
		return Session{}, ErrCredentials
	}
	if e = tx.QueryRow(ctx, "INSERT INTO web_sessions(id,user_id,token_hash,csrf_hash,auth_epoch,expires_at) VALUES($1,$2,$3,$4,$5,clock_timestamp()+make_interval(secs=>$6)) RETURNING expires_at", result.User.SessionID, p.ID, Digest(result.Token), Digest(result.CSRF), epoch, ttl).Scan(&result.Expires); e != nil {
		return Session{}, e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO login_history(user_id,success) VALUES($1,true)", p.ID); e != nil {
		return Session{}, e
	}
	if e = audit.Append(ctx, tx, s.Key, p.ID, "session.created", result.User.SessionID); e != nil {
		return Session{}, e
	}
	return result, tx.Commit(ctx)
}
func (s *Store) Authenticate(ctx context.Context, token string, bearer bool) (Principal, error) {
	if len(token) != 43 {
		return Principal{}, ErrCredentials
	}
	var p Principal
	p.Bearer = bearer
	var e error
	if bearer {
		e = s.DB.QueryRow(ctx, `SELECT u.id::text,u.email,u.role,u.locale,u.auth_epoch,t.scopes FROM api_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=$1 AND t.revoked_at IS NULL AND t.expires_at>clock_timestamp() AND t.auth_epoch=u.auth_epoch AND NOT u.disabled`, Digest(token)).Scan(&p.ID, &p.Email, &p.Role, &p.Locale, &p.Epoch, &p.Scopes)
	} else {
		e = s.DB.QueryRow(ctx, `SELECT u.id::text,u.email,u.role,u.locale,u.auth_epoch,t.id::text,t.csrf_hash,t.reauth_at FROM web_sessions t JOIN users u ON u.id=t.user_id WHERE t.token_hash=$1 AND t.revoked_at IS NULL AND t.expires_at>clock_timestamp() AND t.auth_epoch=u.auth_epoch AND NOT u.disabled`, Digest(token)).Scan(&p.ID, &p.Email, &p.Role, &p.Locale, &p.Epoch, &p.SessionID, &p.CSRFHash, &p.Reauth)
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return Principal{}, ErrCredentials
	}
	return p, e
}
func (p Principal) Allows(scope string) bool { return !p.Bearer || slices.Contains(p.Scopes, scope) }
func (p Principal) CheckCSRF(csrf string) bool {
	return len(csrf) == 43 && subtle.ConstantTimeCompare([]byte(Digest(csrf)), []byte(p.CSRFHash)) == 1
}
func (s *Store) Logout(ctx context.Context, p Principal, id string) error {
	if !domain.ValidID(id) {
		return ErrInvalid
	}
	tag, e := s.DB.Exec(ctx, "UPDATE web_sessions SET revoked_at=clock_timestamp() WHERE id=$1 AND user_id=$2", id, p.ID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrForbidden
	}
	return nil
}
func (s *Store) Reauthenticate(ctx context.Context, p Principal, password string) error {
	if p.Bearer {
		return ErrForbidden
	}
	if e := s.throttle(ctx, "reauth:"+p.ID); e != nil {
		return e
	}
	var hash string
	e := s.DB.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1 AND NOT disabled", p.ID).Scan(&hash)
	if e != nil || !Verify(hash, password) {
		return ErrCredentials
	}
	tag, e := s.DB.Exec(ctx, "UPDATE web_sessions SET reauth_at=clock_timestamp() WHERE id=$1 AND auth_epoch=$2 AND revoked_at IS NULL AND expires_at>clock_timestamp()", p.SessionID, p.Epoch)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return ErrCredentials
	}
	return nil
}
func (s *Store) ChangePassword(ctx context.Context, p Principal, old, newPassword string) error {
	if e := s.Reauthenticate(ctx, p, old); e != nil {
		return e
	}
	hash, e := Hash(newPassword)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var epoch int64
	if e = tx.QueryRow(ctx, "SELECT auth_epoch FROM users WHERE id=$1 AND NOT disabled FOR UPDATE", p.ID).Scan(&epoch); e != nil {
		return e
	}
	if epoch != p.Epoch {
		return ErrCredentials
	}
	rows, e := tx.Query(ctx, "SELECT hash FROM password_history WHERE user_id=$1 ORDER BY id DESC LIMIT 5", p.ID)
	if e != nil {
		return e
	}
	hashes := []string{}
	for rows.Next() {
		var h string
		if e = rows.Scan(&h); e != nil {
			rows.Close()
			return e
		}
		hashes = append(hashes, h)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, h := range hashes {
		if Verify(h, newPassword) {
			return ErrInvalid
		}
	}
	if _, e = tx.Exec(ctx, "UPDATE users SET password_hash=$2,auth_epoch=auth_epoch+1 WHERE id=$1", p.ID, hash); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO password_history(user_id,hash) VALUES($1,$2)", p.ID, hash); e != nil {
		return e
	}
	if e = audit.Append(ctx, tx, s.Key, p.ID, "password.changed", p.ID); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// ThrottleExternal shares durable rate limits with local authentication.
func (s *Store) ThrottleExternal(ctx context.Context, remote, subject string) error {
	if e := s.throttle(ctx, "ip:"+remote); e != nil {
		return e
	}
	return s.throttle(ctx, "external:"+subject)
}

func (s *Store) ThrottleExternalSubject(ctx context.Context, subject string) error {
	return s.throttle(ctx, "external-subject:"+subject)
}

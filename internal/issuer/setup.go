package issuer

import (
	"context"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/cryptobox"
	"filex.local/filex/internal/setup"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"sync"
	"time"
)

const configContext = "filex/issuer_setup/v1/config"

type SetupInput struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RequiredACR  string `json:"requiredAcr"`
	Subject      string `json:"subject"`
}

func (in SetupInput) config(origin string) (Config, error) {
	for _, v := range []string{in.Issuer, in.ClientID, in.ClientSecret, in.RequiredACR, in.Subject} {
		if len(v) == 0 || len(v) > 2048 {
			return Config{}, errors.New("invalid setup input")
		}
	}
	return Config{Origin: origin, Issuer: in.Issuer, ClientID: in.ClientID, ClientSecret: in.ClientSecret, RequiredACR: in.RequiredACR, Operators: map[string]string{in.Subject: "security_admin"}}, nil
}
func saveSetup(ctx context.Context, db *pgxpool.Pool, key []byte, c Config) error {
	raw, e := json.Marshal(c)
	if e != nil {
		return e
	}
	box, e := cryptobox.Seal(key, configContext, raw)
	if e != nil {
		return e
	}
	// A singleton INSERT is the completion latch, including across API replicas.
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(724185041)"); e != nil {
		return e
	}
	var used bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM issuer_issuances) OR EXISTS(SELECT 1 FROM issuer_sessions) OR EXISTS(SELECT 1 FROM issuer_flows)").Scan(&used); e != nil {
		return e
	}
	if used {
		return errors.New("issuer already in use")
	}
	if _, e = tx.Exec(ctx, "INSERT INTO issuer_setup(singleton,config) VALUES(true,$1)", box); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func loadSetup(ctx context.Context, db *pgxpool.Pool, key []byte) (Config, error) {
	var box []byte
	var c Config
	if e := db.QueryRow(ctx, "SELECT config FROM issuer_setup WHERE singleton").Scan(&box); e != nil {
		return c, e
	}
	raw, e := cryptobox.Open(key, configContext, box)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(raw, &c)
	return c, e
}

// SetupRuntime never receives the license signing seed. Its independent key
// protects only the immutable identity-provider bootstrap configuration.
type SetupRuntime struct {
	DB      *pgxpool.Pool
	Key     []byte
	Access  setup.Access
	mu      sync.Mutex
	handler http.Handler
}

func (s *SetupRuntime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.URL.Path == "/issuer/api/setup" {
		if r.Method == http.MethodGet {
			var exists bool
			if e := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM issuer_setup)").Scan(&exists); e != nil {
				setup.Reply(w, 503, "unavailable")
				return
			}
			state := "complete"
			if !exists {
				state = "disabled"
				var used bool
				if e := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM issuer_issuances) OR EXISTS(SELECT 1 FROM issuer_sessions) OR EXISTS(SELECT 1 FROM issuer_flows)").Scan(&used); e != nil {
					setup.Reply(w, 503, "unavailable")
					return
				}
				if !used && len(s.Key) == 32 && s.Access.Enabled() {
					state = "pending"
				}
			}
			setup.Reply(w, 200, state)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if len(s.Key) != 32 || !s.Access.Allowed(r) {
			setup.Reply(w, 403, "disabled")
			return
		}
		var in SetupInput
		if setup.Decode(w, r, &in) != nil {
			setup.Reply(w, 400, "invalid_input")
			return
		}
		c, e := in.config(s.Access.Origin)
		if e != nil {
			setup.Reply(w, 400, "invalid_input")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		var exists bool
		if e = s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM issuer_setup)").Scan(&exists); e != nil {
			setup.Reply(w, 503, "unavailable")
			return
		}
		if exists {
			setup.Reply(w, 409, "complete")
			return
		}
		server, e := New(ctx, s.DB, c)
		if e != nil {
			setup.Reply(w, 400, "identity_unavailable")
			return
		}
		if e = saveSetup(ctx, s.DB, s.Key, c); e != nil {
			if err := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM issuer_setup)").Scan(&exists); err == nil && exists {
				setup.Reply(w, 409, "complete")
			} else {
				setup.Reply(w, 503, "unavailable")
			}
			return
		}
		s.mu.Lock()
		s.handler = server.Handler()
		s.mu.Unlock()
		setup.Reply(w, 200, "complete")
		return
	}
	s.mu.Lock()
	handler := s.handler
	if handler == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		c, e := loadSetup(ctx, s.DB, s.Key)
		if e == nil {
			var server *Server
			server, e = New(ctx, s.DB, c)
			if e == nil {
				handler = server.Handler()
				s.handler = handler
			}
		}
		cancel()
		if e != nil {
			s.mu.Unlock()
			if errors.Is(e, pgx.ErrNoRows) {
				setup.Reply(w, 503, "setup_required")
			} else {
				setup.Reply(w, 503, "unavailable")
			}
			return
		}
	}
	s.mu.Unlock()
	handler.ServeHTTP(w, r)
}

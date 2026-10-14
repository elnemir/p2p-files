package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/issuer"
	"filex.local/filex/internal/setup"
	"log"
	"net/http"
	"os"
	"time"
)

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, e := database.Connect(ctx, os.Getenv("ISSUER_DATABASE_URL"))
	if e != nil {
		return e
	}
	defer db.Close()
	if len(os.Args) == 2 && os.Args[1] == "migrate" {
		if e = issuer.Migrate(ctx, db); e != nil {
			return e
		}
		return issuer.ProvisionRoles(ctx, db, os.Getenv("ISSUER_APP_PASSWORD"), os.Getenv("ISSUER_SIGNER_PASSWORD"))
	}
	var handler http.Handler
	if raw := os.Getenv("ISSUER_CONFIG_KEY"); raw != "" {
		if os.Getenv("ISSUER_OPERATORS") != "" || os.Getenv("ISSUER_OIDC_URL") != "" {
			return errors.New("choose either database setup or environment identity configuration")
		}
		key, e := base64.StdEncoding.Strict().DecodeString(raw)
		if e != nil || len(key) != 32 {
			return errors.New("invalid issuer config key")
		}
		handler = &issuer.SetupRuntime{DB: db, Key: key, Access: setup.FromEnv("ISSUER_", os.Getenv, os.Getenv("ISSUER_ORIGIN"))}
	} else {
		var operators map[string]string
		if json.Unmarshal([]byte(os.Getenv("ISSUER_OPERATORS")), &operators) != nil {
			return errors.New("invalid operator allowlist")
		}
		s, e := issuer.New(ctx, db, issuer.Config{Origin: os.Getenv("ISSUER_ORIGIN"), Issuer: os.Getenv("ISSUER_OIDC_URL"), ClientID: os.Getenv("ISSUER_OIDC_CLIENT_ID"), ClientSecret: os.Getenv("ISSUER_OIDC_CLIENT_SECRET"), RequiredACR: os.Getenv("ISSUER_MFA_ACR"), Operators: operators})
		if e != nil {
			return e
		}
		handler = s.Handler()

	}
	addr := os.Getenv("ISSUER_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8082"
	}
	server := http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	return server.ListenAndServe()
}
func main() {
	if e := run(); e != nil {
		log.Print("issuer stopped; check configuration, database and identity provider")
		os.Exit(1)
	}
}

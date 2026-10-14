package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"filex.local/filex/internal/database"
	"filex.local/filex/internal/issuer"
	"filex.local/filex/internal/licensing"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func run() error {
	if len(os.Args) == 4 && os.Args[1] == "keygen" {
		pub, key, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			return e
		}
		f, e := os.OpenFile(os.Args[3], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		_, e = f.Write(key.Seed())
		closeErr := f.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{os.Args[2]: base64.StdEncoding.EncodeToString(pub)})
	}
	path := os.Getenv("ISSUER_SIGNING_KEY_FILE")
	info, e := os.Stat(path)
	if e != nil {
		return e
	}
	if info.Mode().Perm()&0077 != 0 || !info.Mode().IsRegular() {
		return errors.New("signer key must be a private regular file")
	}
	seed, e := os.ReadFile(path)
	if e != nil || len(seed) != ed25519.SeedSize {
		return errors.New("invalid signer seed")
	}
	key := ed25519.NewKeyFromSeed(seed)
	kid := os.Getenv("ISSUER_KEY_ID")
	if kid == "" {
		return errors.New("key id required")
	}
	if len(os.Args) == 2 && os.Args[1] == "sign-release" {
		raw, e := io.ReadAll(io.LimitReader(os.Stdin, licensing.MaxBytes+1))
		if e != nil {
			return e
		}
		var r licensing.Release
		if e = licensing.Decode(raw, &r); e != nil {
			return e
		}
		signed, e := licensing.SignRelease(r, kid, key)
		if e != nil {
			return e
		}
		_, e = fmt.Fprintln(os.Stdout, signed)
		return e
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, e := database.Connect(ctx, os.Getenv("ISSUER_DATABASE_URL"))
	if e != nil {
		return e
	}
	defer db.Close()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		worked, e := issuer.SignNext(ctx, db, kid, key)
		if e != nil {
			return e
		}
		if worked {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "signer failed; check private key and database configuration")
		os.Exit(1)
	}
}

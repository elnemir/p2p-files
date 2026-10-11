package main

import (
	"encoding/json"
	"filex.local/filex/internal/verify"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("filex-verify 0.1.0-dev")
		return
	}
	key := flag.String("public-key", "", "Trusted Ed25519 PEM obtained separately")
	fingerprint := flag.String("fingerprint", "", "Expected lowercase SHA-256 of DER/SPKI, obtained independently")
	verifyFiles := flag.Bool("verify-files", false, "Also read and hash all payload bytes")
	flag.Parse()
	if *key == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: filex-verify --public-key trusted.pem [--fingerprint sha256] [--verify-files] transfer.zip")
		os.Exit(2)
	}
	raw, e := os.ReadFile(*key)
	if e != nil {
		fmt.Fprintln(os.Stderr, "cannot read trusted public key")
		os.Exit(2)
	}
	count, e := verify.Check(flag.Arg(0), raw, *fingerprint, *verifyFiles)
	if e != nil {
		fmt.Fprintln(os.Stderr, "verification failed:", e)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "verified", "manifestFiles": count, "filesChecked": *verifyFiles})
}

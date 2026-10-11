package main

import (
	"context"
	"encoding/json"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/config"
	"filex.local/filex/internal/httpapi"
	"filex.local/filex/internal/storage"
	"fmt"
	"io"
	"os"
)

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("filex-cli 0.1.0-dev")
		return nil
	}
	if len(os.Args) == 3 && os.Args[1] == "config" && os.Args[2] == "validate" {
		_, err := config.Load(os.Getenv)
		if err == nil {
			fmt.Println("Bootstrap configuration valid.")
		}
		return err
	}
	if len(os.Args) == 3 && os.Args[1] == "user" && os.Args[2] == "create" {
		var v struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		d := json.NewDecoder(io.LimitReader(os.Stdin, 65536))
		d.DisallowUnknownFields()
		if err := d.Decode(&v); err != nil {
			return fmt.Errorf("invalid user JSON on stdin")
		}
		c, p, err := bootstrap.Open(context.Background())
		if err != nil {
			return err
		}
		defer p.Close()
		app, err := httpapi.New(p, c.MasterKEK)
		if err != nil {
			return err
		}
		id, err := app.Auth.CreateUser(context.Background(), "host_cli", v.Email, v.Password, v.Role)
		if err != nil {
			return fmt.Errorf("user provisioning failed")
		}
		fmt.Println(id)
		return nil
	}
	if len(os.Args) == 4 && os.Args[1] == "storage" && os.Args[2] == "local" {
		_, p, e := bootstrap.Open(context.Background())
		if e != nil {
			return e
		}
		defer p.Close()
		id, e := (storage.Registry{DB: p}).ProvisionLocal(context.Background(), os.Args[3])
		if e != nil {
			return fmt.Errorf("local storage provisioning failed")
		}
		fmt.Println(id)
		return nil
	}
	if len(os.Args) == 2 && os.Args[1] == "migrate" {
		_, p, err := bootstrap.Open(context.Background())
		if err != nil {
			return err
		}
		p.Close()
		fmt.Println("Migrations, key check and settings verified.")
		return nil
	}
	return fmt.Errorf("supported commands: --version, config validate, migrate, user create (JSON stdin), storage local /absolute/root")
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

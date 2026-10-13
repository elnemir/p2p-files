package main

import (
	"context"
	"errors"
	"filex.local/filex/internal/bootstrap"
	"filex.local/filex/internal/httpapi"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Embed IANA data so minimal runtime images accept the same zones as local development.
	_ "time/tzdata"
)

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, p, err := bootstrap.Open(ctx)
	if err != nil {
		return err
	}
	defer p.Close()
	app, err := httpapi.New(p, c.MasterKEK)
	if err != nil {
		return err
	}
	defer app.Close()
	servers := []*http.Server{
		{Addr: c.HTTPAddr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 0, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16384},
		{Addr: c.InternalAddr, Handler: app.Metrics(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16384},
	}
	errs := make(chan error, len(servers))
	for _, s := range servers {
		go func() { errs <- s.ListenAndServe() }()
	}
	select {
	case <-ctx.Done():
	case err = <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, s := range servers {
		if e := s.Shutdown(shutdown); e != nil {
			_ = s.Close()
			if err == nil {
				err = e
			}
		}
	}
	return err
}
func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "reason", err.Error())
		os.Exit(1)
	}
}

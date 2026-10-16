// Command np-trophy stores PS3 trophy unlocks for OpenPak accounts. It is a private API
// behind np-account, which verifies the firmware's ticket and maps it to the core account;
// this service trusts its caller (bearer key) and never faces a console.
//
//	NP_TROPHY_DATABASE_URL  Postgres; required
//	NP_TROPHY_KEY           bearer key shared with np-account, at least 32 bytes; required
//	NP_TROPHY_ADDR          listen address, default :8530 (never published)
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbURL, key := os.Getenv("NP_TROPHY_DATABASE_URL"), os.Getenv("NP_TROPHY_KEY")
	if dbURL == "" || len(key) < 32 {
		log.Error("NP_TROPHY_DATABASE_URL and NP_TROPHY_KEY (>= 32 bytes) must be set")
		os.Exit(2)
	}
	addr := os.Getenv("NP_TROPHY_ADDR")
	if addr == "" {
		addr = ":8530"
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	store := &Store{DB: pool}
	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(store, key, log),
		MaxHeaderBytes:    8 << 10,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Info("np-trophy listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

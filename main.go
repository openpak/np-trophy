// Command np-trophy serves native PS3 trophies and a separate private storage API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"openpak/np-common/coreclient"
	"openpak/np-common/events"
	"openpak/np-common/nphttp"
	"openpak/np-common/ticket"
	"openpak/np-trophy/native"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("np-trophy stopped", "err", err)
		os.Exit(1)
	}
}
func run() error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbURL, key := os.Getenv("NP_TROPHY_DATABASE_URL"), os.Getenv("NP_TROPHY_KEY")
	if dbURL == "" || len(key) < 32 {
		log.Error("NP_TROPHY_DATABASE_URL and NP_TROPHY_KEY (>= 32 bytes) must be set")
		return fmt.Errorf("NP_TROPHY_DATABASE_URL and NP_TROPHY_KEY (>= 32 bytes) must be set")
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
		return err
	}
	defer pool.Close()
	store := &Store{DB: pool}
	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate", "err", err)
		return err
	}
	if len(os.Getenv("NP_CORE_KEY")) < 32 {
		return fmt.Errorf("NP_CORE_KEY must be at least 32 bytes")
	}
	core, err := coreclient.Dial(nphttp.Env("NP_CORE_ADDR", "localhost:7000"), os.Getenv("NP_CORE_KEY"))
	if err != nil {
		return err
	}
	defer core.Close()
	signer, err := ticket.Load(os.Getenv("NP_TICKET_KEY"))
	if err != nil {
		return err
	}
	resourceKey, err := os.ReadFile(os.Getenv("NP_TROPHY_RESOURCE_KEY"))
	if err != nil {
		return err
	}
	if len(resourceKey) != 20 {
		return fmt.Errorf("NP_TROPHY_RESOURCE_KEY must contain 20 bytes")
	}
	standing := events.NewState()
	gate := events.Gate{Client: core, State: standing}
	consumer := &events.Consumer{Core: core, State: standing, Log: log}
	go consumer.Run(ctx, 5*time.Second)
	svc := &native.Service{Core: gate, Tickets: signer, TrophyResourceKey: resourceKey, Trophies: nativeStore{store}}
	mux := http.NewServeMux()
	svc.Register(mux)
	nphttp.Health(mux)
	front, err := nphttp.Front()
	if err != nil {
		return err
	}
	// Private API binds separately and is never added to the console mux.
	private := nphttp.Server(addr, Handler(store, key, log))
	private.ReadHeaderTimeout = 5 * time.Second
	private.ReadTimeout = 5 * time.Second
	private.WriteTimeout = 5 * time.Second
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	privateErr := make(chan error, 1)
	go func() { privateErr <- private.Serve(ln); stop() }()
	err = nphttp.Run(ctx, mux, nphttp.Env("NP_HTTP_ADDR", ":8531"), os.Getenv("NP_TLS_ADDR"), front)
	stop()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e := private.Shutdown(shutdown); e != nil {
		private.Close()
		if err == nil {
			err = e
		}
	}
	e := <-privateErr
	if err == nil && !errors.Is(e, http.ErrServerClosed) {
		err = e
	}
	return err
}

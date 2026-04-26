package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sitecraft/backend/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := app.LoadConfig()
	srv := app.NewServer(cfg)

	apiServer := &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           srv.APIRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("sitecraft api listening on %s", apiServer.Addr)
	errCh := make(chan error, 1)
	go func() {
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		log.Printf("sitecraft shutdown requested")
	case err := <-errCh:
		log.Printf("sitecraft server error: %v", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = apiServer.Shutdown(shutdownCtx)

	select {
	case err := <-errCh:
		return err
	default:
		log.Printf("sitecraft shutdown complete")
		return nil
	}
}

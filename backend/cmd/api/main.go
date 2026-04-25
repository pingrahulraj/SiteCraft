package main

import (
	"log"
	"net/http"
	"os"
	"sync"
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
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	deliveryServer := &http.Server{
		Addr:              ":" + cfg.DeliveryPort,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("sitecraft api listening on %s", apiServer.Addr)
	log.Printf("sitecraft delivery listening on %s", deliveryServer.Addr)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("api server error: %v", err)
			os.Exit(1)
		}
	}()
	go func() {
		defer wg.Done()
		if err := deliveryServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("delivery server error: %v", err)
			os.Exit(1)
		}
	}()
	wg.Wait()
	return nil
}

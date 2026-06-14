package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"business-workbench/backend-go/internal/app"
	"business-workbench/backend-go/internal/config"
	"business-workbench/backend-go/internal/db"
)

func main() {
	cfg := config.Load()

	store, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	if err := store.InitSchema(); err != nil {
		log.Fatalf("init database schema: %v", err)
	}

	router := app.NewRouter(cfg, store)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		log.Printf("business-workbench-go starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	if c := app.GetCron(); c != nil {
		c.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	log.Println("server exited")
}

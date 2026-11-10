// Command api runs the Gin REST server.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/handler"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	db, err := repository.OpenDB(cfg.DB.DSN)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	rdb := repository.NewRedisClient(cfg.Redis.Addr)
	defer rdb.Close()
	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}
	st, buckets, err := storage.NewFromEnv()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	repos := repository.New(db, rdb)
	services := service.New(repos, st, buckets, cfg)
	router := handler.SetupRouter(&handler.Deps{
		Config: cfg, Repos: repos, Services: services, Storage: st, Buckets: buckets,
	})
	srv := &http.Server{
		Addr:         cfg.AppAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("api listening on %s (env=%s)", cfg.AppAddr, cfg.Env)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()
	<-ctx.Done()
	log.Printf("shutdown signal received; draining with 30s timeout")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

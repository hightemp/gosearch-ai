package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gosearch-ai/backend/internal/config"
	"gosearch-ai/backend/internal/db"
	"gosearch-ai/backend/internal/httpapi"
	"gosearch-ai/backend/internal/log"
)

func main() {
	ctx := context.Background()
	logger := log.New()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatal().Err(err).Msg("config")
	}

	pool, err := db.NewPGXPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("db.connect")
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		logger.Fatal().Err(err).Msg("db.migrate")
	}

	api := httpapi.NewServer(cfg, pool, logger)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info().Str("addr", cfg.HTTPAddr).Msg("http.listen")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("http.serve")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctxShutdown, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShutdown)
	logger.Info().Msg("shutdown")
}

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

	"affluena/internal/config"
	"affluena/internal/db"
	"affluena/internal/recurring"
	"affluena/internal/server"
	"affluena/internal/transaction"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.RunMigrations {
		if err := db.Migrate(ctx, pool, "migrations"); err != nil {
			slog.Error("database migration failed", "error", err)
			os.Exit(1)
		}
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	if cfg.RecurringSchedulerEnabled {
		transactionRepo := transaction.NewRepository(pool)
		recurringRepo := recurring.NewRepository(pool, transactionRepo)
		recurring.NewScheduler(recurringRepo, cfg.RecurringSchedulerInterval, cfg.RecurringSchedulerBatchSize).Start(appCtx)
		slog.Info("recurring scheduler enabled", "interval", cfg.RecurringSchedulerInterval, "batch_size", cfg.RecurringSchedulerBatchSize)
	}

	router := server.NewRouter(cfg, pool)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("starting api", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("api failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	appCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("api shutdown failed", "error", err)
		os.Exit(1)
	}
}

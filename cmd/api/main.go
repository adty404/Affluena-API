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

	"affluena-api/internal/activity"
	"affluena-api/internal/config"
	"affluena-api/internal/db"
	"affluena-api/internal/httpx"
	"affluena-api/internal/mailer"
	"affluena-api/internal/notification"
	"affluena-api/internal/recurring"
	"affluena-api/internal/server"
	"affluena-api/internal/transaction"
)

// notificationMailer adapts mailer.Mailer (which sends to a []string) to the
// single-recipient notification.MailerPort used by the notification scheduler.
type notificationMailer struct {
	inner mailer.Mailer
}

func (m notificationMailer) Send(ctx context.Context, to string, subject string, htmlBody string) error {
	return m.inner.SendEmail(ctx, []string{to}, subject, htmlBody)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if cfg.UsesInsecureProdDB() {
		slog.Warn("running in production with an unencrypted database connection (sslmode=disable); permitted via ALLOW_INSECURE_DB. Ensure Postgres is reachable only over a trusted host/network.")
	}

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
		activityRepo := activity.NewRepository(pool)
		activityUC := activity.NewUseCase(activityRepo)
		transactionRepo := transaction.NewRepository(pool)
		recurringRepo := recurring.NewRepository(pool, transactionRepo)
		recurring.NewScheduler(recurring.NewUseCase(recurringRepo, activityUC), cfg.RecurringSchedulerInterval, cfg.RecurringSchedulerBatchSize).Start(appCtx)
		slog.Info("recurring scheduler enabled", "interval", cfg.RecurringSchedulerInterval, "batch_size", cfg.RecurringSchedulerBatchSize)
	}

	if cfg.NotificationSchedulerEnabled {
		deliveryRepo := notification.NewDeliveryRepository(pool)
		var notifMailer notification.MailerPort
		if cfg.SMTPHost != "" && cfg.SMTPPort > 0 {
			notifMailer = notificationMailer{mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)}
		}
		notifier := notification.NewNotifier(deliveryRepo, notifMailer)
		notification.NewScheduler(deliveryRepo, notifier, cfg.NotificationSchedulerInterval).Start(appCtx)
		slog.Info("notification scheduler enabled", "interval", cfg.NotificationSchedulerInterval)
	}

	httpx.InitRateLimiters(cfg)
	httpx.AuthLimiter.StartCleanup(appCtx, httpx.CleanupInterval)
	httpx.APILimiter.StartCleanup(appCtx, httpx.CleanupInterval)

	router := server.NewRouter(cfg, pool)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
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

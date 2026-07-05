package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"affluena-api/internal/activity"
	"affluena-api/internal/alert"
	"affluena-api/internal/apilog"
	"affluena-api/internal/auth"
	"affluena-api/internal/budget"
	"affluena-api/internal/category"
	"affluena-api/internal/config"
	"affluena-api/internal/dashboard"
	"affluena-api/internal/debt"
	"affluena-api/internal/export"
	"affluena-api/internal/goal"
	"affluena-api/internal/httpx"
	"affluena-api/internal/mailer"
	"affluena-api/internal/notification"
	"affluena-api/internal/partner"
	"affluena-api/internal/quickentry"
	"affluena-api/internal/recurring"
	"affluena-api/internal/report"
	"affluena-api/internal/splitbill"
	"affluena-api/internal/tag"
	"affluena-api/internal/tracker"
	"affluena-api/internal/transaction"
	"affluena-api/internal/wallet"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	apilogRepo := apilog.NewRepository(pool)
	apilogHandler := apilog.NewHandler(apilogRepo)
	router := gin.New()
	applyTrustedProxies(router, cfg.TrustedProxies)
	router.Use(gin.Recovery())
	router.Use(securityHeaders(cfg.Env))

	// Setup CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedCORSOrigins(cfg.CORSAllowedOrigins),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(apilog.APILogMiddleware(apilogRepo))

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenDuration, cfg.RefreshTokenDuration)
	authRepo := auth.NewRepository(pool)

	activityRepo := activity.NewRepository(pool)
	activityUC := activity.NewUseCase(activityRepo)
	activityHandler := activity.NewHandler(activityUC)

	// Build the SMTP mailer once, unconditionally when SMTP is configured, and
	// reuse it for BOTH transactional auth email (password reset) and budget
	// alerts. Previously the mailer was constructed only inside the alert guard,
	// so auth was always wired without a mailer and forgot-password silently sent
	// nothing.
	var smtpMailer mailer.Mailer
	if cfg.SMTPHost != "" && cfg.SMTPPort > 0 {
		smtpMailer = mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	}

	var authService *auth.Service
	if smtpMailer != nil {
		authService = auth.NewServiceWithMailer(authRepo, tokenManager, activityUC, mailer.AsSingleRecipient(smtpMailer), cfg.SMTPFrom, cfg.AppBaseURL)
	} else {
		authService = auth.NewService(authRepo, tokenManager, activityUC)
	}
	authHandler := auth.NewHandler(authService)

	walletRepo := wallet.NewRepository(pool)
	walletHandler := wallet.NewHandler(wallet.NewUseCase(walletRepo, activityUC))
	partnerHandler := partner.NewHandler(partner.NewUseCase(partner.NewRepository(pool), activityUC))
	categoryHandler := category.NewHandler(category.NewUseCase(category.NewRepository(pool), activityUC))
	tagHandler := tag.NewHandler(tag.NewUseCase(tag.NewRepository(pool), activityUC))
	budgetUC := budget.NewUseCase(budget.NewRepository(pool), activityUC)
	budgetHandler := budget.NewHandler(budgetUC)

	// The notifier is the single rule-gated send path (reuses the smtpMailer
	// built above); the alert use case consults it so budget-alerts respect
	// notification_rules.
	notifDeliveryRepo := notification.NewDeliveryRepository(pool)
	var notifMailer notification.MailerPort
	if smtpMailer != nil {
		notifMailer = singleRecipientMailer{smtpMailer}
	}
	notifier := notification.NewNotifier(notifDeliveryRepo, notifMailer)

	var alertUC alert.UseCase
	if smtpMailer != nil {
		// Gate the budget-alert send on the user's notification_rules row.
		alertUC = alert.NewUseCaseWithGate(alert.NewRepository(pool), budgetUC, smtpMailer, alertGate{notifier})
	}

	feedRepo := alert.NewFeedRepository(pool)
	feedUC := alert.NewFeedUseCase(feedRepo, budgetUC)
	feedHandler := alert.NewFeedHandler(feedUC)

	transactionRepo := transaction.NewRepository(pool)
	transactionUC := transaction.NewUseCase(transactionRepo, activityUC, alertUC, walletRepo)
	transactionHandler := transaction.NewHandler(transactionUC)
	quickEntryHandler := quickentry.NewHandler(quickentry.NewUseCase(quickentry.NewRepository(pool), transactionUC, activityUC))
	dashboardHandler := dashboard.NewHandler(dashboard.NewUseCase(dashboard.NewRepository(pool)))
	debtRepo := debt.NewRepository(pool, transactionRepo)
	debtUseCase := debt.NewUseCase(debtRepo, activityUC)
	debtHandler := debt.NewHandler(debtUseCase)
	splitBillHandler := splitbill.NewHandler(splitbill.NewUseCase(pool, transactionRepo, debtRepo, activityUC))
	goalHandler := goal.NewHandler(goal.NewUsecase(goal.NewRepository(pool), activityUC))
	recurringHandler := recurring.NewHandler(recurring.NewUseCase(recurring.NewRepository(pool, transactionRepo), activityUC))
	trackerHandler := tracker.NewHandler(tracker.NewUseCase(
		tracker.NewInstallmentRepository(pool, transactionRepo),
		tracker.NewSubscriptionRepository(pool, transactionRepo),
		activityUC,
	))
	exportHandler := export.NewHandler(export.NewUseCase(export.NewRepository(pool)))
	reportHandler := report.NewHandler(report.NewUseCase(report.NewRepository(pool)))
	notificationHandler := notification.NewHandler(notification.NewUseCase(notification.NewRepository(pool)))

	// /healthz pings the database so the deploy workflow's `curl -fsS /healthz`
	// gate (and nginx health checks) fail when Postgres is unreachable, instead
	// of reporting a dead app as live. The response never carries connection
	// details — only a terse status ("db":"unreachable"), so nothing sensitive
	// can leak through the public endpoint. apilog middleware still skips this
	// path, so health probes never spam api_logs.
	router.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		var one int
		if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "db": "unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")

	// Apply rate limiting to auth endpoints
	authGroup := v1.Group("/auth")
	authGroup.Use(httpx.AuthLimiter.Middleware())
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/forgot-password", authHandler.ForgotPassword)
	authGroup.POST("/reset-password", authHandler.ResetPassword)

	protected := v1.Group("")
	protected.Use(auth.AuthMiddleware(tokenManager))
	// General per-IP rate limiting for the whole authenticated API (auth
	// endpoints keep their own stricter AuthLimiter above).
	protected.Use(httpx.APILimiter.Middleware())
	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/account", authHandler.UpdateAccount)
	protected.PUT("/auth/password", authHandler.ChangePassword)
	protected.GET("/auth/sessions", authHandler.ListSessions)
	protected.DELETE("/auth/sessions/:session_id", authHandler.RevokeSession)

	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/dashboard/cashflow-trend", dashboardHandler.CashflowTrend)
	protected.GET("/dashboard/expense-distribution", dashboardHandler.ExpenseDistribution)
	protected.GET("/dashboard/forecast", dashboardHandler.Forecast)

	protected.GET("/export/csv", exportHandler.ExportCSV)
	protected.GET("/export/jobs", exportHandler.ListJobs)
	protected.GET("/export/jobs/:id", exportHandler.GetJob)

	protected.GET("/reports/income", reportHandler.Income)
	protected.GET("/reports/expense", reportHandler.Expense)
	protected.GET("/reports/cashflow", reportHandler.Cashflow)
	protected.GET("/reports/debt", reportHandler.Debt)
	protected.GET("/reports/goal", reportHandler.Goal)
	protected.GET("/reports/overview", reportHandler.Overview)

	protected.GET("/system-logs", apilogHandler.List)
	protected.GET("/system-logs/:id", apilogHandler.GetLog)

	protected.GET("/notifications/rules", notificationHandler.List)
	protected.PUT("/notifications/rules/:id", notificationHandler.Update)

	protected.GET("/alerts", feedHandler.List)
	protected.GET("/alerts/:id", feedHandler.Get)

	protected.GET("/activities", activityHandler.ListActivities)
	protected.GET("/activities/:id", activityHandler.GetActivity)

	protected.POST("/wallets", walletHandler.Create)
	protected.GET("/wallets", walletHandler.List)
	protected.GET("/wallets/:id", walletHandler.Get)
	protected.PUT("/wallets/:id", walletHandler.Update)
	protected.DELETE("/wallets/:id", walletHandler.Delete)
	protected.POST("/wallets/:id/invites", walletHandler.InviteMember)
	protected.PATCH("/wallets/:id/members/:member_id", walletHandler.RespondInvite)
	protected.GET("/wallets/:id/members", walletHandler.ListMembers)
	protected.GET("/wallets/:id/analytics", walletHandler.Analytics)

	// Account-level partner links: one-way, read-only viewer of all the owner's
	// wallets (current + future).
	protected.POST("/partners/invites", partnerHandler.Invite)
	protected.GET("/partners", partnerHandler.List)
	protected.PATCH("/partners/:id", partnerHandler.Respond)
	protected.DELETE("/partners/:id", partnerHandler.Revoke)

	protected.POST("/categories", categoryHandler.Create)
	protected.GET("/categories", categoryHandler.List)
	// Static route registered before the ":id" params below; Gin resolves
	// static children ahead of param nodes, so PUT /categories/reorder never
	// hits PUT /categories/:id.
	protected.PUT("/categories/reorder", categoryHandler.Reorder)
	protected.GET("/categories/:id", categoryHandler.Get)
	protected.PUT("/categories/:id", categoryHandler.Update)
	protected.DELETE("/categories/:id", categoryHandler.Delete)

	protected.POST("/tags", tagHandler.Create)
	protected.GET("/tags", tagHandler.List)
	protected.GET("/tags/:id", tagHandler.Get)
	protected.PUT("/tags/:id", tagHandler.Update)
	protected.DELETE("/tags/:id", tagHandler.Delete)

	protected.POST("/transactions/split", splitBillHandler.Split)
	protected.GET("/split-bills", splitBillHandler.List)
	protected.GET("/split-bills/:id", splitBillHandler.Get)
	protected.POST("/transactions", transactionHandler.Create)
	protected.GET("/transactions", transactionHandler.List)
	protected.GET("/transactions/:id", transactionHandler.Get)
	protected.PUT("/transactions/:id", transactionHandler.Update)
	protected.DELETE("/transactions/:id", transactionHandler.Delete)

	protected.POST("/quick-entry-templates", quickEntryHandler.Create)
	protected.GET("/quick-entry-templates", quickEntryHandler.List)
	protected.GET("/quick-entry-templates/:id", quickEntryHandler.Get)
	protected.PUT("/quick-entry-templates/:id", quickEntryHandler.Update)
	protected.DELETE("/quick-entry-templates/:id", quickEntryHandler.Delete)
	protected.POST("/quick-entry-templates/:id/execute", quickEntryHandler.Execute)

	protected.POST("/category-budgets", budgetHandler.Create)
	protected.GET("/category-budgets", budgetHandler.List)
	protected.GET("/category-budgets/alerts", budgetHandler.Alerts)
	protected.GET("/category-budgets/report", budgetHandler.Report)
	protected.GET("/category-budgets/:id", budgetHandler.Get)
	protected.PUT("/category-budgets/:id", budgetHandler.Update)
	protected.DELETE("/category-budgets/:id", budgetHandler.Delete)

	protected.POST("/debts", debtHandler.Create)
	protected.GET("/debts", debtHandler.List)
	protected.GET("/debts/:id", debtHandler.Get)
	protected.PUT("/debts/:id", debtHandler.Update)
	protected.DELETE("/debts/:id", debtHandler.Delete)
	protected.POST("/debts/:id/pay", debtHandler.Pay)

	protected.POST("/goals", goalHandler.Create)
	protected.GET("/goals", goalHandler.List)
	protected.GET("/goals/:id", goalHandler.Get)
	protected.PUT("/goals/:id", goalHandler.Update)
	protected.POST("/goals/:id/members", goalHandler.InviteMember)
	protected.PUT("/goals/:id/members/:user_id/respond", goalHandler.RespondInvite)

	protected.POST("/installments", trackerHandler.CreateInstallment)
	protected.GET("/installments", trackerHandler.ListInstallments)
	protected.GET("/installments/:id", trackerHandler.GetInstallment)
	protected.PUT("/installments/:id", trackerHandler.UpdateInstallment)
	protected.DELETE("/installments/:id", trackerHandler.DeleteInstallment)
	protected.POST("/installments/:id/pay", trackerHandler.PayInstallment)

	protected.POST("/subscriptions", trackerHandler.CreateSubscription)
	protected.GET("/subscriptions", trackerHandler.ListSubscriptions)
	protected.GET("/subscriptions/:id", trackerHandler.GetSubscription)
	protected.PUT("/subscriptions/:id", trackerHandler.UpdateSubscription)
	protected.DELETE("/subscriptions/:id", trackerHandler.DeleteSubscription)
	protected.POST("/subscriptions/:id/pay", trackerHandler.PaySubscription)

	protected.POST("/recurring-transactions", recurringHandler.Create)
	protected.GET("/recurring-transactions", recurringHandler.List)
	protected.GET("/recurring-transactions/:id", recurringHandler.Get)
	protected.PUT("/recurring-transactions/:id", recurringHandler.Update)
	protected.DELETE("/recurring-transactions/:id", recurringHandler.Delete)
	protected.POST("/recurring-transactions/:id/run", recurringHandler.RunManual)

	return router
}

// singleRecipientMailer adapts mailer.Mailer (which sends to a []string) to the
// single-recipient notification.MailerPort used by the notifier.
type singleRecipientMailer struct {
	inner mailer.Mailer
}

func (m singleRecipientMailer) Send(ctx context.Context, to string, subject string, htmlBody string) error {
	return m.inner.SendEmail(ctx, []string{to}, subject, htmlBody)
}

// alertGate adapts *notification.Notifier to the alert package's notificationGate
// interface, mapping notification.Decision onto alert.channelDecision without
// creating an import cycle (alert never imports notification).
type alertGate struct {
	notifier *notification.Notifier
}

func (g alertGate) Decide(ctx context.Context, userID, ruleKey string) (alert.ChannelDecision, error) {
	d, err := g.notifier.Decide(ctx, userID, ruleKey)
	if err != nil {
		return alert.ChannelDecision{}, err
	}
	return alert.ChannelDecision{Send: d.Send, Email: d.Email, InApp: d.InApp}, nil
}

// applyTrustedProxies configures which proxy hops gin trusts for the
// X-Forwarded-For header when resolving ClientIP(). The API sits behind an nginx
// reverse proxy on the same host / Docker network, so only that proxy should be
// trusted. Without this, gin trusts ALL proxies and a client-forged
// X-Forwarded-For lets a single attacker rotate ClientIP() and bypass the per-IP
// AuthLimiter on /auth/login and /auth/register. On a malformed list we fall back
// to trusting NO proxy (ClientIP == direct remote address) rather than
// everything, so we always fail closed.
func applyTrustedProxies(engine *gin.Engine, proxies []string) {
	if err := engine.SetTrustedProxies(proxies); err != nil {
		slog.Error("invalid TRUSTED_PROXIES; trusting no proxy (ClientIP will use the direct remote address)", "error", err)
		_ = engine.SetTrustedProxies(nil)
	}
}

// allowedCORSOrigins parses and validates CORS origins, returning defaults if empty.
func allowedCORSOrigins(raw string) []string {
	const defaultOrigin = "http://localhost:5173"

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{defaultOrigin}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		origin := strings.TrimSpace(part)
		// Never allow a wildcard origin: the API sends credentials
		// (AllowCredentials: true), and "*" + credentials is both invalid per
		// the CORS spec and a security risk. Also drop origins without a valid
		// scheme — gin-contrib/cors PANICS on those, which would crash the server
		// at startup (e.g. a bare host/IP in CORS_ALLOWED_ORIGINS). Fail closed.
		if origin == "" || origin == "*" || !hasURLScheme(origin) {
			continue
		}
		origins = append(origins, origin)
	}

	if len(origins) == 0 {
		return []string{defaultOrigin}
	}

	return origins
}

// hasURLScheme reports whether origin carries a scheme accepted by the CORS
// library (http/https/ws/wss). Origins without a scheme make cors.New() panic,
// so we treat a scheme-less value (e.g. a bare host or IP) as misconfiguration.
func hasURLScheme(origin string) bool {
	for _, scheme := range []string{"http://", "https://", "ws://", "wss://"} {
		if strings.HasPrefix(origin, scheme) {
			return true
		}
	}
	return false
}

// securityHeaders sets baseline hardening response headers on every response.
// The API only ever returns JSON (or CSV attachments), never HTML documents, so
// a strict CSP/frame policy here is safe for the WEB/MOBILE clients that just
// consume the bodies.
func securityHeaders(env string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// HSTS only makes sense over TLS; emit it in production where the API is
		// served via HTTPS so browsers pin the secure scheme.
		if env == "production" {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}

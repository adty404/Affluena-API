package server

import (
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
	"affluena-api/internal/quickentry"
	"affluena-api/internal/recurring"
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
	router := gin.New()
	router.Use(gin.Recovery())

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

	authService := auth.NewService(authRepo, tokenManager, activityUC)
	authHandler := auth.NewHandler(authService)

	walletRepo := wallet.NewRepository(pool)
	walletHandler := wallet.NewHandler(wallet.NewUseCase(walletRepo, activityUC))
	categoryHandler := category.NewHandler(category.NewUseCase(category.NewRepository(pool), activityUC))
	tagHandler := tag.NewHandler(tag.NewUseCase(tag.NewRepository(pool), activityUC))
	budgetUC := budget.NewUseCase(budget.NewRepository(pool), activityUC)
	budgetHandler := budget.NewHandler(budgetUC)

	var alertUC alert.UseCase
	if cfg.SMTPHost != "" && cfg.SMTPPort > 0 {
		smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
		alertUC = alert.NewUseCase(alert.NewRepository(pool), budgetUC, smtpMailer)
	}

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

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")

	// Apply rate limiting to auth endpoints
	authGroup := v1.Group("/auth")
	authGroup.Use(httpx.AuthLimiter.Middleware())
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)

	protected := v1.Group("")
	protected.Use(auth.AuthMiddleware(tokenManager))
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

	protected.GET("/activities", activityHandler.ListActivities)

	protected.POST("/wallets", walletHandler.Create)
	protected.GET("/wallets", walletHandler.List)
	protected.GET("/wallets/:id", walletHandler.Get)
	protected.PUT("/wallets/:id", walletHandler.Update)
	protected.DELETE("/wallets/:id", walletHandler.Delete)
	protected.POST("/wallets/:id/invites", walletHandler.InviteMember)
	protected.PATCH("/wallets/:id/members/:member_id", walletHandler.RespondInvite)
	protected.GET("/wallets/:id/members", walletHandler.ListMembers)
	protected.GET("/wallets/:id/analytics", walletHandler.Analytics)

	protected.POST("/categories", categoryHandler.Create)
	protected.GET("/categories", categoryHandler.List)
	protected.GET("/categories/:id", categoryHandler.Get)
	protected.PUT("/categories/:id", categoryHandler.Update)
	protected.DELETE("/categories/:id", categoryHandler.Delete)

	protected.POST("/tags", tagHandler.Create)
	protected.GET("/tags", tagHandler.List)
	protected.GET("/tags/:id", tagHandler.Get)
	protected.PUT("/tags/:id", tagHandler.Update)
	protected.DELETE("/tags/:id", tagHandler.Delete)

	protected.POST("/transactions/split", splitBillHandler.Split)
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
		if origin != "" {
			origins = append(origins, origin)
		}
	}

	if len(origins) == 0 {
		return []string{defaultOrigin}
	}

	return origins
}

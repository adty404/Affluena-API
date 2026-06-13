package server

import (
	"net/http"

	"affluena-api/internal/apilog"
	"affluena-api/internal/auth"
	"affluena-api/internal/budget"
	"affluena-api/internal/category"
	"affluena-api/internal/config"
	"affluena-api/internal/dashboard"
	"affluena-api/internal/debt"
	"affluena-api/internal/export"
	"affluena-api/internal/goal"
	"affluena-api/internal/quickentry"
	"affluena-api/internal/recurring"
	"affluena-api/internal/tag"
	"affluena-api/internal/tracker"
	"affluena-api/internal/transaction"
	"affluena-api/internal/wallet"

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
	router.Use(apilog.APILogMiddleware(apilogRepo))

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenDuration, cfg.RefreshTokenDuration)
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHandler(authService)

	walletHandler := wallet.NewHandler(wallet.NewUseCase(wallet.NewRepository(pool)))
	categoryHandler := category.NewHandler(category.NewUseCase(category.NewRepository(pool)))
	tagHandler := tag.NewHandler(tag.NewUseCase(tag.NewRepository(pool)))
	transactionRepo := transaction.NewRepository(pool)
	transactionHandler := transaction.NewHandler(transaction.NewUseCase(transactionRepo))
	quickEntryHandler := quickentry.NewHandler(quickentry.NewUseCase(quickentry.NewRepository(pool), transaction.NewUseCase(transactionRepo)))
	budgetHandler := budget.NewHandler(budget.NewUseCase(budget.NewRepository(pool)))
	dashboardHandler := dashboard.NewHandler(dashboard.NewUseCase(dashboard.NewRepository(pool)))
	debtHandler := debt.NewHandler(debt.NewUseCase(debt.NewRepository(pool, transactionRepo)))
	goalHandler := goal.NewHandler(goal.NewUsecase(goal.NewRepository(pool)))
	recurringHandler := recurring.NewHandler(recurring.NewUseCase(recurring.NewRepository(pool, transactionRepo)))
	trackerHandler := tracker.NewHandler(tracker.NewUseCase(
		tracker.NewInstallmentRepository(pool, transactionRepo),
		tracker.NewSubscriptionRepository(pool, transactionRepo),
	))
	exportHandler := export.NewHandler(export.NewUseCase(export.NewRepository(pool)))

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	protected := v1.Group("")
	protected.Use(auth.AuthMiddleware(tokenManager))
	protected.GET("/auth/me", authHandler.Me)

	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/dashboard/cashflow-trend", dashboardHandler.CashflowTrend)
	protected.GET("/dashboard/expense-distribution", dashboardHandler.ExpenseDistribution)
	protected.GET("/dashboard/forecast", dashboardHandler.Forecast)

	protected.GET("/export/csv", exportHandler.ExportCSV)

	protected.POST("/wallets", walletHandler.Create)
	protected.GET("/wallets", walletHandler.List)
	protected.GET("/wallets/:id", walletHandler.Get)
	protected.PUT("/wallets/:id", walletHandler.Update)
	protected.DELETE("/wallets/:id", walletHandler.Delete)
	protected.POST("/wallets/:id/invites", walletHandler.InviteMember)
	protected.PATCH("/wallets/:id/members/:member_id", walletHandler.RespondInvite)

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

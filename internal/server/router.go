package server

import (
	"net/http"

	"affluena/internal/auth"
	"affluena/internal/budget"
	"affluena/internal/category"
	"affluena/internal/config"
	"affluena/internal/quickentry"
	"affluena/internal/recurring"
	"affluena/internal/tracker"
	"affluena/internal/transaction"
	"affluena/internal/wallet"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenDuration, cfg.RefreshTokenDuration)
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHandler(authService)

	walletHandler := wallet.NewHandler(wallet.NewRepository(pool))
	categoryHandler := category.NewHandler(category.NewRepository(pool))
	transactionRepo := transaction.NewRepository(pool)
	transactionHandler := transaction.NewHandler(transactionRepo)
	quickEntryHandler := quickentry.NewHandler(quickentry.NewRepository(pool), transactionRepo)
	budgetHandler := budget.NewHandler(budget.NewRepository(pool))
	recurringHandler := recurring.NewHandler(recurring.NewRepository(pool, transactionRepo))
	trackerHandler := tracker.NewHandler(
		tracker.NewInstallmentRepository(pool, transactionRepo),
		tracker.NewSubscriptionRepository(pool, transactionRepo),
	)

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

	protected.POST("/wallets", walletHandler.Create)
	protected.GET("/wallets", walletHandler.List)
	protected.GET("/wallets/:id", walletHandler.Get)
	protected.PUT("/wallets/:id", walletHandler.Update)
	protected.DELETE("/wallets/:id", walletHandler.Delete)

	protected.POST("/categories", categoryHandler.Create)
	protected.GET("/categories", categoryHandler.List)
	protected.GET("/categories/:id", categoryHandler.Get)
	protected.PUT("/categories/:id", categoryHandler.Update)
	protected.DELETE("/categories/:id", categoryHandler.Delete)

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

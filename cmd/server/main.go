package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"pay-chain.backend/internal/config"
	"pay-chain.backend/internal/infrastructure/jobs"
	"pay-chain.backend/internal/infrastructure/repositories"
	"pay-chain.backend/internal/interfaces/http/handlers"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.URL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("⚠️ Database not available: %v (endpoints will return errors)", err)
	} else {
		log.Println("✅ Connected to PostgreSQL")
	}

	// Initialize JWT service
	jwtService := jwt.NewJWTService(
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	emailVerifRepo := repositories.NewEmailVerificationRepository(db)
	merchantRepo := repositories.NewMerchantRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentEventRepo := repositories.NewPaymentEventRepository(db)
	walletRepo := repositories.NewWalletRepository(db)
	chainRepo := repositories.NewChainRepository(db)
	tokenRepo := repositories.NewTokenRepository(db)
	smartContractRepo := repositories.NewSmartContractRepository(db)
	paymentRequestRepo := repositories.NewPaymentRequestRepository(db)

	// Initialize usecases
	authUsecase := usecases.NewAuthUsecase(userRepo, emailVerifRepo, walletRepo, jwtService)
	paymentUsecase := usecases.NewPaymentUsecase(paymentRepo, paymentEventRepo, walletRepo, merchantRepo, smartContractRepo)
	merchantUsecase := usecases.NewMerchantUsecase(merchantRepo, userRepo)
	walletUsecase := usecases.NewWalletUsecase(walletRepo, userRepo)
	paymentRequestUsecase := usecases.NewPaymentRequestUsecase(paymentRequestRepo, merchantRepo, walletRepo, smartContractRepo)
	webhookUsecase := usecases.NewWebhookUsecase(paymentRepo, paymentEventRepo, paymentRequestRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authUsecase)
	paymentHandler := handlers.NewPaymentHandler(paymentUsecase)
	merchantHandler := handlers.NewMerchantHandler(merchantUsecase)
	walletHandler := handlers.NewWalletHandler(walletUsecase)
	chainHandler := handlers.NewChainHandler(chainRepo)
	tokenHandler := handlers.NewTokenHandler(tokenRepo)
	smartContractHandler := handlers.NewSmartContractHandler(smartContractRepo)
	paymentRequestHandler := handlers.NewPaymentRequestHandler(paymentRequestUsecase)
	webhookHandler := handlers.NewWebhookHandler(webhookUsecase)

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(jwtService)

	// Start background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expiryJob := jobs.NewPaymentRequestExpiryJob(paymentRequestRepo)
	go expiryJob.Start(ctx)

	// Initialize router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "pay-chain-backend",
			"version": "0.2.0",
		})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Payment routes (protected)
		payments := v1.Group("/payments")
		payments.Use(authMiddleware)
		{
			payments.POST("", paymentHandler.CreatePayment)
			payments.GET("/:id", paymentHandler.GetPayment)
			payments.GET("", paymentHandler.ListPayments)
			payments.GET("/:id/events", paymentHandler.GetPaymentEvents)
		}

		// Payment Request routes (protected for merchants)
		paymentRequests := v1.Group("/payment-requests")
		paymentRequests.Use(authMiddleware)
		{
			paymentRequests.POST("", paymentRequestHandler.CreatePaymentRequest)
			paymentRequests.GET("", paymentRequestHandler.ListPaymentRequests)
			paymentRequests.GET("/:id", paymentRequestHandler.GetPaymentRequest)
		}

		// Public payment request route (for payers)
		v1.GET("/pay/:id", paymentRequestHandler.GetPublicPaymentRequest)

		// Wallet routes (protected)
		wallets := v1.Group("/wallets")
		wallets.Use(authMiddleware)
		{
			wallets.POST("/connect", walletHandler.ConnectWallet)
			wallets.GET("", walletHandler.ListWallets)
			wallets.PUT("/:id/primary", walletHandler.SetPrimaryWallet)
			wallets.DELETE("/:id", walletHandler.DisconnectWallet)
		}

		// Merchant routes (protected)
		merchants := v1.Group("/merchants")
		merchants.Use(authMiddleware)
		{
			merchants.POST("/apply", merchantHandler.ApplyMerchant)
			merchants.GET("/status", merchantHandler.GetMerchantStatus)
		}

		// Chain routes (public)
		chains := v1.Group("/chains")
		{
			chains.GET("", chainHandler.ListChains)
		}

		// Token routes (public)
		tokens := v1.Group("/tokens")
		{
			tokens.GET("", tokenHandler.ListTokens)
			tokens.GET("/stablecoins", tokenHandler.ListStablecoins)
		}

		// Smart Contract routes (public read, protected write)
		contracts := v1.Group("/contracts")
		{
			contracts.GET("", smartContractHandler.ListSmartContracts)
			contracts.GET("/lookup", smartContractHandler.GetContractByChainAndAddress)
			contracts.GET("/:id", smartContractHandler.GetSmartContract)
		}

		// Protected smart contract routes (admin only)
		contractsAdmin := v1.Group("/contracts")
		contractsAdmin.Use(authMiddleware)
		{
			contractsAdmin.POST("", smartContractHandler.CreateSmartContract)
			contractsAdmin.DELETE("/:id", smartContractHandler.DeleteSmartContract)
		}

		// Webhook for indexer (internal)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/indexer", webhookHandler.HandleIndexerWebhook)
		}
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("🛑 Shutting down server...")
		expiryJob.Stop()
		cancel()
	}()

	// Start server
	log.Printf("🚀 Pay-Chain Backend starting on port %s", cfg.Server.Port)
	log.Printf("📚 API: http://localhost:%s/api/v1", cfg.Server.Port)
	log.Printf("❤️ Health: http://localhost:%s/health", cfg.Server.Port)

	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

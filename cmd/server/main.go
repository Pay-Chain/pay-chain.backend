package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"pay-chain.backend/internal/config"
	"pay-chain.backend/internal/infrastructure/blockchain"
	"pay-chain.backend/internal/infrastructure/jobs"
	"pay-chain.backend/internal/infrastructure/repositories"
	"pay-chain.backend/internal/interfaces/http/handlers"
	"pay-chain.backend/internal/interfaces/http/middleware"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/jwt"
	"pay-chain.backend/pkg/logger"
	"pay-chain.backend/pkg/redis"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize Logger
	logger.Init(cfg.Server.Env)
	logger.Info(context.Background(), "Logger initialized", zap.String("env", cfg.Server.Env))

	// Initialize Redis
	if err := redis.Init(cfg.Redis.URL, cfg.Redis.PASSWORD); err != nil {
		logger.Error(context.Background(), "Failed to initialize Redis", zap.Error(err))
		// Fail hard if Redis is critical for Idempotency?
		// For now, log error but maybe don't crash if idempotent is critical.
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Info(context.Background(), "Redis initialized")

	// Set Gin mode
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database using GORM
	dsn := cfg.Database.URL()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		PrepareStmt: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get generic database object: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Printf("⚠️ Database not available: %v (endpoints will return errors)", err)
	} else {
		log.Println("✅ Connected to PostgreSQL via GORM")
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
	tokenRepo := repositories.NewTokenRepository(db, chainRepo)
	paymentBridgeRepo := repositories.NewPaymentBridgeRepository(db)
	bridgeConfigRepo := repositories.NewBridgeConfigRepository(db)
	feeConfigRepo := repositories.NewFeeConfigRepository(db)
	routePolicyRepo := repositories.NewRoutePolicyRepository(db)
	layerZeroConfigRepo := repositories.NewLayerZeroConfigRepository(db)
	smartContractRepo := repositories.NewSmartContractRepository(db, chainRepo)
	paymentRequestRepo := repositories.NewPaymentRequestRepository(db)
	teamRepo := repositories.NewTeamRepository(db)
	apiKeyRepo := repositories.NewApiKeyRepository(db) // Added
	uow := repositories.NewUnitOfWork(db)

	// Initialize Session Store
	sessionStore, err := redis.NewSessionStore(cfg.Security.SessionEncryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize session store: %v", err)
	}

	// Initialize blockchain client factory
	clientFactory := blockchain.NewClientFactory()

	// Initialize usecases
	authUsecase := usecases.NewAuthUsecase(userRepo, emailVerifRepo, walletRepo, chainRepo, jwtService)
	// ApiKeyUsecase needs Config for Encryption Key
	apiKeyUsecase := usecases.NewApiKeyUsecase(apiKeyRepo, userRepo, cfg.Security.ApiKeyEncryptionKey)
	paymentUsecase := usecases.NewPaymentUsecase(paymentRepo, paymentEventRepo, walletRepo, merchantRepo, smartContractRepo, chainRepo, tokenRepo, bridgeConfigRepo, feeConfigRepo, routePolicyRepo, uow, clientFactory)
	// PaymentAppUsecase needs PaymentUsecase, UserRepo, WalletRepo, ChainRepo
	paymentAppUsecase := usecases.NewPaymentAppUsecase(paymentUsecase, userRepo, walletRepo, chainRepo)
	merchantUsecase := usecases.NewMerchantUsecase(merchantRepo, userRepo)
	walletUsecase := usecases.NewWalletUsecase(walletRepo, userRepo, chainRepo)
	paymentRequestUsecase := usecases.NewPaymentRequestUsecase(paymentRequestRepo, merchantRepo, walletRepo, chainRepo, smartContractRepo, tokenRepo)
	webhookUsecase := usecases.NewWebhookUsecase(paymentRepo, paymentEventRepo, paymentRequestRepo, uow)
	onchainAdapterUsecase := usecases.NewOnchainAdapterUsecase(chainRepo, smartContractRepo, clientFactory, cfg.Blockchain.OwnerPrivateKey)
	contractConfigAuditUsecase := usecases.NewContractConfigAuditUsecase(chainRepo, smartContractRepo, clientFactory)
	crosschainConfigUsecase := usecases.NewCrosschainConfigUsecase(chainRepo, tokenRepo, smartContractRepo, clientFactory, onchainAdapterUsecase)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authUsecase, sessionStore)
	paymentHandler := handlers.NewPaymentHandler(paymentUsecase)
	merchantHandler := handlers.NewMerchantHandler(merchantUsecase)
	walletHandler := handlers.NewWalletHandler(walletUsecase)
	chainHandler := handlers.NewChainHandler(chainRepo)
	tokenHandler := handlers.NewTokenHandler(tokenRepo, chainRepo)
	smartContractHandler := handlers.NewSmartContractHandler(smartContractRepo, chainRepo)
	paymentRequestHandler := handlers.NewPaymentRequestHandler(paymentRequestUsecase)
	webhookHandler := handlers.NewWebhookHandler(webhookUsecase)
	adminHandler := handlers.NewAdminHandler(userRepo, merchantRepo, paymentRepo)
	teamHandler := handlers.NewTeamHandler(teamRepo)
	apiKeyHandler := handlers.NewApiKeyHandler(apiKeyUsecase)             // Added
	paymentAppHandler := handlers.NewPaymentAppHandler(paymentAppUsecase) // Added
	paymentConfigHandler := handlers.NewPaymentConfigHandler(paymentBridgeRepo, bridgeConfigRepo, feeConfigRepo, chainRepo, tokenRepo)
	onchainAdapterHandler := handlers.NewOnchainAdapterHandler(onchainAdapterUsecase)
	contractConfigAuditHandler := handlers.NewContractConfigAuditHandler(contractConfigAuditUsecase)
	crosschainConfigHandler := handlers.NewCrosschainConfigHandler(crosschainConfigUsecase)
	crosschainPolicyHandler := handlers.NewCrosschainPolicyHandler(routePolicyRepo, layerZeroConfigRepo, chainRepo)

	// Create dual auth middleware
	dualAuthMiddleware := middleware.DualAuthMiddleware(jwtService, apiKeyUsecase, sessionStore) // Added

	// Start background jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expiryJob := jobs.NewPaymentRequestExpiryJob(paymentRequestRepo)
	go expiryJob.Start(ctx)

	// Initialize router
	// Initialize router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.LoggerMiddleware())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

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
			auth.GET("/session-expiry", authHandler.GetSessionExpiry)
			auth.GET("/me", dualAuthMiddleware, authHandler.GetMe) // Updated to Dual Auth
			auth.POST("/change-password", dualAuthMiddleware, authHandler.ChangePassword)
		}

		// Payment routes (protected)
		payments := v1.Group("/payments")
		payments.Use(dualAuthMiddleware) // Updated to Dual Auth
		{
			// Apply Idempotency only to Create Payment
			payments.POST("", middleware.IdempotencyMiddleware(), paymentHandler.CreatePayment)
			payments.GET("/:id", paymentHandler.GetPayment)
			payments.GET("", paymentHandler.ListPayments)
			payments.GET("/:id/events", paymentHandler.GetPaymentEvents)
		}

		// Payment Request routes (protected for merchants)
		paymentRequests := v1.Group("/payment-requests")
		paymentRequests.Use(dualAuthMiddleware) // Updated to Dual Auth
		{
			paymentRequests.POST("", paymentRequestHandler.CreatePaymentRequest)
			paymentRequests.GET("", paymentRequestHandler.ListPaymentRequests)
			paymentRequests.GET("/:id", paymentRequestHandler.GetPaymentRequest)
		}

		// Public payment request route (for payers)
		v1.GET("/pay/:id", paymentRequestHandler.GetPublicPaymentRequest)

		// Wallet routes (protected)
		wallets := v1.Group("/wallets")
		wallets.Use(dualAuthMiddleware) // Updated to Dual Auth
		{
			wallets.POST("/connect", walletHandler.ConnectWallet)
			wallets.GET("", walletHandler.ListWallets)
			wallets.PUT("/:id/primary", walletHandler.SetPrimaryWallet)
			wallets.DELETE("/:id", walletHandler.DisconnectWallet)
		}

		// Merchant routes (protected)
		merchants := v1.Group("/merchants")
		merchants.Use(dualAuthMiddleware) // Updated to Dual Auth
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
			tokens.GET("", tokenHandler.ListSupportedTokens)
			tokens.GET("/stablecoins", tokenHandler.ListStablecoins)
		}

		// Smart Contract routes (public read, protected write)
		contracts := v1.Group("/contracts")
		{
			contracts.GET("", smartContractHandler.ListSmartContracts)
			contracts.GET("/lookup", smartContractHandler.GetContractByChainAndAddress)
			contracts.GET("/:id", smartContractHandler.GetSmartContract)
		}

		// Team routes (public read)
		teams := v1.Group("/teams")
		{
			teams.GET("", teamHandler.ListPublicTeams)
		}

		// Payment config routes (public read)
		paymentBridges := v1.Group("/payment-bridges")
		{
			paymentBridges.GET("", paymentConfigHandler.ListPaymentBridges)
		}
		bridgeConfigs := v1.Group("/bridge-configs")
		{
			bridgeConfigs.GET("", paymentConfigHandler.ListBridgeConfigs)
		}
		feeConfigs := v1.Group("/fee-configs")
		{
			feeConfigs.GET("", paymentConfigHandler.ListFeeConfigs)
		}

		// Protected smart contract routes (admin only)
		contractsAdmin := v1.Group("/contracts")
		contractsAdmin.Use(dualAuthMiddleware) // Updated to Dual Auth
		{
			contractsAdmin.POST("", smartContractHandler.CreateSmartContract)
			contractsAdmin.PUT("/:id", smartContractHandler.UpdateSmartContract)
			contractsAdmin.DELETE("/:id", smartContractHandler.DeleteSmartContract)
		}

		// API Key routes (protected)
		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(dualAuthMiddleware)
		{
			apiKeys.POST("", apiKeyHandler.CreateApiKey)
			apiKeys.GET("", apiKeyHandler.ListApiKeys)
			apiKeys.DELETE("/:id", apiKeyHandler.RevokeApiKey)
		}

		// Payment App (Public App Endpoint)
		// It uses DualAuthMiddleware (Api Key + Sig) OR potentially Wallet Sig (future).
		// For now, we enforce DualAuthMiddleware.
		paymentApp := v1.Group("/payment-app")
		paymentApp.Use(dualAuthMiddleware)
		{
			paymentApp.POST("", paymentAppHandler.CreatePaymentApp)
		}

		// Webhook for indexer (internal)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/indexer", webhookHandler.HandleIndexerWebhook)
		}

		// Admin routes (protected)
		admin := v1.Group("/admin")
		admin.Use(dualAuthMiddleware, middleware.RequireAdmin())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.GET("/merchants", adminHandler.ListMerchants)
			admin.PUT("/merchants/:id/status", adminHandler.UpdateMerchantStatus)
			admin.GET("/stats", adminHandler.GetStats)

			// Chain management
			admin.POST("/chains", chainHandler.CreateChain)
			admin.PUT("/chains/:id", chainHandler.UpdateChain)
			admin.DELETE("/chains/:id", chainHandler.DeleteChain)

			// RPC management
			rpcHandler := handlers.NewRpcHandler(chainRepo)
			admin.GET("/rpcs", rpcHandler.ListRPCs)

			// Supported Token management
			admin.GET("/tokens", tokenHandler.ListSupportedTokens)
			admin.POST("/tokens", tokenHandler.CreateToken)
			admin.PUT("/tokens/:id", tokenHandler.UpdateToken)
			admin.DELETE("/tokens/:id", tokenHandler.DeleteToken)

			// Team management
			admin.GET("/teams", teamHandler.ListAdminTeams)
			admin.POST("/teams", teamHandler.CreateTeam)
			admin.PUT("/teams/:id", teamHandler.UpdateTeam)
			admin.DELETE("/teams/:id", teamHandler.DeleteTeam)

			// Payment bridge management
			admin.GET("/payment-bridges", paymentConfigHandler.ListPaymentBridges)
			admin.POST("/payment-bridges", paymentConfigHandler.CreatePaymentBridge)
			admin.PUT("/payment-bridges/:id", paymentConfigHandler.UpdatePaymentBridge)
			admin.DELETE("/payment-bridges/:id", paymentConfigHandler.DeletePaymentBridge)

			// Bridge config management
			admin.GET("/bridge-configs", paymentConfigHandler.ListBridgeConfigs)
			admin.POST("/bridge-configs", paymentConfigHandler.CreateBridgeConfig)
			admin.PUT("/bridge-configs/:id", paymentConfigHandler.UpdateBridgeConfig)
			admin.DELETE("/bridge-configs/:id", paymentConfigHandler.DeleteBridgeConfig)

			// Fee config management
			admin.GET("/fee-configs", paymentConfigHandler.ListFeeConfigs)
			admin.POST("/fee-configs", paymentConfigHandler.CreateFeeConfig)
			admin.PUT("/fee-configs/:id", paymentConfigHandler.UpdateFeeConfig)
			admin.DELETE("/fee-configs/:id", paymentConfigHandler.DeleteFeeConfig)

			// On-chain adapter management
			admin.GET("/onchain-adapters/status", onchainAdapterHandler.GetStatus)
			admin.POST("/onchain-adapters/register", onchainAdapterHandler.RegisterAdapter)
			admin.POST("/onchain-adapters/default-bridge", onchainAdapterHandler.SetDefaultBridgeType)
			admin.POST("/onchain-adapters/hyperbridge-config", onchainAdapterHandler.SetHyperbridgeConfig)
			admin.POST("/onchain-adapters/ccip-config", onchainAdapterHandler.SetCCIPConfig)
			admin.POST("/onchain-adapters/layerzero-config", onchainAdapterHandler.SetLayerZeroConfig)
			admin.GET("/contracts/config-check", contractConfigAuditHandler.Check)
			admin.GET("/contracts/:id/config-check", contractConfigAuditHandler.CheckByContract)
			admin.GET("/crosschain-config/overview", crosschainConfigHandler.Overview)
			admin.GET("/crosschain-config/preflight", crosschainConfigHandler.Preflight)
			admin.POST("/crosschain-config/recheck", crosschainConfigHandler.Recheck)
			admin.POST("/crosschain-config/recheck-bulk", crosschainConfigHandler.RecheckBulk)
			admin.POST("/crosschain-config/auto-fix", crosschainConfigHandler.AutoFix)
			admin.POST("/crosschain-config/auto-fix-bulk", crosschainConfigHandler.AutoFixBulk)

			// Cross-chain route policy management
			admin.GET("/route-policies", crosschainPolicyHandler.ListRoutePolicies)
			admin.POST("/route-policies", crosschainPolicyHandler.CreateRoutePolicy)
			admin.PUT("/route-policies/:id", crosschainPolicyHandler.UpdateRoutePolicy)
			admin.DELETE("/route-policies/:id", crosschainPolicyHandler.DeleteRoutePolicy)

			// LayerZero route config management (DB level)
			admin.GET("/layerzero-configs", crosschainPolicyHandler.ListLayerZeroConfigs)
			admin.POST("/layerzero-configs", crosschainPolicyHandler.CreateLayerZeroConfig)
			admin.PUT("/layerzero-configs/:id", crosschainPolicyHandler.UpdateLayerZeroConfig)
			admin.DELETE("/layerzero-configs/:id", crosschainPolicyHandler.DeleteLayerZeroConfig)
		}
	}

	// Print all registered routes for debugging
	log.Println("📋 Registered Routes:")
	for _, route := range r.Routes() {
		log.Printf("   %s %s", route.Method, route.Path)
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

package main

import (
	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/interfaces/http/handlers"
	"pay-chain.backend/internal/interfaces/http/middleware"
)

type routeDeps struct {
	authHandler                *handlers.AuthHandler
	paymentHandler             *handlers.PaymentHandler
	merchantHandler            *handlers.MerchantHandler
	walletHandler              *handlers.WalletHandler
	chainHandler               *handlers.ChainHandler
	tokenHandler               *handlers.TokenHandler
	smartContractHandler       *handlers.SmartContractHandler
	paymentRequestHandler      *handlers.PaymentRequestHandler
	webhookHandler             *handlers.WebhookHandler
	adminHandler               *handlers.AdminHandler
	teamHandler                *handlers.TeamHandler
	apiKeyHandler              *handlers.ApiKeyHandler
	paymentAppHandler          *handlers.PaymentAppHandler
	paymentConfigHandler       *handlers.PaymentConfigHandler
	onchainAdapterHandler      *handlers.OnchainAdapterHandler
	contractConfigAuditHandler *handlers.ContractConfigAuditHandler
	crosschainConfigHandler    *handlers.CrosschainConfigHandler
	crosschainPolicyHandler    *handlers.CrosschainPolicyHandler
	routeErrorHandler          *handlers.RouteErrorHandler
	rpcHandler                 *handlers.RpcHandler
	dualAuthMiddleware         gin.HandlerFunc
}

func registerAPIV1Routes(r *gin.Engine, d routeDeps) {
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", d.authHandler.Register)
			auth.POST("/login", d.authHandler.Login)
			auth.POST("/verify-email", d.authHandler.VerifyEmail)
			auth.POST("/refresh", d.authHandler.RefreshToken)
			auth.GET("/session-expiry", d.authHandler.GetSessionExpiry)
			auth.GET("/me", d.dualAuthMiddleware, d.authHandler.GetMe)
			auth.POST("/change-password", d.dualAuthMiddleware, d.authHandler.ChangePassword)
		}

		// Payment routes (protected)
		payments := v1.Group("/payments")
		payments.Use(d.dualAuthMiddleware)
		{
			payments.POST("", middleware.IdempotencyMiddleware(), d.paymentHandler.CreatePayment)
			payments.GET("/:id", d.paymentHandler.GetPayment)
			payments.GET("", d.paymentHandler.ListPayments)
			payments.GET("/:id/events", d.paymentHandler.GetPaymentEvents)
		}

		// Payment Request routes (protected for merchants)
		paymentRequests := v1.Group("/payment-requests")
		paymentRequests.Use(d.dualAuthMiddleware)
		{
			paymentRequests.POST("", d.paymentRequestHandler.CreatePaymentRequest)
			paymentRequests.GET("", d.paymentRequestHandler.ListPaymentRequests)
			paymentRequests.GET("/:id", d.paymentRequestHandler.GetPaymentRequest)
		}

		// Public payment request route (for payers)
		v1.GET("/pay/:id", d.paymentRequestHandler.GetPublicPaymentRequest)

		// Wallet routes (protected)
		wallets := v1.Group("/wallets")
		wallets.Use(d.dualAuthMiddleware)
		{
			wallets.POST("/connect", d.walletHandler.ConnectWallet)
			wallets.GET("", d.walletHandler.ListWallets)
			wallets.PUT("/:id/primary", d.walletHandler.SetPrimaryWallet)
			wallets.DELETE("/:id", d.walletHandler.DisconnectWallet)
		}

		// Merchant routes (protected)
		merchants := v1.Group("/merchants")
		merchants.Use(d.dualAuthMiddleware)
		{
			merchants.POST("/apply", d.merchantHandler.ApplyMerchant)
			merchants.GET("/status", d.merchantHandler.GetMerchantStatus)
		}

		// Chain routes (public)
		chains := v1.Group("/chains")
		{
			chains.GET("", d.chainHandler.ListChains)
		}

		// Token routes (public)
		tokens := v1.Group("/tokens")
		{
			tokens.GET("", d.tokenHandler.ListSupportedTokens)
			tokens.GET("/stablecoins", d.tokenHandler.ListStablecoins)
		}

		// Smart Contract routes (public read, protected write)
		contracts := v1.Group("/contracts")
		{
			contracts.GET("", d.smartContractHandler.ListSmartContracts)
			contracts.GET("/lookup", d.smartContractHandler.GetContractByChainAndAddress)
			contracts.GET("/:id", d.smartContractHandler.GetSmartContract)
		}

		// Team routes (public read)
		teams := v1.Group("/teams")
		{
			teams.GET("", d.teamHandler.ListPublicTeams)
		}

		// Payment config routes (public read)
		paymentBridges := v1.Group("/payment-bridges")
		{
			paymentBridges.GET("", d.paymentConfigHandler.ListPaymentBridges)
		}
		bridgeConfigs := v1.Group("/bridge-configs")
		{
			bridgeConfigs.GET("", d.paymentConfigHandler.ListBridgeConfigs)
		}
		feeConfigs := v1.Group("/fee-configs")
		{
			feeConfigs.GET("", d.paymentConfigHandler.ListFeeConfigs)
		}

		// Protected smart contract routes (admin only)
		contractsAdmin := v1.Group("/contracts")
		contractsAdmin.Use(d.dualAuthMiddleware)
		{
			contractsAdmin.POST("", d.smartContractHandler.CreateSmartContract)
			contractsAdmin.PUT("/:id", d.smartContractHandler.UpdateSmartContract)
			contractsAdmin.DELETE("/:id", d.smartContractHandler.DeleteSmartContract)
		}

		// API Key routes (protected)
		apiKeys := v1.Group("/api-keys")
		apiKeys.Use(d.dualAuthMiddleware)
		{
			apiKeys.POST("", d.apiKeyHandler.CreateApiKey)
			apiKeys.GET("", d.apiKeyHandler.ListApiKeys)
			apiKeys.DELETE("/:id", d.apiKeyHandler.RevokeApiKey)
		}

		// Payment App (Public App Endpoint)
		paymentApp := v1.Group("/payment-app")
		paymentApp.Use(d.dualAuthMiddleware)
		{
			paymentApp.POST("", d.paymentAppHandler.CreatePaymentApp)
			paymentApp.GET("/diagnostics/route-error/:paymentId", d.routeErrorHandler.GetRouteError)
		}

		// Webhook for indexer (internal)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/indexer", d.webhookHandler.HandleIndexerWebhook)
		}

		// Admin routes (protected)
		admin := v1.Group("/admin")
		admin.Use(d.dualAuthMiddleware, middleware.RequireAdmin())
		{
			admin.GET("/users", d.adminHandler.ListUsers)
			admin.GET("/merchants", d.adminHandler.ListMerchants)
			admin.PUT("/merchants/:id/status", d.adminHandler.UpdateMerchantStatus)
			admin.GET("/stats", d.adminHandler.GetStats)

			admin.POST("/chains", d.chainHandler.CreateChain)
			admin.PUT("/chains/:id", d.chainHandler.UpdateChain)
			admin.DELETE("/chains/:id", d.chainHandler.DeleteChain)

			admin.GET("/rpcs", d.rpcHandler.ListRPCs)
			admin.POST("/rpcs", d.rpcHandler.CreateRPC)
			admin.PUT("/rpcs/:id", d.rpcHandler.UpdateRPC)
			admin.DELETE("/rpcs/:id", d.rpcHandler.DeleteRPC)

			admin.GET("/tokens", d.tokenHandler.ListSupportedTokens)
			admin.POST("/tokens", d.tokenHandler.CreateToken)
			admin.PUT("/tokens/:id", d.tokenHandler.UpdateToken)
			admin.DELETE("/tokens/:id", d.tokenHandler.DeleteToken)

			admin.GET("/teams", d.teamHandler.ListAdminTeams)
			admin.POST("/teams", d.teamHandler.CreateTeam)
			admin.PUT("/teams/:id", d.teamHandler.UpdateTeam)
			admin.DELETE("/teams/:id", d.teamHandler.DeleteTeam)

			admin.GET("/payment-bridges", d.paymentConfigHandler.ListPaymentBridges)
			admin.POST("/payment-bridges", d.paymentConfigHandler.CreatePaymentBridge)
			admin.PUT("/payment-bridges/:id", d.paymentConfigHandler.UpdatePaymentBridge)
			admin.DELETE("/payment-bridges/:id", d.paymentConfigHandler.DeletePaymentBridge)

			admin.GET("/bridge-configs", d.paymentConfigHandler.ListBridgeConfigs)
			admin.POST("/bridge-configs", d.paymentConfigHandler.CreateBridgeConfig)
			admin.PUT("/bridge-configs/:id", d.paymentConfigHandler.UpdateBridgeConfig)
			admin.DELETE("/bridge-configs/:id", d.paymentConfigHandler.DeleteBridgeConfig)

			admin.GET("/fee-configs", d.paymentConfigHandler.ListFeeConfigs)
			admin.POST("/fee-configs", d.paymentConfigHandler.CreateFeeConfig)
			admin.PUT("/fee-configs/:id", d.paymentConfigHandler.UpdateFeeConfig)
			admin.DELETE("/fee-configs/:id", d.paymentConfigHandler.DeleteFeeConfig)

			admin.GET("/onchain-adapters/status", d.onchainAdapterHandler.GetStatus)
			admin.POST("/onchain-adapters/register", d.onchainAdapterHandler.RegisterAdapter)
			admin.POST("/onchain-adapters/default-bridge", d.onchainAdapterHandler.SetDefaultBridgeType)
			admin.POST("/onchain-adapters/hyperbridge-config", d.onchainAdapterHandler.SetHyperbridgeConfig)
			admin.POST("/onchain-adapters/ccip-config", d.onchainAdapterHandler.SetCCIPConfig)
			admin.POST("/onchain-adapters/layerzero-config", d.onchainAdapterHandler.SetLayerZeroConfig)
			admin.GET("/contracts/config-check", d.contractConfigAuditHandler.Check)
			admin.GET("/contracts/:id/config-check", d.contractConfigAuditHandler.CheckByContract)
			admin.GET("/crosschain-config/overview", d.crosschainConfigHandler.Overview)
			admin.GET("/crosschain-config/preflight", d.crosschainConfigHandler.Preflight)
			admin.POST("/crosschain-config/recheck", d.crosschainConfigHandler.Recheck)
			admin.POST("/crosschain-config/recheck-bulk", d.crosschainConfigHandler.RecheckBulk)
			admin.POST("/crosschain-config/auto-fix", d.crosschainConfigHandler.AutoFix)
			admin.POST("/crosschain-config/auto-fix-bulk", d.crosschainConfigHandler.AutoFixBulk)

			admin.GET("/route-policies", d.crosschainPolicyHandler.ListRoutePolicies)
			admin.POST("/route-policies", d.crosschainPolicyHandler.CreateRoutePolicy)
			admin.PUT("/route-policies/:id", d.crosschainPolicyHandler.UpdateRoutePolicy)
			admin.DELETE("/route-policies/:id", d.crosschainPolicyHandler.DeleteRoutePolicy)

			admin.GET("/layerzero-configs", d.crosschainPolicyHandler.ListLayerZeroConfigs)
			admin.POST("/layerzero-configs", d.crosschainPolicyHandler.CreateLayerZeroConfig)
			admin.PUT("/layerzero-configs/:id", d.crosschainPolicyHandler.UpdateLayerZeroConfig)
			admin.DELETE("/layerzero-configs/:id", d.crosschainPolicyHandler.DeleteLayerZeroConfig)

			admin.GET("/diagnostics/route-error/:paymentId", d.routeErrorHandler.GetRouteError)
		}
	}
}

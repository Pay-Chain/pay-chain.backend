package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
)

type MerchantSettlementHandler struct {
	merchantRepo          repositories.MerchantRepository
	settlementProfileRepo repositories.MerchantSettlementProfileRepository
	chainRepo             repositories.ChainRepository
	tokenRepo             repositories.TokenRepository
}

func NewMerchantSettlementHandler(
	merchantRepo repositories.MerchantRepository,
	settlementProfileRepo repositories.MerchantSettlementProfileRepository,
	chainRepo repositories.ChainRepository,
	tokenRepo repositories.TokenRepository,
) *MerchantSettlementHandler {
	return &MerchantSettlementHandler{
		merchantRepo:          merchantRepo,
		settlementProfileRepo: settlementProfileRepo,
		chainRepo:             chainRepo,
		tokenRepo:             tokenRepo,
	}
}

func (h *MerchantSettlementHandler) GetMySettlementProfile(c *gin.Context) {
	merchant, ok := h.resolveMerchant(c)
	if !ok {
		return
	}

	profile, err := h.settlementProfileRepo.GetByMerchantID(c.Request.Context(), merchant.ID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Success(c, http.StatusOK, gin.H{
				"merchant_id": merchant.ID.String(),
				"configured":  false,
			})
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"configured":          true,
		"id":                  profile.ID.String(),
		"merchant_id":         profile.MerchantID.String(),
		"invoice_currency":    profile.InvoiceCurrency,
		"dest_chain":          profile.DestChain,
		"dest_token":          profile.DestToken,
		"dest_wallet":         profile.DestWallet,
		"bridge_token_symbol": profile.BridgeTokenSymbol,
		"created_at":          profile.CreatedAt,
		"updated_at":          profile.UpdatedAt,
	})
}

func (h *MerchantSettlementHandler) UpsertMySettlementProfile(c *gin.Context) {
	merchant, ok := h.resolveMerchant(c)
	if !ok {
		return
	}

	var req struct {
		InvoiceCurrency   string `json:"invoice_currency"`
		DestChain         string `json:"dest_chain" binding:"required"`
		DestToken         string `json:"dest_token" binding:"required"`
		DestWallet        string `json:"dest_wallet" binding:"required"`
		BridgeTokenSymbol string `json:"bridge_token_symbol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chain, err := resolveChainForSettlement(c.Request.Context(), h.chainRepo, strings.TrimSpace(req.DestChain))
	if err != nil {
		response.Error(c, err)
		return
	}

	destToken, err := h.tokenRepo.GetByAddress(c.Request.Context(), strings.TrimSpace(req.DestToken), chain.ID)
	if err != nil || destToken == nil || !destToken.IsActive {
		response.Error(c, domainerrors.BadRequest("dest_token is not supported on dest_chain"))
		return
	}
	invoiceCurrency := strings.ToUpper(strings.TrimSpace(destToken.Symbol))

	bridgeTokenSymbol := strings.ToUpper(strings.TrimSpace(req.BridgeTokenSymbol))
	if bridgeTokenSymbol == "" {
		bridgeTokenSymbol = "USDC"
	}
	if _, err := h.tokenRepo.GetBySymbol(c.Request.Context(), bridgeTokenSymbol, chain.ID); err != nil {
		response.Error(c, domainerrors.BadRequest("bridge_token_symbol is not supported on dest_chain"))
		return
	}

	existing, err := h.settlementProfileRepo.GetByMerchantID(c.Request.Context(), merchant.ID)
	if err != nil && err != domainerrors.ErrNotFound {
		response.Error(c, err)
		return
	}

	profile := &entities.MerchantSettlementProfile{
		MerchantID:        merchant.ID,
		InvoiceCurrency:   invoiceCurrency,
		DestChain:         chain.GetCAIP2ID(),
		DestToken:         destToken.ContractAddress,
		DestWallet:        strings.TrimSpace(req.DestWallet),
		BridgeTokenSymbol: bridgeTokenSymbol,
		CreatedAt:         time.Now().UTC(),
	}
	if existing != nil {
		profile.ID = existing.ID
		profile.CreatedAt = existing.CreatedAt
	}

	if err := h.settlementProfileRepo.Upsert(c.Request.Context(), profile); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"message":             "Merchant settlement profile updated",
		"configured":          true,
		"merchant_id":         merchant.ID.String(),
		"invoice_currency":    profile.InvoiceCurrency,
		"dest_chain":          profile.DestChain,
		"dest_token":          profile.DestToken,
		"dest_wallet":         profile.DestWallet,
		"bridge_token_symbol": profile.BridgeTokenSymbol,
	})
}

func (h *MerchantSettlementHandler) resolveMerchant(c *gin.Context) (*entities.Merchant, bool) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Error(c, domainerrors.Unauthorized("User not authenticated"))
		return nil, false
	}

	merchant, err := h.merchantRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.Forbidden("merchant account required"))
			return nil, false
		}
		response.Error(c, err)
		return nil, false
	}

	return merchant, true
}

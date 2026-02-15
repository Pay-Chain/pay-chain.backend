package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/pkg/utils"
)

type PaymentConfigHandler struct {
	paymentBridgeRepo repositories.PaymentBridgeRepository
	bridgeConfigRepo  repositories.BridgeConfigRepository
	feeConfigRepo     repositories.FeeConfigRepository
	chainRepo         repositories.ChainRepository
	tokenRepo         repositories.TokenRepository
}

func NewPaymentConfigHandler(
	paymentBridgeRepo repositories.PaymentBridgeRepository,
	bridgeConfigRepo repositories.BridgeConfigRepository,
	feeConfigRepo repositories.FeeConfigRepository,
	chainRepo repositories.ChainRepository,
	tokenRepo repositories.TokenRepository,
) *PaymentConfigHandler {
	return &PaymentConfigHandler{
		paymentBridgeRepo: paymentBridgeRepo,
		bridgeConfigRepo:  bridgeConfigRepo,
		feeConfigRepo:     feeConfigRepo,
		chainRepo:         chainRepo,
		tokenRepo:         tokenRepo,
	}
}

// --- payment bridges ---

func (h *PaymentConfigHandler) ListPaymentBridges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	items, total, err := h.paymentBridgeRepo.List(c.Request.Context(), pagination)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": items,
		"meta":  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	})
}

func (h *PaymentConfigHandler) CreatePaymentBridge(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	item := &entities.PaymentBridge{
		ID:   utils.GenerateUUIDv7(),
		Name: strings.TrimSpace(input.Name),
	}
	if item.Name == "" {
		response.Error(c, domainerrors.BadRequest("name is required"))
		return
	}

	if err := h.paymentBridgeRepo.Create(c.Request.Context(), item); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"bridge": item})
}

func (h *PaymentConfigHandler) UpdatePaymentBridge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridge id"))
		return
	}

	existing, err := h.paymentBridgeRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	existing.Name = strings.TrimSpace(input.Name)
	if existing.Name == "" {
		response.Error(c, domainerrors.BadRequest("name is required"))
		return
	}
	if err := h.paymentBridgeRepo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"bridge": existing})
}

func (h *PaymentConfigHandler) DeletePaymentBridge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridge id"))
		return
	}
	if err := h.paymentBridgeRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "Bridge deleted"})
}

// --- bridge configs ---

func (h *PaymentConfigHandler) ListBridgeConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	sourceChainID, err := h.parseChainQuery(c.Request.Context(), c.Query("sourceChainId"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid sourceChainId"))
		return
	}
	destChainID, err := h.parseChainQuery(c.Request.Context(), c.Query("destChainId"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid destChainId"))
		return
	}
	bridgeID, err := parseUUIDPtr(c.Query("bridgeId"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridgeId"))
		return
	}

	items, total, err := h.bridgeConfigRepo.List(c.Request.Context(), sourceChainID, destChainID, bridgeID, pagination)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": items,
		"meta":  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	})
}

func (h *PaymentConfigHandler) CreateBridgeConfig(c *gin.Context) {
	var input struct {
		BridgeID      string `json:"bridgeId" binding:"required"`
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		RouterAddress string `json:"routerAddress"`
		FeePercentage string `json:"feePercentage"`
		Config        string `json:"config"`
		IsActive      *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	bridgeID, err := uuid.Parse(input.BridgeID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridgeId"))
		return
	}
	sourceChainID, err := h.parseChainID(c.Request.Context(), input.SourceChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid sourceChainId"))
		return
	}
	destChainID, err := h.parseChainID(c.Request.Context(), input.DestChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid destChainId"))
		return
	}

	item := &entities.BridgeConfig{
		ID:            utils.GenerateUUIDv7(),
		BridgeID:      bridgeID,
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		RouterAddress: strings.TrimSpace(input.RouterAddress),
		FeePercentage: strings.TrimSpace(input.FeePercentage),
		Config:        strings.TrimSpace(input.Config),
		IsActive:      true,
	}
	if item.FeePercentage == "" {
		item.FeePercentage = "0"
	}
	if item.Config == "" {
		item.Config = "{}"
	}
	if input.IsActive != nil {
		item.IsActive = *input.IsActive
	}

	if err := h.bridgeConfigRepo.Create(c.Request.Context(), item); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"config": item})
}

func (h *PaymentConfigHandler) UpdateBridgeConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridge config id"))
		return
	}
	existing, err := h.bridgeConfigRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var input struct {
		BridgeID      string `json:"bridgeId" binding:"required"`
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		RouterAddress string `json:"routerAddress"`
		FeePercentage string `json:"feePercentage"`
		Config        string `json:"config"`
		IsActive      *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	bridgeID, err := uuid.Parse(input.BridgeID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridgeId"))
		return
	}
	sourceChainID, err := h.parseChainID(c.Request.Context(), input.SourceChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid sourceChainId"))
		return
	}
	destChainID, err := h.parseChainID(c.Request.Context(), input.DestChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid destChainId"))
		return
	}

	existing.BridgeID = bridgeID
	existing.SourceChainID = sourceChainID
	existing.DestChainID = destChainID
	existing.RouterAddress = strings.TrimSpace(input.RouterAddress)
	existing.FeePercentage = strings.TrimSpace(input.FeePercentage)
	existing.Config = strings.TrimSpace(input.Config)
	if existing.FeePercentage == "" {
		existing.FeePercentage = "0"
	}
	if existing.Config == "" {
		existing.Config = "{}"
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	if err := h.bridgeConfigRepo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"config": existing})
}

func (h *PaymentConfigHandler) DeleteBridgeConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid bridge config id"))
		return
	}
	if err := h.bridgeConfigRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "Bridge config deleted"})
}

// --- fee configs ---

func (h *PaymentConfigHandler) ListFeeConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chainID, err := h.parseChainQuery(c.Request.Context(), c.Query("chainId"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid chainId"))
		return
	}
	tokenID, err := parseUUIDPtr(c.Query("tokenId"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid tokenId"))
		return
	}

	items, total, err := h.feeConfigRepo.List(c.Request.Context(), chainID, tokenID, pagination)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": items,
		"meta":  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	})
}

func (h *PaymentConfigHandler) CreateFeeConfig(c *gin.Context) {
	var input struct {
		ChainID            string  `json:"chainId" binding:"required"`
		TokenID            string  `json:"tokenId" binding:"required"`
		PlatformFeePercent string  `json:"platformFeePercent"`
		FixedBaseFee       string  `json:"fixedBaseFee"`
		MinFee             string  `json:"minFee"`
		MaxFee             *string `json:"maxFee"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chainID, err := h.parseChainID(c.Request.Context(), input.ChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid chainId"))
		return
	}
	tokenID, err := uuid.Parse(input.TokenID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid tokenId"))
		return
	}
	if _, err := h.tokenRepo.GetByID(c.Request.Context(), tokenID); err != nil {
		response.Error(c, domainerrors.BadRequest("tokenId not found"))
		return
	}

	item := &entities.FeeConfig{
		ID:                 utils.GenerateUUIDv7(),
		ChainID:            chainID,
		TokenID:            tokenID,
		PlatformFeePercent: defaultDecimal(input.PlatformFeePercent),
		FixedBaseFee:       defaultDecimal(input.FixedBaseFee),
		MinFee:             defaultDecimal(input.MinFee),
		MaxFee:             input.MaxFee,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := h.feeConfigRepo.Create(c.Request.Context(), item); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"config": item})
}

func (h *PaymentConfigHandler) UpdateFeeConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid fee config id"))
		return
	}
	existing, err := h.feeConfigRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var input struct {
		ChainID            string  `json:"chainId" binding:"required"`
		TokenID            string  `json:"tokenId" binding:"required"`
		PlatformFeePercent string  `json:"platformFeePercent"`
		FixedBaseFee       string  `json:"fixedBaseFee"`
		MinFee             string  `json:"minFee"`
		MaxFee             *string `json:"maxFee"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chainID, err := h.parseChainID(c.Request.Context(), input.ChainID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid chainId"))
		return
	}
	tokenID, err := uuid.Parse(input.TokenID)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid tokenId"))
		return
	}
	if _, err := h.tokenRepo.GetByID(c.Request.Context(), tokenID); err != nil {
		response.Error(c, domainerrors.BadRequest("tokenId not found"))
		return
	}

	existing.ChainID = chainID
	existing.TokenID = tokenID
	existing.PlatformFeePercent = defaultDecimal(input.PlatformFeePercent)
	existing.FixedBaseFee = defaultDecimal(input.FixedBaseFee)
	existing.MinFee = defaultDecimal(input.MinFee)
	existing.MaxFee = input.MaxFee
	existing.UpdatedAt = time.Now()

	if err := h.feeConfigRepo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"config": existing})
}

func (h *PaymentConfigHandler) DeleteFeeConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid fee config id"))
		return
	}
	if err := h.feeConfigRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "Fee config deleted"})
}

func (h *PaymentConfigHandler) parseChainID(ctx context.Context, input string) (uuid.UUID, error) {
	if parsed, err := uuid.Parse(strings.TrimSpace(input)); err == nil {
		return parsed, nil
	}
	chain, err := h.chainRepo.GetByChainID(ctx, strings.TrimSpace(input))
	if err != nil {
		return uuid.Nil, err
	}
	return chain.ID, nil
}

func (h *PaymentConfigHandler) parseChainQuery(ctx context.Context, input string) (*uuid.UUID, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	id, err := h.parseChainID(ctx, input)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseUUIDPtr(input string) (*uuid.UUID, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func defaultDecimal(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

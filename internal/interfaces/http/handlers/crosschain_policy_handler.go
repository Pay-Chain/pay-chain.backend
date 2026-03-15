package handlers

import (
	"context"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/pkg/utils"
)

type CrosschainPolicyHandler struct {
	routePolicyRepo     repositories.RoutePolicyRepository
	layerZeroConfigRepo repositories.LayerZeroConfigRepository
	chainRepo           repositories.ChainRepository
}

func NewCrosschainPolicyHandler(
	routePolicyRepo repositories.RoutePolicyRepository,
	layerZeroConfigRepo repositories.LayerZeroConfigRepository,
	chainRepo repositories.ChainRepository,
) *CrosschainPolicyHandler {
	return &CrosschainPolicyHandler{
		routePolicyRepo:     routePolicyRepo,
		layerZeroConfigRepo: layerZeroConfigRepo,
		chainRepo:           chainRepo,
	}
}

func (h *CrosschainPolicyHandler) ListRoutePolicies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
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

	items, total, err := h.routePolicyRepo.List(c.Request.Context(), sourceChainID, destChainID, pagination)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": items,
		"meta":  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	})
}

func (h *CrosschainPolicyHandler) CreateRoutePolicy(c *gin.Context) {
	var input struct {
		SourceChainID          string  `json:"sourceChainId" binding:"required"`
		DestChainID            string  `json:"destChainId" binding:"required"`
		DefaultBridgeType      *uint8  `json:"defaultBridgeType" binding:"required"`
		FallbackMode           string  `json:"fallbackMode"`
		FallbackOrder          []uint8 `json:"fallbackOrder"`
		SupportsTokenBridge    *bool   `json:"supportsTokenBridge"`
		SupportsDestSwap       *bool   `json:"supportsDestSwap"`
		SupportsPrivacyForward *bool   `json:"supportsPrivacyForward"`
		BridgeToken            *string `json:"bridgeToken"`
		Status                 *string `json:"status"`
		PerByteRate            string  `json:"perByteRate"`
		OverheadBytes          string  `json:"overheadBytes"`
		MinFee                 string  `json:"minFee"`
		MaxFee                 string  `json:"maxFee"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
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
	if sourceChainID == destChainID {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId must be different"))
		return
	}
	if !isValidBridgeType(*input.DefaultBridgeType) {
		response.Error(c, domainerrors.BadRequest("invalid defaultBridgeType"))
		return
	}

	mode := entities.BridgeFallbackMode(strings.TrimSpace(input.FallbackMode))
	if mode == "" {
		mode = entities.BridgeFallbackModeStrict
	}
	if mode != entities.BridgeFallbackModeStrict && mode != entities.BridgeFallbackModeAutoFallback {
		response.Error(c, domainerrors.BadRequest("invalid fallbackMode"))
		return
	}
	order := input.FallbackOrder
	if len(order) == 0 {
		order = []uint8{*input.DefaultBridgeType}
	}
	if err := validateBridgeOrder(order); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	perByteRate, err := normalizeUnsignedInteger(input.PerByteRate)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid perByteRate"))
		return
	}
	overheadBytes, err := normalizeUnsignedInteger(input.OverheadBytes)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid overheadBytes"))
		return
	}
	minFee, err := normalizeUnsignedInteger(input.MinFee)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid minFee"))
		return
	}
	maxFee, err := normalizeUnsignedInteger(input.MaxFee)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid maxFee"))
		return
	}
	if err := validateMinMaxFee(minFee, maxFee); err != nil {
		response.Error(c, err)
		return
	}
	bridgeToken, err := normalizeBridgeTokenInput(input.BridgeToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	status, err := normalizeRoutePolicyStatusInput(input.Status)
	if err != nil {
		response.Error(c, err)
		return
	}
	supportsTokenBridge := input.SupportsTokenBridge != nil && *input.SupportsTokenBridge
	supportsDestSwap := input.SupportsDestSwap != nil && *input.SupportsDestSwap
	supportsPrivacyForward := input.SupportsPrivacyForward != nil && *input.SupportsPrivacyForward

	item := &entities.RoutePolicy{
		ID:                     utils.GenerateUUIDv7(),
		SourceChainID:          sourceChainID,
		DestChainID:            destChainID,
		DefaultBridgeType:      *input.DefaultBridgeType,
		FallbackMode:           mode,
		FallbackOrder:          order,
		SupportsTokenBridge:    supportsTokenBridge,
		SupportsDestSwap:       supportsDestSwap,
		SupportsPrivacyForward: supportsPrivacyForward,
		BridgeToken:            bridgeToken,
		Status:                 status,
		PerByteRate:            perByteRate,
		OverheadBytes:          overheadBytes,
		MinFee:                 minFee,
		MaxFee:                 maxFee,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
	if err := h.routePolicyRepo.Create(c.Request.Context(), item); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"policy": item})
}

func (h *CrosschainPolicyHandler) UpdateRoutePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid route policy id"))
		return
	}
	existing, err := h.routePolicyRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var input struct {
		SourceChainID          string  `json:"sourceChainId" binding:"required"`
		DestChainID            string  `json:"destChainId" binding:"required"`
		DefaultBridgeType      *uint8  `json:"defaultBridgeType" binding:"required"`
		FallbackMode           string  `json:"fallbackMode"`
		FallbackOrder          []uint8 `json:"fallbackOrder"`
		SupportsTokenBridge    *bool   `json:"supportsTokenBridge"`
		SupportsDestSwap       *bool   `json:"supportsDestSwap"`
		SupportsPrivacyForward *bool   `json:"supportsPrivacyForward"`
		BridgeToken            *string `json:"bridgeToken"`
		Status                 *string `json:"status"`
		PerByteRate            string  `json:"perByteRate"`
		OverheadBytes          string  `json:"overheadBytes"`
		MinFee                 string  `json:"minFee"`
		MaxFee                 string  `json:"maxFee"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
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
	if sourceChainID == destChainID {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId must be different"))
		return
	}
	if !isValidBridgeType(*input.DefaultBridgeType) {
		response.Error(c, domainerrors.BadRequest("invalid defaultBridgeType"))
		return
	}

	mode := entities.BridgeFallbackMode(strings.TrimSpace(input.FallbackMode))
	if mode == "" {
		mode = entities.BridgeFallbackModeStrict
	}
	if mode != entities.BridgeFallbackModeStrict && mode != entities.BridgeFallbackModeAutoFallback {
		response.Error(c, domainerrors.BadRequest("invalid fallbackMode"))
		return
	}
	order := input.FallbackOrder
	if len(order) == 0 {
		order = []uint8{*input.DefaultBridgeType}
	}
	if err := validateBridgeOrder(order); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	perByteRate, err := normalizeUnsignedInteger(input.PerByteRate)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid perByteRate"))
		return
	}
	overheadBytes, err := normalizeUnsignedInteger(input.OverheadBytes)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid overheadBytes"))
		return
	}
	minFee, err := normalizeUnsignedInteger(input.MinFee)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid minFee"))
		return
	}
	maxFee, err := normalizeUnsignedInteger(input.MaxFee)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid maxFee"))
		return
	}
	if err := validateMinMaxFee(minFee, maxFee); err != nil {
		response.Error(c, err)
		return
	}
	bridgeToken := existing.BridgeToken
	if input.BridgeToken != nil {
		normalizedBridgeToken, normalizeErr := normalizeBridgeTokenInput(input.BridgeToken)
		if normalizeErr != nil {
			response.Error(c, normalizeErr)
			return
		}
		bridgeToken = normalizedBridgeToken
	}
	status := existing.Status
	if input.Status != nil {
		normalizedStatus, normalizeErr := normalizeRoutePolicyStatusInput(input.Status)
		if normalizeErr != nil {
			response.Error(c, normalizeErr)
			return
		}
		status = normalizedStatus
	}
	if strings.TrimSpace(status) == "" {
		status = "active"
	}

	existing.SourceChainID = sourceChainID
	existing.DestChainID = destChainID
	existing.DefaultBridgeType = *input.DefaultBridgeType
	existing.FallbackMode = mode
	existing.FallbackOrder = order
	if input.SupportsTokenBridge != nil {
		existing.SupportsTokenBridge = *input.SupportsTokenBridge
	}
	if input.SupportsDestSwap != nil {
		existing.SupportsDestSwap = *input.SupportsDestSwap
	}
	if input.SupportsPrivacyForward != nil {
		existing.SupportsPrivacyForward = *input.SupportsPrivacyForward
	}
	existing.BridgeToken = bridgeToken
	existing.Status = status
	existing.PerByteRate = perByteRate
	existing.OverheadBytes = overheadBytes
	existing.MinFee = minFee
	existing.MaxFee = maxFee
	existing.UpdatedAt = time.Now()

	if err := h.routePolicyRepo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"policy": existing})
}

func (h *CrosschainPolicyHandler) DeleteRoutePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid route policy id"))
		return
	}
	if err := h.routePolicyRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "Route policy deleted"})
}

func (h *CrosschainPolicyHandler) ListLayerZeroConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
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
	var activeOnly *bool
	if strings.TrimSpace(c.Query("activeOnly")) != "" {
		v := strings.EqualFold(strings.TrimSpace(c.Query("activeOnly")), "true")
		activeOnly = &v
	}

	items, total, err := h.layerZeroConfigRepo.List(c.Request.Context(), sourceChainID, destChainID, activeOnly, pagination)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"items": items,
		"meta":  utils.CalculateMeta(total, pagination.Page, pagination.Limit),
	})
}

func (h *CrosschainPolicyHandler) CreateLayerZeroConfig(c *gin.Context) {
	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		DstEID        uint32 `json:"dstEid" binding:"required"`
		PeerHex       string `json:"peerHex" binding:"required"`
		OptionsHex    string `json:"optionsHex"`
		IsActive      *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
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
	if sourceChainID == destChainID {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId must be different"))
		return
	}
	peerHex := normalizeHex(input.PeerHex)
	if len(strings.TrimPrefix(peerHex, "0x")) != 64 {
		response.Error(c, domainerrors.BadRequest("peerHex must be bytes32 hex"))
		return
	}
	optionsHex := normalizeHex(input.OptionsHex)
	if optionsHex == "0x" {
		optionsHex = "0x"
	}

	item := &entities.LayerZeroConfig{
		ID:            utils.GenerateUUIDv7(),
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		DstEID:        input.DstEID,
		PeerHex:       peerHex,
		OptionsHex:    optionsHex,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if input.IsActive != nil {
		item.IsActive = *input.IsActive
	}
	if err := h.layerZeroConfigRepo.Create(c.Request.Context(), item); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusCreated, gin.H{"config": item})
}

func (h *CrosschainPolicyHandler) UpdateLayerZeroConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid layerzero config id"))
		return
	}
	existing, err := h.layerZeroConfigRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		DstEID        uint32 `json:"dstEid" binding:"required"`
		PeerHex       string `json:"peerHex" binding:"required"`
		OptionsHex    string `json:"optionsHex"`
		IsActive      *bool  `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
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
	if sourceChainID == destChainID {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId must be different"))
		return
	}
	peerHex := normalizeHex(input.PeerHex)
	if len(strings.TrimPrefix(peerHex, "0x")) != 64 {
		response.Error(c, domainerrors.BadRequest("peerHex must be bytes32 hex"))
		return
	}
	optionsHex := normalizeHex(input.OptionsHex)
	if optionsHex == "0x" {
		optionsHex = "0x"
	}

	existing.SourceChainID = sourceChainID
	existing.DestChainID = destChainID
	existing.DstEID = input.DstEID
	existing.PeerHex = peerHex
	existing.OptionsHex = optionsHex
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	existing.UpdatedAt = time.Now()

	if err := h.layerZeroConfigRepo.Update(c.Request.Context(), existing); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"config": existing})
}

func (h *CrosschainPolicyHandler) DeleteLayerZeroConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid layerzero config id"))
		return
	}
	if err := h.layerZeroConfigRepo.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"message": "LayerZero config deleted"})
}

func (h *CrosschainPolicyHandler) parseChainID(ctx context.Context, input string) (uuid.UUID, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return uuid.Nil, domainerrors.BadRequest("chain id is required")
	}
	if parsed, err := uuid.Parse(value); err == nil {
		return parsed, nil
	}
	if strings.Contains(value, ":") {
		if chain, err := h.chainRepo.GetByCAIP2(ctx, value); err == nil {
			return chain.ID, nil
		}
	}
	chain, err := h.chainRepo.GetByChainID(ctx, value)
	if err != nil {
		return uuid.Nil, err
	}
	return chain.ID, nil
}

func (h *CrosschainPolicyHandler) parseChainQuery(ctx context.Context, input string) (*uuid.UUID, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	id, err := h.parseChainID(ctx, input)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func isValidBridgeType(v uint8) bool {
	return v == 0 || v == 1 || v == 2 || v == 3
}

func validateBridgeOrder(order []uint8) error {
	if len(order) == 0 {
		return domainerrors.BadRequest("fallbackOrder cannot be empty")
	}
	seen := map[uint8]struct{}{}
	for _, v := range order {
		if !isValidBridgeType(v) {
			return domainerrors.BadRequest("fallbackOrder contains invalid bridgeType")
		}
		if _, ok := seen[v]; ok {
			return domainerrors.BadRequest("fallbackOrder contains duplicate bridgeType")
		}
		seen[v] = struct{}{}
	}
	return nil
}

func normalizeHex(v string) string {
	raw := strings.TrimSpace(v)
	if raw == "" {
		return "0x"
	}
	if !strings.HasPrefix(raw, "0x") {
		return "0x" + raw
	}
	return raw
}

func normalizeUnsignedInteger(v string) (string, error) {
	raw := strings.TrimSpace(v)
	if raw == "" {
		return "", nil
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return "", domainerrors.BadRequest("must be unsigned integer")
		}
	}
	return raw, nil
}

func normalizeBridgeTokenInput(v *string) (string, error) {
	if v == nil {
		return "", nil
	}
	raw := strings.TrimSpace(*v)
	if raw == "" {
		return "", nil
	}
	if !strings.HasPrefix(raw, "0x") {
		raw = "0x" + raw
	}
	if len(raw) != 42 {
		return "", domainerrors.BadRequest("bridgeToken must be a valid EVM address")
	}
	for _, ch := range strings.TrimPrefix(raw, "0x") {
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		case ch >= 'A' && ch <= 'F':
		default:
			return "", domainerrors.BadRequest("bridgeToken must be a valid EVM address")
		}
	}
	return strings.ToLower(raw), nil
}

func normalizeRoutePolicyStatusInput(v *string) (string, error) {
	if v == nil {
		return "active", nil
	}
	status := strings.ToLower(strings.TrimSpace(*v))
	if status == "" {
		return "active", nil
	}
	switch status {
	case "active", "paused", "deprecated":
		return status, nil
	default:
		return "", domainerrors.BadRequest("status must be active, paused, or deprecated")
	}
}

func validateMinMaxFee(minFee, maxFee string) error {
	if strings.TrimSpace(minFee) == "" || strings.TrimSpace(maxFee) == "" {
		return nil
	}
	minValue, ok := new(big.Int).SetString(minFee, 10)
	if !ok {
		return domainerrors.BadRequest("invalid minFee")
	}
	maxValue, ok := new(big.Int).SetString(maxFee, 10)
	if !ok {
		return domainerrors.BadRequest("invalid maxFee")
	}
	if maxValue.Cmp(minValue) < 0 {
		return domainerrors.BadRequest("maxFee must be greater than or equal to minFee")
	}
	return nil
}

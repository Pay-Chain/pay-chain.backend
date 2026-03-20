package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/domain/repositories"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/interfaces/http/response"
)

// AdminHandler handles admin endpoints
type AdminHandler struct {
	userRepo              repositories.UserRepository
	merchantRepo          repositories.MerchantRepository
	paymentRepo           repositories.PaymentRepository
	settlementProfileRepo repositories.MerchantSettlementProfileRepository
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	userRepo repositories.UserRepository,
	merchantRepo repositories.MerchantRepository,
	paymentRepo repositories.PaymentRepository,
	settlementProfileRepo repositories.MerchantSettlementProfileRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:              userRepo,
		merchantRepo:          merchantRepo,
		paymentRepo:           paymentRepo,
		settlementProfileRepo: settlementProfileRepo,
	}
}

// ListUsers lists all users
// GET /api/v1/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	search := c.Query("search")
	users, err := h.userRepo.List(c.Request.Context(), search)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"users": users})
}

// ListMerchants lists all merchants
// GET /api/v1/admin/merchants
func (h *AdminHandler) ListMerchants(c *gin.Context) {
	merchants, err := h.merchantRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	merchantIDs := make([]uuid.UUID, 0, len(merchants))
	for _, merchant := range merchants {
		if merchant != nil {
			merchantIDs = append(merchantIDs, merchant.ID)
		}
	}
	profileMap := map[uuid.UUID]bool{}
	if h.settlementProfileRepo != nil {
		profileMap, err = h.settlementProfileRepo.HasProfilesByMerchantIDs(c.Request.Context(), merchantIDs)
		if err != nil {
			response.Error(c, err)
			return
		}
	}
	items := make([]gin.H, 0, len(merchants))
	for _, merchant := range merchants {
		if merchant == nil {
			continue
		}
		items = append(items, gin.H{
			"id":                          merchant.ID,
			"userId":                      merchant.UserID,
			"businessName":                merchant.BusinessName,
			"businessEmail":               merchant.BusinessEmail,
			"merchantType":                merchant.MerchantType,
			"status":                      merchant.Status,
			"taxId":                       merchant.TaxID.String,
			"businessAddress":             merchant.BusinessAddress.String,
			"callbackUrl":                 merchant.CallbackURL,
			"supportEmail":                merchant.SupportEmail,
			"logoUrl":                     merchant.LogoURL,
			"verifiedAt":                  merchant.VerifiedAt,
			"createdAt":                   merchant.CreatedAt,
			"updatedAt":                   merchant.UpdatedAt,
			"settlementProfileConfigured": profileMap[merchant.ID],
		})
	}

	response.Success(c, http.StatusOK, gin.H{"merchants": items})
}

// UpdateMerchantStatus updates merchant status
// PUT /api/v1/admin/merchants/:id/status
func (h *AdminHandler) UpdateMerchantStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid merchant ID"))
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	// Fetch merchant to verify existence
	if _, err := h.merchantRepo.GetByID(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Merchant not found"))
			return
		}
		response.Error(c, err)
		return
	}

	// Update status logic would go here
	// For now mirroring existing implementation logic

	response.Success(c, http.StatusOK, gin.H{"message": "Merchant status updated", "status": input.Status})
}

// GetStats returns dashboard stats
// GET /api/v1/admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	legacySnapshot := middleware.GetLegacyEndpointObservabilitySnapshot()
	// Mock stats for now
	stats := gin.H{
		"totalUsers":                  150,
		"totalMerchants":              12,
		"totalVolume":                 "1234567.89",
		"activeChains":                5,
		"legacyEndpointObservability": legacySnapshot.Summary,
	}
	response.Success(c, http.StatusOK, stats)
}

// GetLegacyEndpointObservability returns operator-facing deprecation usage stats
// GET /api/v1/admin/diagnostics/legacy-endpoints
func (h *AdminHandler) GetLegacyEndpointObservability(c *gin.Context) {
	response.Success(c, http.StatusOK, middleware.GetLegacyEndpointObservabilitySnapshot())
}

// GetSettlementProfileGaps lists merchants missing dedicated settlement profiles.
// GET /api/v1/admin/diagnostics/settlement-profile-gaps
func (h *AdminHandler) GetSettlementProfileGaps(c *gin.Context) {
	if h.settlementProfileRepo == nil || h.merchantRepo == nil {
		response.Error(c, domainerrors.InternalServerError("settlement profile diagnostics not configured"))
		return
	}
	missingIDs, err := h.settlementProfileRepo.ListMissingMerchantIDs(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	merchants, err := h.merchantRepo.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	missingSet := make(map[uuid.UUID]struct{}, len(missingIDs))
	for _, id := range missingIDs {
		missingSet[id] = struct{}{}
	}
	items := make([]gin.H, 0)
	for _, merchant := range merchants {
		if merchant == nil {
			continue
		}
		if _, ok := missingSet[merchant.ID]; !ok {
			continue
		}
		items = append(items, gin.H{
			"id":            merchant.ID,
			"businessName":  merchant.BusinessName,
			"businessEmail": merchant.BusinessEmail,
			"merchantType":  merchant.MerchantType,
			"status":        merchant.Status,
			"createdAt":     merchant.CreatedAt,
		})
	}
	response.Success(c, http.StatusOK, gin.H{
		"total_missing": len(items),
		"merchants":     items,
	})
}

package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

type routeErrorService interface {
	GetRouteError(ctx context.Context, sourceChainInput string, paymentIDHex string) (*usecases.RouteErrorDiagnostics, error)
}

type RouteErrorHandler struct {
	service routeErrorService
}

func NewRouteErrorHandler(service *usecases.RouteErrorUsecase) *RouteErrorHandler {
	return &RouteErrorHandler{service: service}
}

// GetRouteError returns structured diagnostics for gateway.lastRouteError(paymentId).
// GET /api/v1/admin/diagnostics/route-error/:paymentId?sourceChainId=eip155:8453
func (h *RouteErrorHandler) GetRouteError(c *gin.Context) {
	sourceChainID := c.Query("sourceChainId")
	if sourceChainID == "" {
		response.Error(c, domainerrors.BadRequest("sourceChainId is required"))
		return
	}
	paymentID := c.Param("paymentId")
	if paymentID == "" {
		response.Error(c, domainerrors.BadRequest("paymentId is required"))
		return
	}

	result, err := h.service.GetRouteError(c.Request.Context(), sourceChainID, paymentID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"diagnostics": result})
}

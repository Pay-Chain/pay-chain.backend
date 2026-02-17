package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
	"pay-chain.backend/pkg/utils"
)

type crosschainConfigService interface {
	Overview(ctx context.Context, sourceChainInput, destChainInput string, pagination utils.PaginationParams) (*usecases.CrosschainOverview, error)
	RecheckRoute(ctx context.Context, sourceChainInput, destChainInput string) (*usecases.CrosschainRouteStatus, error)
	Preflight(ctx context.Context, sourceChainInput, destChainInput string) (*usecases.CrosschainPreflightResult, error)
	AutoFix(ctx context.Context, req *usecases.AutoFixRequest) (*usecases.AutoFixResult, error)
}

type CrosschainConfigHandler struct {
	usecase crosschainConfigService
}

func NewCrosschainConfigHandler(usecase *usecases.CrosschainConfigUsecase) *CrosschainConfigHandler {
	return &CrosschainConfigHandler{usecase: usecase}
}

func (h *CrosschainConfigHandler) Overview(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	pagination := utils.GetPaginationParams(page, limit)
	result, err := h.usecase.Overview(
		c.Request.Context(),
		strings.TrimSpace(c.Query("sourceChainId")),
		strings.TrimSpace(c.Query("destChainId")),
		pagination,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"items": result.Items, "meta": result.Meta})
}

func (h *CrosschainConfigHandler) Recheck(c *gin.Context) {
	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	result, err := h.usecase.RecheckRoute(c.Request.Context(), input.SourceChainID, input.DestChainID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"route": result})
}

func (h *CrosschainConfigHandler) Preflight(c *gin.Context) {
	sourceChainID := strings.TrimSpace(c.Query("sourceChainId"))
	destChainID := strings.TrimSpace(c.Query("destChainId"))
	if sourceChainID == "" || destChainID == "" {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId are required"))
		return
	}
	result, err := h.usecase.Preflight(c.Request.Context(), sourceChainID, destChainID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"preflight": result})
}

func (h *CrosschainConfigHandler) AutoFix(c *gin.Context) {
	var input usecases.AutoFixRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	result, err := h.usecase.AutoFix(c.Request.Context(), &input)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"result": result})
}

func (h *CrosschainConfigHandler) RecheckBulk(c *gin.Context) {
	var input struct {
		Routes []struct {
			SourceChainID string `json:"sourceChainId" binding:"required"`
			DestChainID   string `json:"destChainId" binding:"required"`
		} `json:"routes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	results := make([]usecases.CrosschainRouteStatus, 0, len(input.Routes))
	for _, route := range input.Routes {
		item, err := h.usecase.RecheckRoute(c.Request.Context(), route.SourceChainID, route.DestChainID)
		if err != nil {
			results = append(results, usecases.CrosschainRouteStatus{
				RouteKey:      route.SourceChainID + "->" + route.DestChainID,
				SourceChainID: route.SourceChainID,
				DestChainID:   route.DestChainID,
				OverallStatus: "ERROR",
				Issues: []usecases.ContractConfigCheckItem{
					{
						Code:    "RECHECK_FAILED",
						Status:  "ERROR",
						Message: err.Error(),
					},
				},
			})
			continue
		}
		results = append(results, *item)
	}
	response.Success(c, http.StatusOK, gin.H{"items": results})
}

func (h *CrosschainConfigHandler) AutoFixBulk(c *gin.Context) {
	var input struct {
		Routes []usecases.AutoFixRequest `json:"routes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	results := make([]*usecases.AutoFixResult, 0, len(input.Routes))
	for _, route := range input.Routes {
		routeCopy := route
		item, err := h.usecase.AutoFix(c.Request.Context(), &routeCopy)
		if err != nil {
			results = append(results, &usecases.AutoFixResult{
				SourceChainID: route.SourceChainID,
				DestChainID:   route.DestChainID,
				Steps: []usecases.AutoFixStep{
					{
						Step:    "autoFix",
						Status:  "FAILED",
						Message: err.Error(),
					},
				},
			})
			continue
		}
		results = append(results, item)
	}
	response.Success(c, http.StatusOK, gin.H{"items": results})
}

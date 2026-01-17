package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"pay-chain.backend/internal/domain/repositories"
)

// ChainHandler handles chain endpoints
type ChainHandler struct {
	chainRepo repositories.ChainRepository
}

// NewChainHandler creates a new chain handler
func NewChainHandler(chainRepo repositories.ChainRepository) *ChainHandler {
	return &ChainHandler{chainRepo: chainRepo}
}

// ListChains lists all active chains
// GET /api/v1/chains
func (h *ChainHandler) ListChains(c *gin.Context) {
	chains, err := h.chainRepo.GetActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list chains"})
		return
	}

	// Format response with CAIP-2 IDs
	type chainResponse struct {
		ID          int    `json:"id"`
		CAIP2       string `json:"caip2"`
		Name        string `json:"name"`
		ChainType   string `json:"chainType"`
		ExplorerURL string `json:"explorerUrl"`
		IsActive    bool   `json:"isActive"`
	}

	var response []chainResponse
	for _, chain := range chains {
		response = append(response, chainResponse{
			ID:          chain.ID,
			CAIP2:       chain.GetCAIP2ID(),
			Name:        chain.Name,
			ChainType:   string(chain.ChainType),
			ExplorerURL: chain.ExplorerURL,
			IsActive:    chain.IsActive,
		})
	}

	if response == nil {
		response = []chainResponse{}
	}

	c.JSON(http.StatusOK, gin.H{"chains": response})
}

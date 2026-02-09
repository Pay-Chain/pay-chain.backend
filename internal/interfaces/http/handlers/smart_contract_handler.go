package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/pkg/utils"
)

// SmartContractHandler handles smart contract endpoints
type SmartContractHandler struct {
	repo repositories.SmartContractRepository
}

// NewSmartContractHandler creates a new smart contract handler
func NewSmartContractHandler(repo repositories.SmartContractRepository) *SmartContractHandler {
	return &SmartContractHandler{repo: repo}
}

// CreateSmartContract creates a new smart contract record
// POST /api/v1/contracts
func (h *SmartContractHandler) CreateSmartContract(c *gin.Context) {
	var input entities.CreateSmartContractInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contract := &entities.SmartContract{
		Name:            input.Name,
		ChainID:         input.ChainID,
		ContractAddress: input.ContractAddress,
		ABI:             input.ABI,
	}

	if err := h.repo.Create(c.Request.Context(), contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contract"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"contract": contract,
	})
}

// GetSmartContract gets a smart contract by ID
// GET /api/v1/contracts/:id
func (h *SmartContractHandler) GetSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	contract, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contract"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contract": contract})
}

// ListSmartContracts lists all smart contracts
// GET /api/v1/contracts
func (h *SmartContractHandler) ListSmartContracts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chainID := c.Query("chainId")

	var contracts []*entities.SmartContract
	var totalCount int64
	var err error

	if chainID != "" {
		contracts, totalCount, err = h.repo.GetByChain(c.Request.Context(), chainID, pagination)
	} else {
		contracts, totalCount, err = h.repo.GetAll(c.Request.Context(), pagination)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list contracts"})
		return
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	c.JSON(http.StatusOK, gin.H{
		"items": contracts,
		"meta":  meta,
	})
}

// GetContractByChainAndAddress gets a contract by chain ID and address
// GET /api/v1/contracts/lookup?chainId=xxx&address=xxx
func (h *SmartContractHandler) GetContractByChainAndAddress(c *gin.Context) {
	chainID := c.Query("chainId")
	address := c.Query("address")

	if chainID == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chainId and address are required"})
		return
	}

	contract, err := h.repo.GetByChainAndAddress(c.Request.Context(), chainID, address)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contract"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contract": contract})
}

// DeleteSmartContract soft deletes a smart contract
// DELETE /api/v1/contracts/:id
func (h *SmartContractHandler) DeleteSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	if err := h.repo.SoftDelete(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contract"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contract deleted successfully"})
}

// UpdateSmartContract updates a smart contract
// PUT /api/v1/contracts/:id
func (h *SmartContractHandler) UpdateSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	var input entities.UpdateSmartContractInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contract, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}

	if input.Name != "" {
		contract.Name = input.Name
	}
	if input.Version != "" {
		contract.Version = input.Version
	}
	if input.ABI != nil {
		contract.ABI = input.ABI
	}
	if input.IsActive != nil {
		contract.IsActive = *input.IsActive
	}
	if input.Metadata != nil {
		// Conversion needed if entity and input types differ slightly
		// Assuming map[string]interface{} to null.JSON conversion logic here...
		// For now simple placeholder if needed, or if json.Marshal works
	}

	if err := h.repo.Update(c.Request.Context(), contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contract"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contract updated", "contract": contract})
}

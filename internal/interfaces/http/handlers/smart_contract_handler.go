package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/pkg/utils"
)

// SmartContractHandler handles smart contract endpoints
type SmartContractHandler struct {
	repo      repositories.SmartContractRepository
	chainRepo repositories.ChainRepository
}

// NewSmartContractHandler creates a new smart contract handler
func NewSmartContractHandler(repo repositories.SmartContractRepository, chainRepo repositories.ChainRepository) *SmartContractHandler {
	return &SmartContractHandler{
		repo:      repo,
		chainRepo: chainRepo,
	}
}

// CreateSmartContract creates a new smart contract record
// POST /api/v1/contracts
func (h *SmartContractHandler) CreateSmartContract(c *gin.Context) {
	var input entities.CreateSmartContractInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	chainUUID, err := uuid.Parse(input.ChainID)
	if err != nil {
		// Try lookup by legacy blockchain ID
		chain, err := h.chainRepo.GetByChainID(c.Request.Context(), input.ChainID)
		if err != nil {
			response.Error(c, domainerrors.BadRequest("Invalid chain ID"))
			return
		}
		chainUUID = chain.ID
	}

	contract := &entities.SmartContract{
		Name:            input.Name,
		ChainUUID:       chainUUID, // Changed to ChainUUID
		ContractAddress: input.ContractAddress,
		ABI:             input.ABI,
	}

	if err := h.repo.Create(c.Request.Context(), contract); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, gin.H{
		"contract": contract,
	})
}

// GetSmartContract gets a smart contract by ID
// GET /api/v1/contracts/:id
func (h *SmartContractHandler) GetSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid contract ID"))
		return
	}

	contract, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Contract not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"contract": contract})
}

// ListSmartContracts lists all smart contracts
// GET /api/v1/contracts
func (h *SmartContractHandler) ListSmartContracts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	pagination := utils.GetPaginationParams(page, limit)

	chainUUIDStr := c.Query("chainId")
	var chainUUID *uuid.UUID
	if chainUUIDStr != "" {
		id, err := uuid.Parse(chainUUIDStr)
		if err == nil {
			chainUUID = &id
		} else {
			// Try lookup by legacy blockchain ID
			chain, err := h.chainRepo.GetByChainID(c.Request.Context(), chainUUIDStr)
			if err == nil {
				chainUUID = &chain.ID
			}
		}
	}

	var contracts []*entities.SmartContract
	var totalCount int64
	var err error

	if chainUUID != nil {
		contracts, totalCount, err = h.repo.GetByChain(c.Request.Context(), *chainUUID, pagination)
	} else {
		contracts, totalCount, err = h.repo.GetAll(c.Request.Context(), pagination)
	}

	if err != nil {
		response.Error(c, err)
		return
	}

	meta := utils.CalculateMeta(totalCount, pagination.Page, pagination.Limit)
	response.Success(c, http.StatusOK, gin.H{
		"items": contracts,
		"meta":  meta,
	})
}

// GetContractByChainAndAddress gets a contract by chain ID and address
// GET /api/v1/contracts/lookup?chainId=xxx&address=xxx
func (h *SmartContractHandler) GetContractByChainAndAddress(c *gin.Context) {
	chainUUIDStr := c.Query("chainId")
	address := c.Query("address")

	if chainUUIDStr == "" || address == "" {
		response.Error(c, domainerrors.BadRequest("chainId and address are required"))
		return
	}

	chainUUID, err := uuid.Parse(chainUUIDStr)
	if err != nil {
		// Try lookup by legacy blockchain ID
		chain, err := h.chainRepo.GetByChainID(c.Request.Context(), chainUUIDStr)
		if err != nil {
			response.Error(c, domainerrors.BadRequest("Invalid chain ID"))
			return
		}
		chainUUID = chain.ID
	}

	contract, err := h.repo.GetByChainAndAddress(c.Request.Context(), chainUUID, address)
	if err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Contract not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"contract": contract})
}

// DeleteSmartContract soft deletes a smart contract
// DELETE /api/v1/contracts/:id
func (h *SmartContractHandler) DeleteSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid contract ID"))
		return
	}

	if err := h.repo.SoftDelete(c.Request.Context(), id); err != nil {
		if err == domainerrors.ErrNotFound {
			response.Error(c, domainerrors.NotFound("Contract not found"))
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Contract deleted successfully"})
}

// UpdateSmartContract updates a smart contract
// PUT /api/v1/contracts/:id
func (h *SmartContractHandler) UpdateSmartContract(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("Invalid contract ID"))
		return
	}

	var input entities.UpdateSmartContractInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	contract, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, domainerrors.NotFound("Contract not found"))
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
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Contract updated", "contract": contract})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
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
		Type:            input.Type,
		Version:         input.Version,
		ChainUUID:       chainUUID,
		ContractAddress: input.ContractAddress,
		DeployerAddress: null.NewString(input.DeployerAddress, input.DeployerAddress != ""),
		Token0Address:   null.NewString(input.Token0Address, input.Token0Address != ""),
		Token1Address:   null.NewString(input.Token1Address, input.Token1Address != ""),
		FeeTier:         null.NewInt(input.FeeTier, input.FeeTier != 0),
		HookAddress:     null.NewString(input.HookAddress, input.HookAddress != ""),
		StartBlock:      input.StartBlock,
		ABI:             input.ABI,
		IsActive:        true,
	}
	if input.IsActive != nil {
		contract.IsActive = *input.IsActive
	}

	if input.Metadata != nil {
		if raw, marshalErr := json.Marshal(input.Metadata); marshalErr == nil {
			contract.Metadata = null.JSONFrom(raw)
		}
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
	typeStr := strings.ToUpper(strings.TrimSpace(c.Query("type")))
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
	contracts, totalCount, err = h.repo.GetFiltered(
		c.Request.Context(),
		chainUUID,
		entities.SmartContractType(typeStr),
		pagination,
	)

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
	if input.Type != "" {
		contract.Type = input.Type
	}
	if input.Version != "" {
		contract.Version = input.Version
	}
	if input.ChainID != "" {
		chainUUID, parseErr := uuid.Parse(input.ChainID)
		if parseErr != nil {
			chain, chainErr := h.chainRepo.GetByChainID(c.Request.Context(), input.ChainID)
			if chainErr != nil {
				response.Error(c, domainerrors.BadRequest("Invalid chain ID"))
				return
			}
			chainUUID = chain.ID
		}
		contract.ChainUUID = chainUUID
	}
	if input.ContractAddress != "" {
		contract.ContractAddress = input.ContractAddress
	}
	if input.DeployerAddress != "" {
		contract.DeployerAddress = null.NewString(input.DeployerAddress, true)
	}
	if input.Token0Address != "" {
		contract.Token0Address = null.NewString(input.Token0Address, true)
	}
	if input.Token1Address != "" {
		contract.Token1Address = null.NewString(input.Token1Address, true)
	}
	if input.FeeTier != nil {
		contract.FeeTier = null.NewInt(*input.FeeTier, true)
	}
	if input.HookAddress != "" {
		contract.HookAddress = null.NewString(input.HookAddress, true)
	}
	if input.StartBlock != nil {
		contract.StartBlock = *input.StartBlock
	}
	if input.ABI != nil {
		contract.ABI = input.ABI
	}
	if input.IsActive != nil {
		contract.IsActive = *input.IsActive
	}
	if input.Metadata != nil {
		if raw, marshalErr := json.Marshal(input.Metadata); marshalErr == nil {
			contract.Metadata = null.JSONFrom(raw)
		}
	}

	if err := h.repo.Update(c.Request.Context(), contract); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Contract updated", "contract": contract})
}

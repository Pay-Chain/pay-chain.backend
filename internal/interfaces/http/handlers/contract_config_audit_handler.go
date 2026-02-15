package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

type ContractConfigAuditHandler struct {
	usecase *usecases.ContractConfigAuditUsecase
}

func NewContractConfigAuditHandler(usecase *usecases.ContractConfigAuditUsecase) *ContractConfigAuditHandler {
	return &ContractConfigAuditHandler{usecase: usecase}
}

func (h *ContractConfigAuditHandler) Check(c *gin.Context) {
	sourceChainID := strings.TrimSpace(c.Query("sourceChainId"))
	destChainID := strings.TrimSpace(c.Query("destChainId"))
	if sourceChainID == "" {
		response.Error(c, domainerrors.BadRequest("sourceChainId is required"))
		return
	}

	result, err := h.usecase.Check(c.Request.Context(), sourceChainID, destChainID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"result": result,
	})
}

func (h *ContractConfigAuditHandler) CheckByContract(c *gin.Context) {
	idRaw := strings.TrimSpace(c.Param("id"))
	contractID, err := uuid.Parse(idRaw)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid contract id"))
		return
	}

	result, err := h.usecase.CheckByContractID(c.Request.Context(), contractID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"result": result,
	})
}

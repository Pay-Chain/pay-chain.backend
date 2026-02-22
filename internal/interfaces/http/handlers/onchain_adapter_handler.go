package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/interfaces/http/response"
	"pay-chain.backend/internal/usecases"
)

type OnchainAdapterHandler struct {
	usecase onchainAdapterService
}
type onchainAdapterService interface {
	GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*usecases.OnchainAdapterStatus, error)
	RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error)
	SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error)
	SetHyperbridgeConfig(ctx context.Context, sourceChainInput, destChainInput string, stateMachineIDHex, destinationContractHex string) (string, []string, error)
	SetCCIPConfig(ctx context.Context, sourceChainInput, destChainInput string, chainSelector *uint64, destinationAdapterHex string) (string, []string, error)
	SetLayerZeroConfig(ctx context.Context, sourceChainInput, destChainInput string, dstEid *uint32, peerHex, optionsHex string) (string, []string, error)
	GenericInteract(ctx context.Context, sourceChainInput, contractAddress, method, abiStr string, args []interface{}) (interface{}, bool, error)
}

func NewOnchainAdapterHandler(usecase *usecases.OnchainAdapterUsecase) *OnchainAdapterHandler {
	return &OnchainAdapterHandler{usecase: usecase}
}

func (h *OnchainAdapterHandler) GetStatus(c *gin.Context) {
	sourceChainID := c.Query("sourceChainId")
	destChainID := c.Query("destChainId")
	if sourceChainID == "" || destChainID == "" {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId are required"))
		return
	}

	status, err := h.usecase.GetStatus(c.Request.Context(), sourceChainID, destChainID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": status})
}

func (h *OnchainAdapterHandler) RegisterAdapter(c *gin.Context) {
	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		BridgeType    *uint8 `json:"bridgeType" binding:"required"`
		Adapter       string `json:"adapterAddress" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	txHash, err := h.usecase.RegisterAdapter(c.Request.Context(), input.SourceChainID, input.DestChainID, *input.BridgeType, input.Adapter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"txHash":      txHash,
		"bridgeType":  strconv.Itoa(int(*input.BridgeType)),
		"destChainId": input.DestChainID,
	})
}

func (h *OnchainAdapterHandler) SetDefaultBridgeType(c *gin.Context) {
	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		BridgeType    *uint8 `json:"bridgeType" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	txHash, err := h.usecase.SetDefaultBridgeType(c.Request.Context(), input.SourceChainID, input.DestChainID, *input.BridgeType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"txHash":      txHash,
		"bridgeType":  strconv.Itoa(int(*input.BridgeType)),
		"destChainId": input.DestChainID,
	})
}

func (h *OnchainAdapterHandler) SetHyperbridgeConfig(c *gin.Context) {
	var input struct {
		SourceChainID          string `json:"sourceChainId" binding:"required"`
		DestChainID            string `json:"destChainId" binding:"required"`
		StateMachineIDHex      string `json:"stateMachineIdHex"`
		DestinationContractHex string `json:"destinationContractHex"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	adapter, txHashes, err := h.usecase.SetHyperbridgeConfig(
		c.Request.Context(),
		input.SourceChainID,
		input.DestChainID,
		input.StateMachineIDHex,
		input.DestinationContractHex,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"adapterAddress": adapter,
		"txHashes":       txHashes,
		"destChainId":    input.DestChainID,
	})
}

func (h *OnchainAdapterHandler) SetCCIPConfig(c *gin.Context) {
	var input struct {
		SourceChainID         string  `json:"sourceChainId" binding:"required"`
		DestChainID           string  `json:"destChainId" binding:"required"`
		ChainSelector         *uint64 `json:"chainSelector"`
		DestinationAdapterHex string  `json:"destinationAdapterHex"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	adapter, txHashes, err := h.usecase.SetCCIPConfig(
		c.Request.Context(),
		input.SourceChainID,
		input.DestChainID,
		input.ChainSelector,
		input.DestinationAdapterHex,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"adapterAddress": adapter,
		"txHashes":       txHashes,
		"destChainId":    input.DestChainID,
	})
}

func (h *OnchainAdapterHandler) SetLayerZeroConfig(c *gin.Context) {
	var input struct {
		SourceChainID string  `json:"sourceChainId" binding:"required"`
		DestChainID   string  `json:"destChainId" binding:"required"`
		DstEID        *uint32 `json:"dstEid"`
		PeerHex       string  `json:"peerHex"`
		OptionsHex    string  `json:"optionsHex"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	adapter, txHashes, err := h.usecase.SetLayerZeroConfig(
		c.Request.Context(),
		input.SourceChainID,
		input.DestChainID,
		input.DstEID,
		input.PeerHex,
		input.OptionsHex,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"adapterAddress": adapter,
		"txHashes":       txHashes,
		"destChainId":    input.DestChainID,
	})
}

func (h *OnchainAdapterHandler) Interact(c *gin.Context) {
	var input struct {
		SourceChainID   string        `json:"sourceChainId" binding:"required"`
		ContractAddress string        `json:"contractAddress" binding:"required"`
		Method          string        `json:"method" binding:"required"`
		ABI             string        `json:"abi" binding:"required"`
		Args            []interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	result, isWrite, err := h.usecase.GenericInteract(
		c.Request.Context(),
		input.SourceChainID,
		input.ContractAddress,
		input.Method,
		input.ABI,
		input.Args,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"result":  result,
		"isWrite": isWrite,
	})
}

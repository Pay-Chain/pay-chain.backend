package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/interfaces/http/response"
	"payment-kita.backend/internal/usecases"
)

type OnchainAdapterHandler struct {
	usecase onchainAdapterService
}
type onchainAdapterService interface {
	GetStatus(ctx context.Context, sourceChainInput, destChainInput string) (*usecases.OnchainAdapterStatus, error)
	RegisterAdapter(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8, adapterAddress string) (string, error)
	SetDefaultBridgeType(ctx context.Context, sourceChainInput, destChainInput string, bridgeType uint8) (string, error)
	SetHyperbridgeConfig(ctx context.Context, sourceChainInput, destChainInput string, stateMachineIDHex, destinationContractHex string) (string, []string, error)
	SetHyperbridgeTokenGatewayConfig(ctx context.Context, input usecases.HyperbridgeTokenGatewayConfigInput) (string, []string, error)
	SetCCIPConfig(ctx context.Context, input usecases.CCIPConfigInput) (string, []string, error)
	SetLayerZeroConfig(ctx context.Context, sourceChainInput, destChainInput string, dstEid *uint32, peerHex, optionsHex string) (string, []string, error)
	ConfigureLayerZeroE2E(ctx context.Context, input usecases.LayerZeroE2EConfigureInput) (*usecases.LayerZeroE2EConfigureResult, error)
	GetLayerZeroE2EStatus(ctx context.Context, input usecases.LayerZeroE2EStatusInput) (*usecases.LayerZeroE2EStatusResult, error)
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

func (h *OnchainAdapterHandler) SetHyperbridgeTokenGatewayConfig(c *gin.Context) {
	var input struct {
		SourceChainID            string  `json:"sourceChainId" binding:"required"`
		DestChainID              string  `json:"destChainId" binding:"required"`
		StateMachineIDHex        string  `json:"stateMachineIdHex"`
		SettlementExecutor       string  `json:"settlementExecutorAddress"`
		NativeCost               *string `json:"nativeCost"`
		RelayerFee               *string `json:"relayerFee"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	adapter, txHashes, err := h.usecase.SetHyperbridgeTokenGatewayConfig(
		c.Request.Context(),
		usecases.HyperbridgeTokenGatewayConfigInput{
			SourceChainInput:   input.SourceChainID,
			DestChainInput:     input.DestChainID,
			StateMachineIDHex:  input.StateMachineIDHex,
			SettlementExecutor: input.SettlementExecutor,
			NativeCost:         input.NativeCost,
			RelayerFee:         input.RelayerFee,
		},
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
		SourceChainID              string  `json:"sourceChainId" binding:"required"`
		DestChainID                string  `json:"destChainId" binding:"required"`
		ChainSelectorRaw           any     `json:"chainSelector"`
		DestinationAdapterHex      string  `json:"destinationAdapterHex"`
		DestinationGasLimitRaw     any     `json:"destinationGasLimit"`
		DestinationExtraArgsHex    string  `json:"destinationExtraArgsHex"`
		DestinationFeeTokenAddress string  `json:"destinationFeeTokenAddress"`
		DestinationReceiverAddress string  `json:"destinationReceiverAddress"`
		SourceChainSelectorRaw     any     `json:"sourceChainSelector"`
		TrustedSenderHex           string  `json:"trustedSenderHex"`
		AllowSourceChain           *bool   `json:"allowSourceChain"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}
	chainSelector, err := parseOptionalUint64(input.ChainSelectorRaw)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid chainSelector"))
		return
	}
	destinationGasLimit, err := parseOptionalUint64(input.DestinationGasLimitRaw)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid destinationGasLimit"))
		return
	}
	sourceChainSelector, err := parseOptionalUint64(input.SourceChainSelectorRaw)
	if err != nil {
		response.Error(c, domainerrors.BadRequest("invalid sourceChainSelector"))
		return
	}

	adapter, txHashes, err := h.usecase.SetCCIPConfig(c.Request.Context(), usecases.CCIPConfigInput{
		SourceChainInput:        input.SourceChainID,
		DestChainInput:          input.DestChainID,
		ChainSelector:           chainSelector,
		DestinationAdapterHex:   input.DestinationAdapterHex,
		DestinationGasLimit:     destinationGasLimit,
		DestinationExtraArgsHex: input.DestinationExtraArgsHex,
		DestinationFeeToken:     input.DestinationFeeTokenAddress,
		DestinationReceiver:     input.DestinationReceiverAddress,
		SourceChainSelector:     sourceChainSelector,
		TrustedSenderHex:        input.TrustedSenderHex,
		AllowSourceChain:        input.AllowSourceChain,
	})
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

func parseOptionalUint64(raw any) (*uint64, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, nil
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}
		out := uint64(n)
		return &out, nil
	case float64:
		// JSON numbers decoded into interface{} become float64 by default.
		if v < 0 || v > float64(^uint64(0)) || v != float64(uint64(v)) {
			return nil, strconv.ErrRange
		}
		out := uint64(v)
		return &out, nil
	case json.Number:
		n, err := strconv.ParseUint(strings.TrimSpace(v.String()), 10, 64)
		if err != nil {
			return nil, err
		}
		out := uint64(n)
		return &out, nil
	default:
		return nil, strconv.ErrSyntax
	}
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

func (h *OnchainAdapterHandler) ConfigureLayerZeroE2E(c *gin.Context) {
	var input struct {
		SourceChainID string `json:"sourceChainId" binding:"required"`
		DestChainID   string `json:"destChainId" binding:"required"`
		Source        struct {
			RegisterAdapterIfMissing bool   `json:"registerAdapterIfMissing"`
			SetDefaultBridgeType     bool   `json:"setDefaultBridgeType"`
			SenderAddress            string `json:"senderAddress"`
			DstEID                   uint32 `json:"dstEid"`
			DstPeerHex               string `json:"dstPeerHex"`
			OptionsHex               string `json:"optionsHex"`
			RegisterDelegate         bool   `json:"registerDelegate"`
			AuthorizeVaultSpender    bool   `json:"authorizeVaultSpender"`
		} `json:"source"`
		Destination struct {
			ReceiverAddress         string `json:"receiverAddress"`
			SrcEID                  uint32 `json:"srcEid"`
			SrcSenderHex            string `json:"srcSenderHex"`
			VaultAddress            string `json:"vaultAddress"`
			GatewayAddress          string `json:"gatewayAddress"`
			AuthorizeVaultSpender   bool   `json:"authorizeVaultSpender"`
			AuthorizeGatewayAdapter bool   `json:"authorizeGatewayAdapter"`
		} `json:"destination"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, domainerrors.BadRequest(err.Error()))
		return
	}

	result, err := h.usecase.ConfigureLayerZeroE2E(c.Request.Context(), usecases.LayerZeroE2EConfigureInput{
		SourceChainInput: input.SourceChainID,
		DestChainInput:   input.DestChainID,
		Source: usecases.LayerZeroConfigureSourceInput{
			RegisterAdapterIfMissing: input.Source.RegisterAdapterIfMissing,
			SetDefaultBridgeType:     input.Source.SetDefaultBridgeType,
			SenderAddress:            input.Source.SenderAddress,
			DstEID:                   input.Source.DstEID,
			DstPeerHex:               input.Source.DstPeerHex,
			OptionsHex:               input.Source.OptionsHex,
			RegisterDelegate:         input.Source.RegisterDelegate,
			AuthorizeVaultSpender:    input.Source.AuthorizeVaultSpender,
		},
		Destination: usecases.LayerZeroConfigureDestinationInput{
			ReceiverAddress:         input.Destination.ReceiverAddress,
			SrcEID:                  input.Destination.SrcEID,
			SrcSenderHex:            input.Destination.SrcSenderHex,
			VaultAddress:            input.Destination.VaultAddress,
			GatewayAddress:          input.Destination.GatewayAddress,
			AuthorizeVaultSpender:   input.Destination.AuthorizeVaultSpender,
			AuthorizeGatewayAdapter: input.Destination.AuthorizeGatewayAdapter,
		},
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"result": result})
}

func (h *OnchainAdapterHandler) GetLayerZeroE2EStatus(c *gin.Context) {
	sourceChainID := c.Query("sourceChainId")
	destChainID := c.Query("destChainId")
	if sourceChainID == "" || destChainID == "" {
		response.Error(c, domainerrors.BadRequest("sourceChainId and destChainId are required"))
		return
	}

	srcEID := uint32(0)
	if raw := c.Query("destinationSrcEid"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			response.Error(c, domainerrors.BadRequest("destinationSrcEid must be uint32"))
			return
		}
		srcEID = uint32(parsed)
	}

	status, err := h.usecase.GetLayerZeroE2EStatus(c.Request.Context(), usecases.LayerZeroE2EStatusInput{
		SourceChainInput:           sourceChainID,
		DestChainInput:             destChainID,
		DestinationReceiverAddress: c.Query("destinationReceiverAddress"),
		DestinationSrcEID:          srcEID,
		DestinationSrcSenderHex:    c.Query("destinationSrcSenderHex"),
		DestinationVaultAddress:    c.Query("destinationVaultAddress"),
		DestinationGatewayAddress:  c.Query("destinationGatewayAddress"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": status})
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

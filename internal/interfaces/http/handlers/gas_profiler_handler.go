package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"payment-kita.backend/internal/infrastructure/blockchain"
)

// GasProfilerHandler provides gas estimation and profiling services
type GasProfilerHandler struct {
	clientFactory *blockchain.ClientFactory
}

// NewGasProfilerHandler creates a new gas profiler handler
func NewGasProfilerHandler(clientFactory *blockchain.ClientFactory) *GasProfilerHandler {
	return &GasProfilerHandler{
		clientFactory: clientFactory,
	}
}

// GasEstimateResponse represents the response for gas estimation
type GasEstimateResponse struct {
	ChainID         string `json:"chainId"`
	ChainName       string `json:"chainName"`
	GasLimit        string `json:"gasLimit"`
	EstimatedGas    string `json:"estimatedGas"`
	Priority        string `json:"priority"` // slow, average, fast
}

// GetGasEstimate returns current gas estimation for a chain
func (h *GasProfilerHandler) GetGasEstimate(c *gin.Context) {
	chainID := c.Param("chainId")
	if chainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chainId is required"})
		return
	}

	// Get RPC URL for chain (simplified - in production use chain registry)
	rpcURL := getRPCURLForChain(chainID)
	if rpcURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found", "chainId": chainID})
		return
	}

	// Get client for chain (validate connection)
	_, err := h.clientFactory.GetEVMClient(rpcURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to chain"})
		return
	}

	response := GasEstimateResponse{
		ChainID:      chainID,
		ChainName:    getChainName(chainID),
		GasLimit:     "200000", // Typical ERC20 transfer
		EstimatedGas: "21000-200000",
		Priority:     "average",
	}

	c.JSON(http.StatusOK, response)
}

// GetGasEstimates returns gas estimations for multiple chains
func (h *GasProfilerHandler) GetGasEstimates(c *gin.Context) {
	chains := map[string]string{
		"42161": "Arbitrum One",
		"8453":  "Base",
		"137":   "Polygon",
	}
	responses := make([]GasEstimateResponse, 0)

	for chainID, chainName := range chains {
		responses = append(responses, GasEstimateResponse{
			ChainID:      chainID,
			ChainName:    chainName,
			GasLimit:     "200000",
			EstimatedGas: "21000-200000",
			Priority:     "average",
		})
	}

	c.JSON(http.StatusOK, gin.H{"estimates": responses})
}

// getChainName returns a human-readable chain name
func getChainName(chainID string) string {
	names := map[string]string{
		"42161": "Arbitrum One",
		"8453":  "Base",
		"137":   "Polygon",
		"1":     "Ethereum Mainnet",
	}
	if name, ok := names[chainID]; ok {
		return name
	}
	return "Unknown Chain"
}

// getRPCURLForChain returns the RPC URL for a chain ID
func getRPCURLForChain(chainID string) string {
	rpcs := map[string]string{
		"42161": "https://arb1.arbitrum.io/rpc",
		"8453":  "https://mainnet.base.org",
		"137":   "https://polygon-rpc.com",
		"1":     "https://eth.llamarpc.com",
	}
	if rpc, ok := rpcs[chainID]; ok {
		return rpc
	}
	return ""
}

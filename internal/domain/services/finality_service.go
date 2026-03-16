package services

import (
	"strings"
)

type FinalityService interface {
	GetRequiredConfirmations(networkID string) int
}

type finalityService struct{}

func NewFinalityService() FinalityService {
	return &finalityService{}
}

func (s *finalityService) GetRequiredConfirmations(networkID string) int {
	// Maturity Model based on Phase 8.1
	// networkID is CAIP-2 format (e.g., eip155:1)
	
	parts := strings.Split(networkID, ":")
	if len(parts) < 2 {
		return 12 // Default fallback
	}
	
	namespace := parts[0]
	reference := parts[1]
	
	switch namespace {
	case "eip155":
		switch reference {
		case "1": // Ethereum Mainnet
			return 12
		case "137": // Polygon
			return 12
		case "10", "42161": // Optimism, Arbitrum
			return 1
		case "8453": // Base
			return 12
		default:
			return 12
		}
	case "solana":
		return 32
	default:
		return 12
	}
}

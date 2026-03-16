package services

import (
	"context"
	"log"
)

type ComplianceService interface {
	ValidatePayer(ctx context.Context, walletAddress string) (int, string, error)
}

type complianceService struct {
	maxRiskThreshold int
}

func NewComplianceService(maxRiskThreshold int) ComplianceService {
	return &complianceService{
		maxRiskThreshold: maxRiskThreshold,
	}
}

func (s *complianceService) ValidatePayer(ctx context.Context, walletAddress string) (int, string, error) {
	// MOCK: In production, this would call TRM Labs or Chainalysis
	// For now, we simulate a low risk score for all wallets
	// unless it's a known test "bad" wallet
	
	score := 10 // Mock low risk
	level := "LOW"
	
	if walletAddress == "0xBAD0000000000000000000000000000000000000" {
		score = 95
		level = "HIGH"
	}
	
	log.Printf("[ComplianceService] Validating wallet %s: Score %d, Level %s", walletAddress, score, level)
	
	return score, level, nil
}

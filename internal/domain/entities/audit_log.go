package entities

import (
	"time"

	"github.com/google/uuid"
)

type ResolveAudit struct {
	ID            uuid.UUID `json:"id"`
	SessionID     uuid.UUID `json:"session_id"`
	WalletAddress string    `json:"wallet_address"`
	RiskScore     int       `json:"risk_score"`
	RiskLevel     string    `json:"risk_level"`
	UserAgent     string    `json:"user_agent"`
	IPAddress     string    `json:"ip_address"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

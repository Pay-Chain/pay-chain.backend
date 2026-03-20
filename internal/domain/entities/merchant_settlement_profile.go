package entities

import (
	"time"

	"github.com/google/uuid"
)

type MerchantSettlementProfile struct {
	ID                uuid.UUID  `json:"id"`
	MerchantID        uuid.UUID  `json:"merchant_id"`
	InvoiceCurrency   string     `json:"invoice_currency"`
	DestChain         string     `json:"dest_chain"`
	DestToken         string     `json:"dest_token"`
	DestWallet        string     `json:"dest_wallet"`
	BridgeTokenSymbol string     `json:"bridge_token_symbol"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"-"`
}

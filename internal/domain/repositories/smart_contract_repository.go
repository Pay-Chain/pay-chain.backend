package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

// SmartContractRepository defines smart contract data operations
type SmartContractRepository interface {
	Create(ctx context.Context, contract *entities.SmartContract) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error)
	GetByChainAndAddress(ctx context.Context, chainID, address string) (*entities.SmartContract, error)
	GetByChain(ctx context.Context, chainID string) ([]*entities.SmartContract, error)
	GetAll(ctx context.Context) ([]*entities.SmartContract, error)
	Update(ctx context.Context, contract *entities.SmartContract) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

package repositories

import (
	"context"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/pkg/utils"
)

// SmartContractRepository defines smart contract data operations
type SmartContractRepository interface {
	Create(ctx context.Context, contract *entities.SmartContract) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error)
	GetByChainAndAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.SmartContract, error)
	// GetActiveContract returns the currently active contract of a specific type on a chain
	GetActiveContract(ctx context.Context, chainID uuid.UUID, contractType entities.SmartContractType) (*entities.SmartContract, error)
	GetFiltered(ctx context.Context, chainID *uuid.UUID, contractType entities.SmartContractType, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error)
	GetByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error)
	GetAll(ctx context.Context, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error)
	Update(ctx context.Context, contract *entities.SmartContract) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

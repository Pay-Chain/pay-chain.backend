package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// SmartContractRepository implements smart contract data operations
type SmartContractRepository struct {
	db *sql.DB
}

// NewSmartContractRepository creates a new smart contract repository
func NewSmartContractRepository(db *sql.DB) *SmartContractRepository {
	return &SmartContractRepository{db: db}
}

// Create creates a new smart contract record
func (r *SmartContractRepository) Create(ctx context.Context, contract *entities.SmartContract) error {
	query := `
		INSERT INTO smart_contracts (id, name, chain_id, contract_address, abi, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	contract.ID = uuid.New()
	contract.CreatedAt = time.Now()
	contract.UpdatedAt = time.Now()

	abiJSON, err := json.Marshal(contract.ABI)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		contract.ID,
		contract.Name,
		contract.ChainID,
		contract.ContractAddress,
		abiJSON,
		contract.CreatedAt,
		contract.UpdatedAt,
	)

	return err
}

// GetByID gets a smart contract by ID
func (r *SmartContractRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	query := `
		SELECT id, name, chain_id, contract_address, abi, created_at, updated_at
		FROM smart_contracts
		WHERE id = $1 AND deleted_at IS NULL
	`

	var abiBytes []byte
	contract := &entities.SmartContract{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&contract.ID,
		&contract.Name,
		&contract.ChainID,
		&contract.ContractAddress,
		&abiBytes,
		&contract.CreatedAt,
		&contract.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(abiBytes, &contract.ABI); err != nil {
		return nil, err
	}

	return contract, nil
}

// GetByChainAndAddress gets a smart contract by chain ID and address
func (r *SmartContractRepository) GetByChainAndAddress(ctx context.Context, chainID, address string) (*entities.SmartContract, error) {
	query := `
		SELECT id, name, chain_id, contract_address, abi, created_at, updated_at
		FROM smart_contracts
		WHERE chain_id = $1 AND contract_address = $2 AND deleted_at IS NULL
	`

	var abiBytes []byte
	contract := &entities.SmartContract{}
	err := r.db.QueryRowContext(ctx, query, chainID, address).Scan(
		&contract.ID,
		&contract.Name,
		&contract.ChainID,
		&contract.ContractAddress,
		&abiBytes,
		&contract.CreatedAt,
		&contract.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(abiBytes, &contract.ABI); err != nil {
		return nil, err
	}

	return contract, nil
}

// GetByChain gets all smart contracts for a chain
func (r *SmartContractRepository) GetByChain(ctx context.Context, chainID string) ([]*entities.SmartContract, error) {
	query := `
		SELECT id, name, chain_id, contract_address, abi, created_at, updated_at
		FROM smart_contracts
		WHERE chain_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []*entities.SmartContract
	for rows.Next() {
		var abiBytes []byte
		contract := &entities.SmartContract{}
		if err := rows.Scan(
			&contract.ID,
			&contract.Name,
			&contract.ChainID,
			&contract.ContractAddress,
			&abiBytes,
			&contract.CreatedAt,
			&contract.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(abiBytes, &contract.ABI); err != nil {
			return nil, err
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// GetAll gets all smart contracts
func (r *SmartContractRepository) GetAll(ctx context.Context) ([]*entities.SmartContract, error) {
	query := `
		SELECT id, name, chain_id, contract_address, abi, created_at, updated_at
		FROM smart_contracts
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []*entities.SmartContract
	for rows.Next() {
		var abiBytes []byte
		contract := &entities.SmartContract{}
		if err := rows.Scan(
			&contract.ID,
			&contract.Name,
			&contract.ChainID,
			&contract.ContractAddress,
			&abiBytes,
			&contract.CreatedAt,
			&contract.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(abiBytes, &contract.ABI); err != nil {
			return nil, err
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// Update updates a smart contract
func (r *SmartContractRepository) Update(ctx context.Context, contract *entities.SmartContract) error {
	query := `
		UPDATE smart_contracts
		SET name = $2, abi = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	contract.UpdatedAt = time.Now()

	abiJSON, err := json.Marshal(contract.ABI)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, query,
		contract.ID,
		contract.Name,
		abiJSON,
		contract.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

// SoftDelete soft deletes a smart contract
func (r *SmartContractRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE smart_contracts
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domainerrors.ErrNotFound
	}

	return nil
}

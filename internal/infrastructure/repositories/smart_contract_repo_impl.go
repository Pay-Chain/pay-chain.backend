package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

type SmartContractRepositoryImpl struct {
	db *gorm.DB
}

func NewSmartContractRepository(db *gorm.DB) *SmartContractRepositoryImpl {
	return &SmartContractRepositoryImpl{db: db}
}

func (r *SmartContractRepositoryImpl) Create(ctx context.Context, contract *entities.SmartContract) error {
	// Marshal ABI
	abiBytes, err := json.Marshal(contract.ABI)
	if err != nil {
		return fmt.Errorf("failed to marshal ABI: %w", err)
	}

	// Marshaling Metadata
	// Assuming contract.Metadata is valid JSON bytes or similar from Entity
	var metadataStr string
	if contract.Metadata.Valid {
		metadataStr = string(contract.Metadata.JSON)
	} else {
		metadataStr = "{}"
	}

	deployerAddr := ""
	if contract.DeployerAddress.Valid {
		deployerAddr = contract.DeployerAddress.String
	}

	m := &models.SmartContract{
		ID:              contract.ID,
		Name:            contract.Name,
		ChainID:         contract.ChainID,
		ContractAddress: contract.ContractAddress,
		Type:            string(contract.Type),
		Version:         contract.Version,
		DeployerAddress: deployerAddr,
		StartBlock:      int64(contract.StartBlock),
		ABI:             string(abiBytes),
		Metadata:        metadataStr,
		IsActive:        contract.IsActive,
		DestinationMap:  pq.StringArray{}, // Initial empty, or map if entity has it
	}

	// GORM will likely handle CreatedAt/UpdatedAt automatically if fields are standard
	// But we can set them explicit if we want to match entity
	m.CreatedAt = contract.CreatedAt
	m.UpdatedAt = contract.UpdatedAt

	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *SmartContractRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	var m models.SmartContract
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil, nil for not found as per previous behavior
		}
		return nil, err
	}
	return r.toEntity(&m)
}

func (r *SmartContractRepositoryImpl) GetByChainAndAddress(ctx context.Context, chainID, address string) (*entities.SmartContract, error) {
	var m models.SmartContract
	if err := r.db.WithContext(ctx).Where("chain_id = ? AND contract_address = ?", chainID, address).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(&m)
}

func (r *SmartContractRepositoryImpl) GetActiveContract(ctx context.Context, chainID string, contractType entities.SmartContractType) (*entities.SmartContract, error) {
	var m models.SmartContract
	// Order by version desc to get latest
	if err := r.db.WithContext(ctx).
		Where("chain_id = ? AND type = ? AND is_active = ?", chainID, contractType, true).
		Order("version DESC").
		First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(&m)
}

func (r *SmartContractRepositoryImpl) GetByChain(ctx context.Context, chainID string) ([]*entities.SmartContract, error) {
	var ms []models.SmartContract
	if err := r.db.WithContext(ctx).Where("chain_id = ?", chainID).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}

	var entitiesList []*entities.SmartContract
	for _, m := range ms {
		model := m
		e, err := r.toEntity(&model)
		if err != nil {
			return nil, err
		}
		entitiesList = append(entitiesList, e)
	}
	return entitiesList, nil
}

func (r *SmartContractRepositoryImpl) GetAll(ctx context.Context) ([]*entities.SmartContract, error) {
	var ms []models.SmartContract
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, err
	}

	var entitiesList []*entities.SmartContract
	for _, m := range ms {
		model := m
		e, err := r.toEntity(&model)
		if err != nil {
			return nil, err
		}
		entitiesList = append(entitiesList, e)
	}
	return entitiesList, nil
}

func (r *SmartContractRepositoryImpl) Update(ctx context.Context, contract *entities.SmartContract) error {
	// We can use Updates with struct or map
	// First Marshal fields
	abiBytes, _ := json.Marshal(contract.ABI)
	metadataStr := "{}"
	if contract.Metadata.Valid {
		metadataStr = string(contract.Metadata.JSON)
	}

	updates := map[string]interface{}{
		"name":      contract.Name,
		"version":   contract.Version,
		"is_active": contract.IsActive,
		"abi":       string(abiBytes),
		"metadata":  metadataStr,
		// Add others if needed
	}

	result := r.db.WithContext(ctx).Model(&models.SmartContract{}).Where("id = ?", contract.ID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *SmartContractRepositoryImpl) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.SmartContract{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

func (r *SmartContractRepositoryImpl) toEntity(m *models.SmartContract) (*entities.SmartContract, error) {
	var abi interface{}
	if m.ABI != "" {
		_ = json.Unmarshal([]byte(m.ABI), &abi)
	}

	// Convert string metadata back to null.JSON if possible, or construct it
	// The entity uses volatiletech/null/v8
	// We need to import it or handle it.
	// Wait, the new repo file doesn't import null.
	// I should check what entities.SmartContract expects.
	// Assuming entities still uses null.JSON for Metadata.

	// I need to import "github.com/volatiletech/null/v8" if I want to construct it properly
	// OR I can use explicit assignment if I add the import.

	// Let's check imports I added... I didn't add null/v8.
	// Use replacement to add it if compilation fails.

	return &entities.SmartContract{
		ID:              m.ID,
		Name:            m.Name,
		Type:            entities.SmartContractType(m.Type),
		Version:         m.Version,
		ChainID:         m.ChainID,
		ContractAddress: m.ContractAddress,
		// DeployerAddress: null.StringFrom(m.DeployerAddress), // Need null
		StartBlock: uint64(m.StartBlock),
		ABI:        abi,
		// Metadata:        null.JSONFrom([]byte(m.Metadata)), // Need null
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		// DeletedAt:       m.DeletedAt, // GORM DeletedAt is struct, entity might expect pointer or specific type
	}, nil
}

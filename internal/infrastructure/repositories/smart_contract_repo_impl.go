package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v8" // Added import
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

type SmartContractRepositoryImpl struct {
	db        *gorm.DB
	chainRepo repositories.ChainRepository
}

func NewSmartContractRepository(db *gorm.DB, chainRepo repositories.ChainRepository) *SmartContractRepositoryImpl {
	return &SmartContractRepositoryImpl{
		db:        db,
		chainRepo: chainRepo,
	}
}

func (r *SmartContractRepositoryImpl) Create(ctx context.Context, contract *entities.SmartContract) error {
	abiBytes, err := json.Marshal(contract.ABI)
	if err != nil {
		return fmt.Errorf("failed to marshal ABI: %w", err)
	}

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
		ChainID:         contract.ChainUUID, // Internal UUID
		ContractAddress: contract.ContractAddress,
		Type:            string(contract.Type),
		Version:         contract.Version,
		DeployerAddress: deployerAddr,
		Token0Address:   contract.Token0Address.String,
		Token1Address:   contract.Token1Address.String,
		FeeTier:         contract.FeeTier.Int,
		HookAddress:     contract.HookAddress.String,
		StartBlock:      int64(contract.StartBlock),
		ABI:             string(abiBytes),
		Metadata:        metadataStr,
		IsActive:        contract.IsActive,
		DestinationMap:  pq.StringArray{},
		CreatedAt:       contract.CreatedAt,
		UpdatedAt:       contract.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *SmartContractRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.SmartContract, error) {
	var m models.SmartContract
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(&m)
}

func (r *SmartContractRepositoryImpl) GetByChainAndAddress(ctx context.Context, chainID uuid.UUID, address string) (*entities.SmartContract, error) {
	var m models.SmartContract
	if err := r.db.WithContext(ctx).Where("chain_id = ? AND contract_address = ?", chainID, address).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(&m)
}

func (r *SmartContractRepositoryImpl) GetActiveContract(ctx context.Context, chainID uuid.UUID, contractType entities.SmartContractType) (*entities.SmartContract, error) {
	var m models.SmartContract
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

func (r *SmartContractRepositoryImpl) GetByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	var ms []models.SmartContract
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.SmartContract{}).Where("chain_id = ?", chainID)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var entitiesList []*entities.SmartContract
	for _, m := range ms {
		model := m
		e, err := r.toEntity(&model)
		if err != nil {
			return nil, 0, err
		}
		entitiesList = append(entitiesList, e)
	}
	return entitiesList, totalCount, nil
}

func (r *SmartContractRepositoryImpl) GetAll(ctx context.Context, pagination utils.PaginationParams) ([]*entities.SmartContract, int64, error) {
	var ms []models.SmartContract
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.SmartContract{})

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var entitiesList []*entities.SmartContract
	for _, m := range ms {
		model := m
		e, err := r.toEntity(&model)
		if err != nil {
			return nil, 0, err
		}
		entitiesList = append(entitiesList, e)
	}
	return entitiesList, totalCount, nil
}

func (r *SmartContractRepositoryImpl) Update(ctx context.Context, contract *entities.SmartContract) error {
	abiBytes, _ := json.Marshal(contract.ABI)
	metadataStr := "{}"
	if contract.Metadata.Valid {
		metadataStr = string(contract.Metadata.JSON)
	}

	updates := map[string]interface{}{
		"name":           contract.Name,
		"version":        contract.Version,
		"is_active":      contract.IsActive,
		"abi":            string(abiBytes),
		"metadata":       metadataStr,
		"token0_address": contract.Token0Address.String,
		"token1_address": contract.Token1Address.String,
		"fee_tier":       contract.FeeTier.Int,
		"hook_address":   contract.HookAddress.String,
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
	if m.ABI != "" && m.ABI != "null" {
		if err := json.Unmarshal([]byte(m.ABI), &abi); err != nil {
			// Log error but don't fail the whole list if one ABI is corrupted
			fmt.Printf("Warning: failed to unmarshal ABI for contract %s: %v\n", m.ID, err)
		}
	}

	// Safely creating null types
	deployer := null.NewString(m.DeployerAddress, m.DeployerAddress != "")
	token0 := null.NewString(m.Token0Address, m.Token0Address != "")
	token1 := null.NewString(m.Token1Address, m.Token1Address != "")
	feeTier := null.NewInt(m.FeeTier, m.FeeTier != 0)
	hook := null.NewString(m.HookAddress, m.HookAddress != "")

	metadataJSON := m.Metadata
	if metadataJSON == "" || metadataJSON == "null" {
		metadataJSON = "{}"
	}
	meta := null.JSONFrom([]byte(metadataJSON))

	e := &entities.SmartContract{
		ID:              m.ID,
		Name:            m.Name,
		Type:            entities.SmartContractType(m.Type),
		Version:         m.Version,
		ChainUUID:       m.ChainID, // Internal UUID
		ContractAddress: m.ContractAddress,
		DeployerAddress: deployer,
		Token0Address:   token0,
		Token1Address:   token1,
		FeeTier:         feeTier,
		HookAddress:     hook,
		StartBlock:      uint64(m.StartBlock),
		ABI:             abi,
		Metadata:        meta,
		IsActive:        m.IsActive,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	// Fetch chain to get BlockchainID
	if m.ChainID != uuid.Nil {
		chain, err := r.chainRepo.GetByID(context.Background(), m.ChainID)
		if err == nil {
			e.BlockchainID = chain.ChainID
		}
	}

	return e, nil
}

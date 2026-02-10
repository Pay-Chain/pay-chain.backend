package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

// chainRepo implements repositories.ChainRepository
type chainRepo struct {
	db *gorm.DB
}

// NewChainRepository creates a new chain repository
func NewChainRepository(db *gorm.DB) repositories.ChainRepository {
	return &chainRepo{db: db}
}

// GetByID gets a chain by ID
func (r *chainRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Chain, error) {
	var m models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByChainID gets a chain by external ChainID (NetworkID)
func (r *chainRepo) GetByChainID(ctx context.Context, chainID string) (*entities.Chain, error) {
	var m models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Where("chain_id = ?", chainID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByCAIP2 gets a chain by CAIP-2 ID (namespace:reference)
func (r *chainRepo) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	parts := strings.Split(caip2, ":")
	if len(parts) != 2 {
		return nil, domainerrors.ErrInvalidInput
	}
	namespace := parts[0]
	networkID := parts[1]

	var m models.Chain
	// Map NetworkID to chain_id column (Chain Model NetworkID field)
	if err := r.db.WithContext(ctx).Preload("RPCs").Where("namespace = ? AND chain_id = ?", namespace, networkID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetAll gets all chains
func (r *chainRepo) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	var ms []models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Find(&ms).Error; err != nil {
		return nil, err
	}

	var chains []*entities.Chain
	for _, m := range ms {
		model := m
		chains = append(chains, r.toEntity(&model))
	}
	return chains, nil
}

// GetAllRPCs gets all RPCs
func (r *chainRepo) GetAllRPCs(ctx context.Context, chainID *uuid.UUID, isActive *bool, search *string, pagination utils.PaginationParams) ([]*entities.ChainRPC, int64, error) {
	var ms []models.ChainRPC
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.ChainRPC{})

	if chainID != nil {
		query = query.Where("chain_id = ?", *chainID)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if search != nil && *search != "" {
		term := "%" + *search + "%"
		query = query.Where("url ILIKE ?", term)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Preload Chain for response mapping
	query = query.Preload("Chain", func(db *gorm.DB) *gorm.DB {
		return db.Unscoped()
	}).Order("chain_id, priority DESC")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var rpcs []*entities.ChainRPC
	for _, m := range ms {
		model := m
		rpcs = append(rpcs, r.toRpcEntity(&model))
	}
	return rpcs, totalCount, nil
}

// GetActive gets all active chains
func (r *chainRepo) GetActive(ctx context.Context, pagination utils.PaginationParams) ([]*entities.Chain, int64, error) {
	var ms []models.Chain
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.Chain{}).Where("is_active = ?", true)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("RPCs").Order("name")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var chains []*entities.Chain
	for _, m := range ms {
		model := m
		chains = append(chains, r.toEntity(&model))
	}
	return chains, totalCount, nil
}

// Create creates a new chain
func (r *chainRepo) Create(ctx context.Context, chain *entities.Chain) error {
	m := &models.Chain{
		ID:        chain.ID,
		NetworkID: chain.ChainID,
		// Namespace:      r.getNamespace(chain.Type), // Deprecated/Derived
		Name:           chain.Name,
		ChainType:      string(chain.Type),
		RPCURL:         chain.RPCURL,
		ExplorerURL:    chain.ExplorerURL,
		Symbol:         chain.CurrencySymbol,
		LogoURL:        chain.ImageURL,
		IsActive:       chain.IsActive,
		StateMachineID: "", // Entity doesn't have this field
		CreatedAt:      chain.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

// Update updates a chain
func (r *chainRepo) Update(ctx context.Context, chain *entities.Chain) error {
	updates := map[string]interface{}{
		"chain_id": chain.ChainID,
		"name":     chain.Name,
		// "namespace":       r.getNamespace(chain.Type), // Removed
		"type":            string(chain.Type),
		"rpc_url":         chain.RPCURL,
		"explorer_url":    chain.ExplorerURL,
		"currency_symbol": chain.CurrencySymbol,
		"image_url":       chain.ImageURL,
		"is_active":       chain.IsActive,
		// "state_machine_id": chain.StateMachineID, // Removed
	}

	result := r.db.WithContext(ctx).Model(&models.Chain{}).Where("id = ?", chain.ID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

// Delete deletes a chain
func (r *chainRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Chain{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrNotFound
	}
	return nil
}

// toEntity converts GORM model to Domain Entity
func (r *chainRepo) toEntity(m *models.Chain) *entities.Chain {
	// Logic to pick main RPC URL if legacy is empty?
	// The entity has RPCURL string.
	// Model has RPCURL string.
	// We can just usage m.RPCURL.

	// Preload RPCs are in m.RPCs
	var rpcs []entities.ChainRPC
	for _, rpc := range m.RPCs {
		rpcs = append(rpcs, *r.toRpcEntity(&rpc))
	}

	return &entities.Chain{
		ID:             m.ID,
		ChainID:        m.NetworkID,
		Name:           m.Name,
		Type:           entities.ChainType(strings.ToUpper(m.ChainType)),
		RPCURL:         m.RPCURL,
		ExplorerURL:    m.ExplorerURL,
		CurrencySymbol: m.Symbol,
		ImageURL:       m.LogoURL,
		IsActive:       m.IsActive,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		// DeletedAt:      &m.DeletedAt.Time, // GORM DeletedAt is struct?
		RPCs: rpcs,
	}
}

func (r *chainRepo) getNamespace(chainType entities.ChainType) string {
	switch chainType {
	case entities.ChainTypeEVM:
		return "eip155"
	case entities.ChainTypeSVM:
		return "solana"
	case entities.ChainTypeSubstrate:
		return "substrate"
	default:
		return "unknown"
	}
}

// toRpcEntity converts GORM RPC model to Entity
func (r *chainRepo) toRpcEntity(m *models.ChainRPC) *entities.ChainRPC {
	e := &entities.ChainRPC{
		ID:          m.ID,
		ChainID:     m.ChainID,
		URL:         m.URL,
		Priority:    m.Priority,
		IsActive:    m.IsActive,
		LastErrorAt: m.LastErrorAt,
		ErrorCount:  m.ErrorCount,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	// Map Chain if preloaded
	if m.Chain.ID != uuid.Nil {
		e.Chain = r.toEntity(&m.Chain)
	}

	return e
}

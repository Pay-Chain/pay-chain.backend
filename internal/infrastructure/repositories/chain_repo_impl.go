package repositories

import (
	"context"

	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
)

// ChainRepository implements chain data operations
type ChainRepository struct {
	db *gorm.DB
}

// NewChainRepository creates a new chain repository
func NewChainRepository(db *gorm.DB) *ChainRepository {
	return &ChainRepository{db: db}
}

// GetByID gets a chain by ID
func (r *ChainRepository) GetByID(ctx context.Context, id int) (*entities.Chain, error) {
	var chainModel models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Where("id = ?", id).First(&chainModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	return r.toEntity(&chainModel), nil
}

// GetByCAIP2 gets a chain by CAIP-2 identifier
func (r *ChainRepository) GetByCAIP2(ctx context.Context, caip2 string) (*entities.Chain, error) {
	// Parse CAIP-2: namespace:chainId
	var namespace string
	var chainID int
	_, err := parseCAIP2(caip2, &namespace, &chainID)
	if err != nil {
		return nil, domainerrors.ErrBadRequest
	}

	var chainModel models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Where("namespace = ? AND id = ?", namespace, chainID).First(&chainModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}

	return r.toEntity(&chainModel), nil
}

// GetAll gets all active chains
func (r *ChainRepository) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	var chainModels []models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Order("name").Find(&chainModels).Error; err != nil {
		return nil, err
	}

	var entitiesList []*entities.Chain
	for _, m := range chainModels {
		// Create a local copy to safe pointer referencing
		model := m
		entitiesList = append(entitiesList, r.toEntity(&model))
	}

	return entitiesList, nil
}

// GetActive gets only active chains
func (r *ChainRepository) GetActive(ctx context.Context) ([]*entities.Chain, error) {
	var chainModels []models.Chain
	if err := r.db.WithContext(ctx).Preload("RPCs").Where("is_active = ?", true).Order("name").Find(&chainModels).Error; err != nil {
		return nil, err
	}

	var entitiesList []*entities.Chain
	for _, m := range chainModels {
		model := m
		entitiesList = append(entitiesList, r.toEntity(&model))
	}

	return entitiesList, nil
}

// toEntity converts GORM model to Domain Entity
func (r *ChainRepository) toEntity(m *models.Chain) *entities.Chain {
	rpcURLs := make([]string, 0)
	for _, rpc := range m.RPCs {
		if rpc.IsActive {
			rpcURLs = append(rpcURLs, rpc.URL)
		}
	}

	// Fallback/Legacy
	legacyURL := m.RPCURL
	if len(rpcURLs) > 0 {
		legacyURL = rpcURLs[0]
	}

	return &entities.Chain{
		ID:             m.ID,
		Namespace:      m.Namespace,
		Name:           m.Name,
		ChainType:      entities.ChainType(m.ChainType),
		RPCURL:         legacyURL,
		ExplorerURL:    m.ExplorerURL,
		IsActive:       m.IsActive,
		RPCURLs:        rpcURLs,
		CreatedAt:      m.CreatedAt,
		StateMachineID: m.StateMachineID,
	}
}

// parseCAIP2 parses a CAIP-2 identifier into namespace and chainId
func parseCAIP2(caip2 string, namespace *string, chainID *int) (bool, error) {
	n, err := parseCAIP2Internal(caip2)
	if err != nil {
		return false, err
	}
	*namespace = n.Namespace
	*chainID = n.ChainID
	return true, nil
}

type caip2Parsed struct {
	Namespace string
	ChainID   int
}

func parseCAIP2Internal(caip2 string) (*caip2Parsed, error) {
	// Simple parsing: namespace:chainId
	for i, c := range caip2 {
		if c == ':' {
			namespace := caip2[:i]
			chainIDStr := caip2[i+1:]
			var chainID int
			for _, d := range chainIDStr {
				if d < '0' || d > '9' {
					return nil, domainerrors.ErrBadRequest
				}
				chainID = chainID*10 + int(d-'0')
			}
			return &caip2Parsed{Namespace: namespace, ChainID: chainID}, nil
		}
	}
	return nil, domainerrors.ErrBadRequest
}

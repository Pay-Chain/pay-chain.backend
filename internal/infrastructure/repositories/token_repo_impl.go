package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

// TokenRepository implements token data operations
type TokenRepository struct {
	db        *gorm.DB
	chainRepo repositories.ChainRepository
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *gorm.DB, chainRepo repositories.ChainRepository) *TokenRepository {
	return &TokenRepository{
		db:        db,
		chainRepo: chainRepo,
	}
}

// GetByID gets a token by ID
func (r *TokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error) {
	var m models.Token
	// Preload Chain
	if err := r.db.WithContext(ctx).Preload("Chain").Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetBySymbol gets a token by symbol and chain ID
func (r *TokenRepository) GetBySymbol(ctx context.Context, symbol string, chainID uuid.UUID) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).Preload("Chain").Where("symbol = ? AND chain_id = ?", symbol, chainID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByAddress gets a token by contract address and chain ID
func (r *TokenRepository) GetByAddress(ctx context.Context, address string, chainID uuid.UUID) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).Preload("Chain").Where("contract_address = ? AND chain_id = ?", address, chainID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetAll gets all tokens
func (r *TokenRepository) GetAll(ctx context.Context) ([]*entities.Token, error) {
	var ms []models.Token
	if err := r.db.WithContext(ctx).Preload("Chain").Order("symbol").Find(&ms).Error; err != nil {
		return nil, err
	}

	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, nil
}

// GetStablecoins gets only stablecoin tokens
func (r *TokenRepository) GetStablecoins(ctx context.Context) ([]*entities.Token, error) {
	var ms []models.Token
	if err := r.db.WithContext(ctx).Preload("Chain").Where("is_stablecoin = ?", true).Order("symbol").Find(&ms).Error; err != nil {
		return nil, err
	}

	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, nil
}

// GetNative gets the native token for a chain
func (r *TokenRepository) GetNative(ctx context.Context, chainID uuid.UUID) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).
		Where("chain_id = ? AND is_native = ?", chainID, true).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetTokensByChain gets tokens supported on a chain
func (r *TokenRepository) GetTokensByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.Token, int64, error) {
	var ms []models.Token
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.Token{}).
		Where("chain_id = ? AND is_active = ?", chainID, true)

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("Chain").Order("symbol")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, totalCount, nil
}

// GetAllTokens gets all tokens with filters
func (r *TokenRepository) GetAllTokens(ctx context.Context, chainID *uuid.UUID, search *string, pagination utils.PaginationParams) ([]*entities.Token, int64, error) {
	var ms []models.Token
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.Token{})

	if chainID != nil {
		query = query.Where("chain_id = ?", *chainID)
	}
	if search != nil && *search != "" {
		term := "%" + *search + "%"
		query = query.Where("symbol ILIKE ? OR name ILIKE ? OR contract_address ILIKE ?", term, term, term)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("Chain").Order("symbol")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, totalCount, nil
}

func (r *TokenRepository) toEntity(m *models.Token) *entities.Token {
	e := &entities.Token{
		ID:              m.ID,
		ChainUUID:       m.ChainID, // Changed ChainID to ChainUUID
		Symbol:          m.Symbol,
		Name:            m.Name,
		Decimals:        m.Decimals,
		LogoURL:         m.LogoURL,
		ContractAddress: m.ContractAddress,
		Type:            entities.TokenType(m.Type),
		IsActive:        m.IsActive,
		IsNative:        m.IsNative,
		IsStablecoin:    m.IsStablecoin,
		MinAmount:       m.MinAmount,
		MaxAmount:       null.StringFromPtr(m.MaxAmount), // Added MaxAmount
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		DeletedAt:       &m.DeletedAt.Time, // Added DeletedAt
	}

	// Populating BlockchainID from Chain if available
	// Populating BlockchainID from Chain if available
	if m.Chain.ID != uuid.Nil {
		// Map directly from preloaded model to avoid N+1
		e.Chain = &entities.Chain{
			ID:             m.Chain.ID,
			ChainID:        m.Chain.NetworkID,
			Name:           m.Chain.Name,
			Type:           entities.ChainType(strings.ToUpper(m.Chain.ChainType)),
			RPCURL:         m.Chain.RPCURL,
			ExplorerURL:    m.Chain.ExplorerURL,
			CurrencySymbol: m.Chain.Symbol,
			ImageURL:       m.Chain.LogoURL,
			IsActive:       m.Chain.IsActive,
			CreatedAt:      m.Chain.CreatedAt,
			UpdatedAt:      m.Chain.UpdatedAt,
		}
		e.BlockchainID = e.Chain.ChainID
	} else {
		// Try to find blockchainId by matching ChainUUID if needed,
		// but usually it should be preloaded.
		e.BlockchainID = "" // Or fallback logic
	}

	return e
}

// Create creates a new token
func (r *TokenRepository) Create(ctx context.Context, token *entities.Token) error {
	m := &models.Token{
		ID:              token.ID,
		ChainID:         token.ChainUUID, // Changed ChainID to ChainUUID
		Symbol:          token.Symbol,
		Name:            token.Name,
		Decimals:        token.Decimals,
		ContractAddress: token.ContractAddress,
		Type:            string(token.Type),
		LogoURL:         token.LogoURL,
		IsActive:        token.IsActive,
		IsNative:        token.IsNative,
		IsStablecoin:    token.IsStablecoin,
		MinAmount:       token.MinAmount,
		MaxAmount:       token.MaxAmount.Ptr(), // Added MaxAmount
		CreatedAt:       token.CreatedAt,
		UpdatedAt:       token.UpdatedAt, // Fixed line break
	}

	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

// Update updates an existing token
func (r *TokenRepository) Update(ctx context.Context, token *entities.Token) error {
	// Simple updates
	return r.db.WithContext(ctx).Save(&models.Token{
		ID: token.ID,
		// ... populate all
	}).Error
}

// SoftDelete soft deletes a token
func (r *TokenRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Token{}, "id = ?", id).Error
}

package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
	"pay-chain.backend/pkg/utils"
)

// TokenRepository implements token data operations
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// GetByID gets a token by ID
func (r *TokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrNotFound
		}
		return nil, err
	}
	return r.toEntity(&m), nil
}

// GetByChainID gets tokens by chain ID
func (r *TokenRepository) GetByChainID(ctx context.Context, chainID uuid.UUID) ([]*entities.Token, error) {
	var ms []models.Token
	if err := r.db.WithContext(ctx).Where("chain_id = ?", chainID).Find(&ms).Error; err != nil {
		return nil, err
	}
	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, nil
}

// GetBySymbol gets a token by symbol and chain ID
func (r *TokenRepository) GetBySymbol(ctx context.Context, symbol string, chainID uuid.UUID) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).Where("symbol = ? AND chain_id = ?", symbol, chainID).First(&m).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Where("contract_address = ? AND chain_id = ?", address, chainID).First(&m).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Order("symbol").Find(&ms).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Where("is_stablecoin = ?", true).Order("symbol").Find(&ms).Error; err != nil {
		return nil, err
	}

	var tokens []*entities.Token
	for _, m := range ms {
		model := m
		tokens = append(tokens, r.toEntity(&model))
	}
	return tokens, nil
}

// GetSupportedByChain gets tokens supported on a chain
func (r *TokenRepository) GetSupportedByChain(ctx context.Context, chainID uuid.UUID, pagination utils.PaginationParams) ([]*entities.SupportedToken, int64, error) {
	var ms []models.SupportedToken
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.SupportedToken{}).
		Where("chain_id = ? AND is_active = ?", chainID, true).
		Joins("JOIN tokens ON tokens.id = supported_tokens.token_id")

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("Token").Order("tokens.symbol")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var supported []*entities.SupportedToken
	for _, m := range ms {
		st := &entities.SupportedToken{
			ID:              m.ID,
			ChainID:         m.ChainID,
			TokenID:         m.TokenID,
			ContractAddress: m.ContractAddress,
			IsActive:        m.IsActive,
			MinAmount:       m.MinAmount,
			// MaxAmount:       null.StringFromPtr(m.MaxAmount),
			CreatedAt: m.CreatedAt,
			Token:     r.toEntity(&m.Token),
		}
		supported = append(supported, st)
	}
	return supported, totalCount, nil
}

// GetAllSupported gets all supported tokens with filters
func (r *TokenRepository) GetAllSupported(ctx context.Context, chainID *uuid.UUID, search *string, pagination utils.PaginationParams) ([]*entities.SupportedToken, int64, error) {
	var ms []models.SupportedToken
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.SupportedToken{}).
		Joins("JOIN tokens ON tokens.id = supported_tokens.token_id")

	if chainID != nil {
		query = query.Where("supported_tokens.chain_id = ?", *chainID)
	}
	if search != nil && *search != "" {
		// Search by Symbol, Name or Contract Address
		query = query.Where(
			"tokens.symbol ILIKE ? OR tokens.name ILIKE ? OR supported_tokens.contract_address ILIKE ?",
			"%"+*search+"%", "%"+*search+"%", "%"+*search+"%",
		)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("Token").Order("tokens.symbol")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit).Offset(pagination.CalculateOffset())
	}

	if err := query.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	var supported []*entities.SupportedToken
	for _, m := range ms {
		st := &entities.SupportedToken{
			ID:              m.ID,
			ChainID:         m.ChainID,
			TokenID:         m.TokenID,
			ContractAddress: m.ContractAddress,
			IsActive:        m.IsActive,
			MinAmount:       m.MinAmount,
			// MaxAmount:       null.StringFromPtr(m.MaxAmount),
			CreatedAt: m.CreatedAt,
			Token:     r.toEntity(&m.Token),
		}
		supported = append(supported, st)
	}
	return supported, totalCount, nil
}

func (r *TokenRepository) toEntity(m *models.Token) *entities.Token {
	return &entities.Token{
		ID:              m.ID,
		Symbol:          m.Symbol,
		Name:            m.Name,
		Decimals:        m.Decimals,
		LogoURL:         m.LogoURL,
		ContractAddress: m.ContractAddress,
		Type:            entities.TokenType(m.Type),
		IsActive:        m.IsActive,
		IsStablecoin:    m.IsStablecoin,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// Create creates a new token and its support entry
func (r *TokenRepository) Create(ctx context.Context, token *entities.Token) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create Token record
		m := &models.Token{
			ID:              token.ID,
			ChainID:         token.ChainID,
			Symbol:          token.Symbol,
			Name:            token.Name,
			Decimals:        token.Decimals,
			ContractAddress: token.ContractAddress,
			Type:            string(token.Type),
			LogoURL:         token.LogoURL,
			IsActive:        token.IsActive,
			IsStablecoin:    token.IsStablecoin,
			CreatedAt:       token.CreatedAt,
			UpdatedAt:       token.UpdatedAt,
		}

		if err := tx.Create(m).Error; err != nil {
			return err
		}

		// 2. Create SupportedToken record
		st := &models.SupportedToken{
			ID:              uuid.New(),
			ChainID:         token.ChainID,
			TokenID:         token.ID,
			ContractAddress: token.ContractAddress,
			IsActive:        token.IsActive,
			CreatedAt:       time.Now(),
		}

		if err := tx.Create(st).Error; err != nil {
			return err
		}

		return nil
	})
}

// Update updates an existing token and syncs relevant fields to support entry
func (r *TokenRepository) Update(ctx context.Context, token *entities.Token) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updateData := map[string]interface{}{
			"symbol":        token.Symbol,
			"name":          token.Name,
			"decimals":      token.Decimals,
			"type":          string(token.Type),
			"logo_url":      token.LogoURL,
			"is_stablecoin": token.IsStablecoin,
			"updated_at":    token.UpdatedAt,
		}

		if err := tx.Model(&models.Token{}).
			Where("id = ?", token.ID).
			Updates(updateData).Error; err != nil {
			return err
		}

		// Also update contract address in supported_tokens table if it exists
		if token.ContractAddress != "" {
			if err := tx.Model(&models.SupportedToken{}).Where("token_id = ?", token.ID).Update("contract_address", token.ContractAddress).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// SoftDelete soft deletes a token and its support entry
func (r *TokenRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete from supported_tokens first (or just the one pointing to this token)
		if err := tx.Where("token_id = ?", id).Delete(&models.SupportedToken{}).Error; err != nil {
			return err
		}

		// Delete from tokens
		if err := tx.Delete(&models.Token{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

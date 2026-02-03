package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/internal/infrastructure/models"
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

// GetBySymbol gets a token by symbol
func (r *TokenRepository) GetBySymbol(ctx context.Context, symbol string) (*entities.Token, error) {
	var m models.Token
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&m).Error; err != nil {
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
func (r *TokenRepository) GetSupportedByChain(ctx context.Context, chainID int) ([]*entities.SupportedToken, error) {
	// Joins tokens
	var ms []models.SupportedToken
	if err := r.db.WithContext(ctx).Preload("Token").
		Where("chain_id = ? AND is_active = ?", chainID, true).
		Joins("JOIN tokens ON tokens.id = supported_tokens.token_id").
		Order("tokens.symbol").
		Find(&ms).Error; err != nil {
		return nil, err
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
	return supported, nil
}

func (r *TokenRepository) toEntity(m *models.Token) *entities.Token {
	return &entities.Token{
		ID:           m.ID,
		Symbol:       m.Symbol,
		Name:         m.Name,
		Decimals:     m.Decimals,
		LogoURL:      m.LogoURL,
		IsStablecoin: m.IsStablecoin,
		CreatedAt:    m.CreatedAt,
	}
}

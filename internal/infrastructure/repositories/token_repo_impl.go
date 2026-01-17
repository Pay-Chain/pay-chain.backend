package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// TokenRepository implements token data operations
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// GetByID gets a token by ID
func (r *TokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Token, error) {
	query := `
		SELECT id, symbol, name, decimals, logo_url, is_stablecoin, created_at
		FROM tokens
		WHERE id = $1 AND deleted_at IS NULL
	`

	token := &entities.Token{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&token.ID,
		&token.Symbol,
		&token.Name,
		&token.Decimals,
		&token.LogoURL,
		&token.IsStablecoin,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return token, nil
}

// GetBySymbol gets a token by symbol
func (r *TokenRepository) GetBySymbol(ctx context.Context, symbol string) (*entities.Token, error) {
	query := `
		SELECT id, symbol, name, decimals, logo_url, is_stablecoin, created_at
		FROM tokens
		WHERE symbol = $1 AND deleted_at IS NULL
	`

	token := &entities.Token{}
	err := r.db.QueryRowContext(ctx, query, symbol).Scan(
		&token.ID,
		&token.Symbol,
		&token.Name,
		&token.Decimals,
		&token.LogoURL,
		&token.IsStablecoin,
		&token.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return token, nil
}

// GetAll gets all tokens
func (r *TokenRepository) GetAll(ctx context.Context) ([]*entities.Token, error) {
	query := `
		SELECT id, symbol, name, decimals, logo_url, is_stablecoin, created_at
		FROM tokens
		WHERE deleted_at IS NULL
		ORDER BY symbol
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*entities.Token
	for rows.Next() {
		token := &entities.Token{}
		if err := rows.Scan(
			&token.ID,
			&token.Symbol,
			&token.Name,
			&token.Decimals,
			&token.LogoURL,
			&token.IsStablecoin,
			&token.CreatedAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// GetStablecoins gets only stablecoin tokens
func (r *TokenRepository) GetStablecoins(ctx context.Context) ([]*entities.Token, error) {
	query := `
		SELECT id, symbol, name, decimals, logo_url, is_stablecoin, created_at
		FROM tokens
		WHERE is_stablecoin = true AND deleted_at IS NULL
		ORDER BY symbol
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*entities.Token
	for rows.Next() {
		token := &entities.Token{}
		if err := rows.Scan(
			&token.ID,
			&token.Symbol,
			&token.Name,
			&token.Decimals,
			&token.LogoURL,
			&token.IsStablecoin,
			&token.CreatedAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// GetSupportedByChain gets tokens supported on a chain
func (r *TokenRepository) GetSupportedByChain(ctx context.Context, chainID int) ([]*entities.SupportedToken, error) {
	query := `
		SELECT st.id, st.chain_id, st.token_id, st.contract_address, st.is_active, 
		       st.min_amount, st.max_amount, st.created_at,
		       t.symbol, t.name, t.decimals
		FROM supported_tokens st
		JOIN tokens t ON st.token_id = t.id
		WHERE st.chain_id = $1 AND st.is_active = true AND st.deleted_at IS NULL
		ORDER BY t.symbol
	`

	rows, err := r.db.QueryContext(ctx, query, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*entities.SupportedToken
	for rows.Next() {
		st := &entities.SupportedToken{}
		if err := rows.Scan(
			&st.ID,
			&st.ChainID,
			&st.TokenID,
			&st.ContractAddress,
			&st.IsActive,
			&st.MinAmount,
			&st.MaxAmount,
			&st.CreatedAt,
			&st.Symbol,
			&st.Name,
			&st.Decimals,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, st)
	}

	return tokens, nil
}

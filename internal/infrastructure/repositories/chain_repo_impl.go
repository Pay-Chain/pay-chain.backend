package postgres

import (
	"context"
	"database/sql"

	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

// ChainRepository implements chain data operations
type ChainRepository struct {
	db *sql.DB
}

// NewChainRepository creates a new chain repository
func NewChainRepository(db *sql.DB) *ChainRepository {
	return &ChainRepository{db: db}
}

// GetByID gets a chain by ID
func (r *ChainRepository) GetByID(ctx context.Context, id int) (*entities.Chain, error) {
	query := `
		SELECT id, namespace, name, chain_type, rpc_url, explorer_url, is_active, created_at
		FROM chains
		WHERE id = $1 AND deleted_at IS NULL
	`

	chain := &entities.Chain{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chain.ID,
		&chain.Namespace,
		&chain.Name,
		&chain.ChainType,
		&chain.RpcURL,
		&chain.ExplorerURL,
		&chain.IsActive,
		&chain.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return chain, nil
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

	query := `
		SELECT id, namespace, name, chain_type, rpc_url, explorer_url, is_active, created_at
		FROM chains
		WHERE namespace = $1 AND id = $2 AND deleted_at IS NULL
	`

	chain := &entities.Chain{}
	err = r.db.QueryRowContext(ctx, query, namespace, chainID).Scan(
		&chain.ID,
		&chain.Namespace,
		&chain.Name,
		&chain.ChainType,
		&chain.RpcURL,
		&chain.ExplorerURL,
		&chain.IsActive,
		&chain.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domainerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// GetAll gets all active chains
func (r *ChainRepository) GetAll(ctx context.Context) ([]*entities.Chain, error) {
	query := `
		SELECT id, namespace, name, chain_type, rpc_url, explorer_url, is_active, created_at
		FROM chains
		WHERE deleted_at IS NULL
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*entities.Chain
	for rows.Next() {
		chain := &entities.Chain{}
		if err := rows.Scan(
			&chain.ID,
			&chain.Namespace,
			&chain.Name,
			&chain.ChainType,
			&chain.RpcURL,
			&chain.ExplorerURL,
			&chain.IsActive,
			&chain.CreatedAt,
		); err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

// GetActive gets only active chains
func (r *ChainRepository) GetActive(ctx context.Context) ([]*entities.Chain, error) {
	query := `
		SELECT id, namespace, name, chain_type, rpc_url, explorer_url, is_active, created_at
		FROM chains
		WHERE is_active = true AND deleted_at IS NULL
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*entities.Chain
	for rows.Next() {
		chain := &entities.Chain{}
		if err := rows.Scan(
			&chain.ID,
			&chain.Namespace,
			&chain.Name,
			&chain.ChainType,
			&chain.RpcURL,
			&chain.ExplorerURL,
			&chain.IsActive,
			&chain.CreatedAt,
		); err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	return chains, nil
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

package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"pay-chain.backend/internal/domain/entities"
)

type PaymentRequestRepository interface {
	Create(ctx context.Context, request *entities.PaymentRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentRequestStatus) error
	UpdateTxHash(ctx context.Context, id uuid.UUID, txHash, payerAddress string) error
	MarkCompleted(ctx context.Context, id uuid.UUID, txHash string) error
	GetExpiredPending(ctx context.Context, limit int) ([]*entities.PaymentRequest, error)
	ExpireRequests(ctx context.Context, ids []uuid.UUID) error
}

type RpcEndpointRepository interface {
	GetByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error)
	GetActiveByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error)
	MarkError(ctx context.Context, id uuid.UUID) error
	ResetErrors(ctx context.Context, chainID int) error
}

type BackgroundJobRepository interface {
	Create(ctx context.Context, job *entities.BackgroundJob) error
	GetPending(ctx context.Context, limit int) ([]*entities.BackgroundJob, error)
	MarkProcessing(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error
}

// PaymentRequestRepositoryImpl implements PaymentRequestRepository
type PaymentRequestRepositoryImpl struct {
	db *sql.DB
}

func NewPaymentRequestRepository(db *sql.DB) *PaymentRequestRepositoryImpl {
	return &PaymentRequestRepositoryImpl{db: db}
}

func (r *PaymentRequestRepositoryImpl) Create(ctx context.Context, req *entities.PaymentRequest) error {
	query := `
		INSERT INTO payment_requests (
			id, merchant_id, wallet_id, chain_id, token_address,
			amount, decimals, description, status, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.MerchantID, req.WalletID, req.ChainID, req.TokenAddress,
		req.Amount, req.Decimals, req.Description, req.Status, req.ExpiresAt,
	)
	return err
}

func (r *PaymentRequestRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.PaymentRequest, error) {
	query := `
		SELECT id, merchant_id, wallet_id, chain_id, token_address,
			   amount, decimals, description, status, expires_at,
			   tx_hash, payer_address, completed_at, created_at, updated_at
		FROM payment_requests
		WHERE id = $1 AND deleted_at IS NULL
	`

	var req entities.PaymentRequest
	var txHash, payerAddress, description sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.MerchantID, &req.WalletID, &req.ChainID, &req.TokenAddress,
		&req.Amount, &req.Decimals, &description, &req.Status, &req.ExpiresAt,
		&txHash, &payerAddress, &completedAt, &req.CreatedAt, &req.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		req.Description = description.String
	}
	if txHash.Valid {
		req.TxHash = txHash.String
	}
	if payerAddress.Valid {
		req.PayerAddress = payerAddress.String
	}
	if completedAt.Valid {
		req.CompletedAt = &completedAt.Time
	}

	return &req, nil
}

func (r *PaymentRequestRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*entities.PaymentRequest, int, error) {
	countQuery := `SELECT COUNT(*) FROM payment_requests WHERE merchant_id = $1 AND deleted_at IS NULL`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, merchantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, merchant_id, wallet_id, chain_id, token_address,
			   amount, decimals, description, status, expires_at,
			   tx_hash, payer_address, completed_at, created_at, updated_at
		FROM payment_requests
		WHERE merchant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, merchantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []*entities.PaymentRequest
	for rows.Next() {
		var req entities.PaymentRequest
		var txHash, payerAddress, description sql.NullString
		var completedAt sql.NullTime

		if err := rows.Scan(
			&req.ID, &req.MerchantID, &req.WalletID, &req.ChainID, &req.TokenAddress,
			&req.Amount, &req.Decimals, &description, &req.Status, &req.ExpiresAt,
			&txHash, &payerAddress, &completedAt, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		if description.Valid {
			req.Description = description.String
		}
		if txHash.Valid {
			req.TxHash = txHash.String
		}
		if payerAddress.Valid {
			req.PayerAddress = payerAddress.String
		}
		if completedAt.Valid {
			req.CompletedAt = &completedAt.Time
		}

		requests = append(requests, &req)
	}

	return requests, total, nil
}

func (r *PaymentRequestRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.PaymentRequestStatus) error {
	query := `UPDATE payment_requests SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PaymentRequestRepositoryImpl) UpdateTxHash(ctx context.Context, id uuid.UUID, txHash, payerAddress string) error {
	query := `UPDATE payment_requests SET tx_hash = $1, payer_address = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, txHash, payerAddress, id)
	return err
}

func (r *PaymentRequestRepositoryImpl) MarkCompleted(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `
		UPDATE payment_requests 
		SET status = 'completed', tx_hash = $1, completed_at = NOW(), updated_at = NOW() 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, txHash, id)
	return err
}

func (r *PaymentRequestRepositoryImpl) GetExpiredPending(ctx context.Context, limit int) ([]*entities.PaymentRequest, error) {
	query := `
		SELECT id, merchant_id, wallet_id, chain_id, token_address,
			   amount, decimals, description, status, expires_at, created_at, updated_at
		FROM payment_requests
		WHERE status = 'pending' AND expires_at < NOW() AND deleted_at IS NULL
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*entities.PaymentRequest
	for rows.Next() {
		var req entities.PaymentRequest
		var description sql.NullString

		if err := rows.Scan(
			&req.ID, &req.MerchantID, &req.WalletID, &req.ChainID, &req.TokenAddress,
			&req.Amount, &req.Decimals, &description, &req.Status, &req.ExpiresAt,
			&req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if description.Valid {
			req.Description = description.String
		}
		requests = append(requests, &req)
	}

	return requests, nil
}

func (r *PaymentRequestRepositoryImpl) ExpireRequests(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query := `UPDATE payment_requests SET status = 'expired', updated_at = NOW() WHERE id = ANY($1)`
	_, err := r.db.ExecContext(ctx, query, ids)
	return err
}

// RpcEndpointRepositoryImpl implements RpcEndpointRepository
type RpcEndpointRepositoryImpl struct {
	db *sql.DB
}

func NewRpcEndpointRepository(db *sql.DB) *RpcEndpointRepositoryImpl {
	return &RpcEndpointRepositoryImpl{db: db}
}

func (r *RpcEndpointRepositoryImpl) GetByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error) {
	query := `
		SELECT id, chain_id, url, priority, is_active, last_error_at, error_count, created_at, updated_at
		FROM rpc_endpoints
		WHERE chain_id = $1
		ORDER BY priority ASC
	`

	return r.scanEndpoints(ctx, query, chainID)
}

func (r *RpcEndpointRepositoryImpl) GetActiveByChainID(ctx context.Context, chainID int) ([]*entities.RpcEndpoint, error) {
	query := `
		SELECT id, chain_id, url, priority, is_active, last_error_at, error_count, created_at, updated_at
		FROM rpc_endpoints
		WHERE chain_id = $1 AND is_active = true
		ORDER BY priority ASC, error_count ASC
	`

	return r.scanEndpoints(ctx, query, chainID)
}

func (r *RpcEndpointRepositoryImpl) scanEndpoints(ctx context.Context, query string, chainID int) ([]*entities.RpcEndpoint, error) {
	rows, err := r.db.QueryContext(ctx, query, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []*entities.RpcEndpoint
	for rows.Next() {
		var ep entities.RpcEndpoint
		var lastErrorAt sql.NullTime

		if err := rows.Scan(
			&ep.ID, &ep.ChainID, &ep.URL, &ep.Priority, &ep.IsActive,
			&lastErrorAt, &ep.ErrorCount, &ep.CreatedAt, &ep.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if lastErrorAt.Valid {
			ep.LastErrorAt = &lastErrorAt.Time
		}
		endpoints = append(endpoints, &ep)
	}

	return endpoints, nil
}

func (r *RpcEndpointRepositoryImpl) MarkError(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE rpc_endpoints 
		SET error_count = error_count + 1, last_error_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *RpcEndpointRepositoryImpl) ResetErrors(ctx context.Context, chainID int) error {
	query := `UPDATE rpc_endpoints SET error_count = 0, updated_at = NOW() WHERE chain_id = $1`
	_, err := r.db.ExecContext(ctx, query, chainID)
	return err
}

// BackgroundJobRepositoryImpl implements BackgroundJobRepository
type BackgroundJobRepositoryImpl struct {
	db *sql.DB
}

func NewBackgroundJobRepository(db *sql.DB) *BackgroundJobRepositoryImpl {
	return &BackgroundJobRepositoryImpl{db: db}
}

func (r *BackgroundJobRepositoryImpl) Create(ctx context.Context, job *entities.BackgroundJob) error {
	query := `
		INSERT INTO background_jobs (id, job_type, payload, status, max_attempts, scheduled_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query, job.ID, job.JobType, job.Payload, job.Status, job.MaxAttempts, job.ScheduledAt)
	return err
}

func (r *BackgroundJobRepositoryImpl) GetPending(ctx context.Context, limit int) ([]*entities.BackgroundJob, error) {
	query := `
		SELECT id, job_type, payload, status, attempts, max_attempts, scheduled_at, 
			   started_at, completed_at, error_message, created_at, updated_at
		FROM background_jobs
		WHERE status = 'pending' AND scheduled_at <= NOW() AND attempts < max_attempts
		ORDER BY scheduled_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*entities.BackgroundJob
	for rows.Next() {
		var job entities.BackgroundJob
		var startedAt, completedAt sql.NullTime
		var errorMsg sql.NullString

		if err := rows.Scan(
			&job.ID, &job.JobType, &job.Payload, &job.Status, &job.Attempts, &job.MaxAttempts,
			&job.ScheduledAt, &startedAt, &completedAt, &errorMsg, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if errorMsg.Valid {
			job.ErrorMessage = errorMsg.String
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

func (r *BackgroundJobRepositoryImpl) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE background_jobs SET status = 'processing', started_at = NOW(), attempts = attempts + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *BackgroundJobRepositoryImpl) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE background_jobs SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *BackgroundJobRepositoryImpl) MarkFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `UPDATE background_jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestNewUnitOfWork_CreatesImplementation(t *testing.T) {
	db := newTestDB(t)
	u := NewUnitOfWork(db)
	require.NotNil(t, u)
}

func TestChainRepository_Create_AndNamespaceMapping(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	repo := NewChainRepository(db).(*chainRepo)
	ctx := context.Background()

	chain := &entities.Chain{
		ID:             uuid.New(),
		ChainID:        "10",
		Name:           "Optimism",
		Type:           entities.ChainTypeEVM,
		RPCURL:         "https://rpc.example",
		ExplorerURL:    "https://scan.example",
		CurrencySymbol: "ETH",
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	require.NoError(t, repo.Create(ctx, chain))

	got, err := repo.GetByID(ctx, chain.ID)
	require.NoError(t, err)
	require.Equal(t, "10", got.ChainID)
	require.Equal(t, "Optimism", got.Name)

	require.Equal(t, "eip155", repo.getNamespace(entities.ChainTypeEVM))
	require.Equal(t, "solana", repo.getNamespace(entities.ChainTypeSVM))
	require.Equal(t, "substrate", repo.getNamespace(entities.ChainTypeSubstrate))
	require.Equal(t, "unknown", repo.getNamespace("UNKNOWN"))
}

func TestTokenRepository_Create_Method(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewTokenRepository(db, nil)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)

	token := &entities.Token{
		ID:              uuid.New(),
		ChainUUID:       chainID,
		Symbol:          "IDRX",
		Name:            "IDRX",
		Decimals:        6,
		ContractAddress: "0x1111",
		Type:            entities.TokenTypeERC20,
		IsActive:        true,
		IsNative:        false,
		IsStablecoin:    true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	require.NoError(t, repo.Create(ctx, token))

	got, err := repo.GetByID(ctx, token.ID)
	require.NoError(t, err)
	require.Equal(t, token.Symbol, got.Symbol)
}

func TestEmailVerificationRepository_CRUDFlow(t *testing.T) {
	db := newTestDB(t)
	createUserTable(t, db)
	mustExec(t, db, `CREATE TABLE email_verifications (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		verified_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)

	userRepo := NewUserRepository(db)
	emailRepo := NewEmailVerificationRepository(db)
	ctx := context.Background()
	now := time.Now()

	user := &entities.User{
		ID:           uuid.New(),
		Email:        "verify@paychain.io",
		Name:         "Verify User",
		PasswordHash: "x",
		Role:         entities.UserRoleUser,
		KYCStatus:    entities.KYCNotStarted,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, userRepo.Create(ctx, user))

	token := "verify-token-1"
	require.NoError(t, emailRepo.Create(ctx, user.ID, token))

	gotUser, err := emailRepo.GetByToken(ctx, token)
	require.NoError(t, err)
	require.Equal(t, user.ID, gotUser.ID)

	require.NoError(t, emailRepo.MarkVerified(ctx, token))

	_, err = emailRepo.GetByToken(ctx, token)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = emailRepo.MarkVerified(ctx, token)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

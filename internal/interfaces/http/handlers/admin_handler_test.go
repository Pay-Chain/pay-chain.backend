package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
	"payment-kita.backend/internal/infrastructure/repositories"
)

type adminUserRepoStub struct {
	listFn func(ctx context.Context, search string) ([]*entities.User, error)
}

func (s *adminUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (s *adminUserRepoStub) GetByID(context.Context, uuid.UUID) (*entities.User, error) {
	return nil, nil
}
func (s *adminUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, nil
}
func (s *adminUserRepoStub) Update(context.Context, *entities.User) error            { return nil }
func (s *adminUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (s *adminUserRepoStub) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (s *adminUserRepoStub) List(ctx context.Context, search string) ([]*entities.User, error) {
	return s.listFn(ctx, search)
}

type adminMerchantRepoStub struct {
	listFn   func(ctx context.Context) ([]*entities.Merchant, error)
	getByID  func(ctx context.Context, id uuid.UUID) (*entities.Merchant, error)
	updateFn func(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error
}

func (s *adminMerchantRepoStub) Create(context.Context, *entities.Merchant) error { return nil }
func (s *adminMerchantRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*entities.Merchant, error) {
	return s.getByID(ctx, id)
}
func (s *adminMerchantRepoStub) GetByUserID(context.Context, uuid.UUID) (*entities.Merchant, error) {
	return nil, nil
}
func (s *adminMerchantRepoStub) Update(context.Context, *entities.Merchant) error { return nil }
func (s *adminMerchantRepoStub) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.MerchantStatus) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, id, status)
	}
	return nil
}
func (s *adminMerchantRepoStub) SoftDelete(context.Context, uuid.UUID) error { return nil }
func (s *adminMerchantRepoStub) List(ctx context.Context) ([]*entities.Merchant, error) {
	return s.listFn(ctx)
}

type adminPaymentRepoStub struct{}

func (adminPaymentRepoStub) Create(context.Context, *entities.Payment) error { return nil }
func (adminPaymentRepoStub) GetByID(context.Context, uuid.UUID) (*entities.Payment, error) {
	return nil, nil
}
func (adminPaymentRepoStub) GetByUserID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (adminPaymentRepoStub) GetByMerchantID(context.Context, uuid.UUID, int, int) ([]*entities.Payment, int, error) {
	return nil, 0, nil
}
func (adminPaymentRepoStub) UpdateStatus(context.Context, uuid.UUID, entities.PaymentStatus) error {
	return nil
}
func (adminPaymentRepoStub) UpdateDestTxHash(context.Context, uuid.UUID, string) error { return nil }
func (adminPaymentRepoStub) MarkRefunded(context.Context, uuid.UUID) error             { return nil }
func (adminPaymentRepoStub) Update(context.Context, *entities.Payment) error           { return nil }

func TestAdminHandler_ListAndUpdateStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	merchantID := uuid.New()

	h := NewAdminHandler(
		&adminUserRepoStub{
			listFn: func(_ context.Context, search string) ([]*entities.User, error) {
				if search != "abc" {
					t.Fatalf("unexpected search %s", search)
				}
				return []*entities.User{{ID: uuid.New(), Email: "u@paymentkita.io"}}, nil
			},
		},
		&adminMerchantRepoStub{
			listFn: func(context.Context) ([]*entities.Merchant, error) {
				return []*entities.Merchant{{ID: merchantID, BusinessName: "Biz"}}, nil
			},
			getByID: func(_ context.Context, id uuid.UUID) (*entities.Merchant, error) {
				if id == merchantID {
					return &entities.Merchant{ID: merchantID}, nil
				}
				return nil, domainerrors.ErrNotFound
			},
		},
		adminPaymentRepoStub{},
		nil,
	)

	r := gin.New()
	r.GET("/users", h.ListUsers)
	r.GET("/merchants", h.ListMerchants)
	r.PUT("/merchants/:id/status", h.UpdateMerchantStatus)

	req := httptest.NewRequest(http.MethodGet, "/users?search=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "u@paymentkita.io")

	req = httptest.NewRequest(http.MethodGet, "/merchants", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Biz")

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+merchantID.String()+"/status", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Merchant status updated")

	req = httptest.NewRequest(http.MethodPut, "/merchants/"+uuid.NewString()+"/status", strings.NewReader(`{"status":"ACTIVE"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminHandler_GetSettlementProfileGaps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testDB := repositoriesTestDBForAdminSettlement(t)
	repositoriesTestCreateAdminSettlementTables(t, testDB)
	now := time.Now().UTC()
	merchantID := uuid.New()
	missingID := uuid.New()
	chainID := uuid.New()

	mustExecAdminSettlement(t, testDB, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, merchantID.String(), uuid.NewString(), "Configured", "configured@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, missingID.String(), uuid.NewString(), "Missing", "missing@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO chains (id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, chainID.String(), "8453", "Base", "EVM", "", "", "ETH", "", true, "", "", 0, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), chainID.String(), "IDRX", "IDRX", 2, "0xidrxtoken", "ERC20", "", true, false, false, "0", nil, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), chainID.String(), "USDC", "USDC", 6, "0xusdctoken", "ERC20", "", true, false, true, "0", nil, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO merchant_settlement_profiles (id, merchant_id, invoice_currency, dest_chain, dest_token, dest_wallet, bridge_token_symbol, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`, uuid.NewString(), merchantID.String(), "IDRX", "eip155:8453", "0xidrxtoken", "0xwallet", "USDC", now, now)

	h := NewAdminHandler(
		nil,
		repositories.NewMerchantRepository(testDB),
		adminPaymentRepoStub{},
		repositories.NewMerchantSettlementProfileRepository(testDB),
	)

	r := gin.New()
	r.GET("/admin/diagnostics/settlement-profile-gaps", h.GetSettlementProfileGaps)

	req := httptest.NewRequest(http.MethodGet, "/admin/diagnostics/settlement-profile-gaps", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"total_missing":1`)
	require.Contains(t, w.Body.String(), "missing@example.com")
	require.NotContains(t, w.Body.String(), "configured@example.com")
}

package usecases_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/infrastructure/blockchain"
	"payment-kita.backend/internal/usecases"
)

func TestPaymentUsecase_GetPaymentPrivacyStatus_UsesOnchainForwardCompletionFallback(t *testing.T) {
	paymentRepo := new(MockPaymentRepository)
	eventRepo := new(MockPaymentEventRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo := new(MockChainRepository)
	clientFactory := blockchain.NewClientFactory()

	uc := usecases.NewPaymentUsecase(
		paymentRepo,
		eventRepo,
		new(MockWalletRepository),
		new(MockMerchantRepository),
		contractRepo,
		chainRepo,
		new(MockTokenRepository),
		nil,
		nil,
		nil,
		new(MockUnitOfWork),
		clientFactory,
	)

	paymentID := uuid.New()
	chainID := uuid.New()
	rpcURL := "https://unit-test.invalid"
	gatewayAddress := "0x1111111111111111111111111111111111111111"
	settledToken := common.HexToAddress("0x2222222222222222222222222222222222222222")
	pidBytes := uuidToBytes32ForTest(paymentID)

	payment := &entities.Payment{
		ID:            paymentID,
		SourceChainID: chainID,
		Status:        entities.PaymentStatusCompleted,
	}
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_PAYMENT_CREATED")},
	}
	sourceChain := &entities.Chain{
		ID:      chainID,
		ChainID: "8453",
		Type:    entities.ChainTypeEVM,
		RPCURL:  rpcURL,
	}
	gatewayContract := &entities.SmartContract{
		ChainUUID:       chainID,
		ContractAddress: gatewayAddress,
		Type:            entities.ContractTypeGateway,
	}

	chainRepo.On("GetByID", context.Background(), chainID).Return(sourceChain, nil).Once()
	contractRepo.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(gatewayContract, nil).Once()
	paymentRepo.On("GetByID", context.Background(), paymentID).Return(payment, nil).Once()
	eventRepo.On("GetByPaymentID", context.Background(), paymentID).Return(events, nil).Once()

	clientFactory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(8453), func(_ context.Context, to string, data []byte) ([]byte, error) {
		assert.Equal(t, gatewayAddress, to)
		selector := common.Bytes2Hex(data[:4])
		switch selector {
		case selectorHexForTest("privacyIntentByPayment(bytes32)"):
			return packReturnValueForTest(t, "bytes32", pidBytes), nil
		case selectorHexForTest("privacyStealthByPayment(bytes32)"):
			return packReturnValueForTest(t, "address", common.HexToAddress("0x3333333333333333333333333333333333333333")), nil
		case selectorHexForTest("privacyForwardCompleted(bytes32)"):
			return packReturnValueForTest(t, "bool", true), nil
		case selectorHexForTest("privacyForwardRetryCount(bytes32)"):
			return packReturnValueForTest(t, "uint8", uint8(0)), nil
		case selectorHexForTest("paymentSettledToken(bytes32)"):
			return packReturnValueForTest(t, "address", settledToken), nil
		case selectorHexForTest("paymentSettledAmount(bytes32)"):
			return packReturnValueForTest(t, "uint256", big.NewInt(12345)), nil
		default:
			t.Fatalf("unexpected selector: %s", selector)
			return nil, nil
		}
	}))

	privacy, err := uc.GetPaymentPrivacyStatus(context.Background(), paymentID)
	assert.NoError(t, err)
	assert.Equal(t, entities.PrivacyLifecycleForwardedFinal, privacy.Stage)
	assert.True(t, privacy.IsPrivacyCandidate)
	assert.Contains(t, privacy.Signals, "onchain_privacy_forward_completed")
	assert.Contains(t, privacy.Signals, "onchain_privacy_settlement_present")
}

func TestPaymentUsecase_GetPaymentPrivacyStatus_UsesOnchainRetryFallbackWhenEventsTrimmed(t *testing.T) {
	paymentRepo := new(MockPaymentRepository)
	eventRepo := new(MockPaymentEventRepository)
	contractRepo := new(MockSmartContractRepository)
	chainRepo := new(MockChainRepository)
	clientFactory := blockchain.NewClientFactory()

	uc := usecases.NewPaymentUsecase(
		paymentRepo,
		eventRepo,
		new(MockWalletRepository),
		new(MockMerchantRepository),
		contractRepo,
		chainRepo,
		new(MockTokenRepository),
		nil,
		nil,
		nil,
		new(MockUnitOfWork),
		clientFactory,
	)

	paymentID := uuid.New()
	chainID := uuid.New()
	rpcURL := "https://unit-test-2.invalid"
	gatewayAddress := "0x4444444444444444444444444444444444444444"

	payment := &entities.Payment{
		ID:            paymentID,
		SourceChainID: chainID,
		Status:        entities.PaymentStatusFailed,
	}
	payment.FailureReason.SetValid("privacy forward failed after settlement")
	events := []*entities.PaymentEvent{
		{EventType: entities.PaymentEventType("PRIVACY_PAYMENT_CREATED")},
	}
	sourceChain := &entities.Chain{
		ID:      chainID,
		ChainID: "42161",
		Type:    entities.ChainTypeEVM,
		RPCURL:  rpcURL,
	}
	gatewayContract := &entities.SmartContract{
		ChainUUID:       chainID,
		ContractAddress: gatewayAddress,
		Type:            entities.ContractTypeGateway,
	}

	chainRepo.On("GetByID", context.Background(), chainID).Return(sourceChain, nil).Once()
	contractRepo.On("GetActiveContract", context.Background(), chainID, entities.ContractTypeGateway).Return(gatewayContract, nil).Once()
	paymentRepo.On("GetByID", context.Background(), paymentID).Return(payment, nil).Once()
	eventRepo.On("GetByPaymentID", context.Background(), paymentID).Return(events, nil).Once()

	clientFactory.RegisterEVMClient(rpcURL, blockchain.NewEVMClientWithCallView(big.NewInt(42161), func(_ context.Context, to string, data []byte) ([]byte, error) {
		assert.Equal(t, gatewayAddress, to)
		selector := common.Bytes2Hex(data[:4])
		switch selector {
		case selectorHexForTest("privacyIntentByPayment(bytes32)"):
			return packReturnValueForTest(t, "bytes32", uuidToBytes32ForTest(paymentID)), nil
		case selectorHexForTest("privacyStealthByPayment(bytes32)"):
			return packReturnValueForTest(t, "address", common.HexToAddress("0x5555555555555555555555555555555555555555")), nil
		case selectorHexForTest("privacyForwardCompleted(bytes32)"):
			return packReturnValueForTest(t, "bool", false), nil
		case selectorHexForTest("privacyForwardRetryCount(bytes32)"):
			return packReturnValueForTest(t, "uint8", uint8(1)), nil
		case selectorHexForTest("paymentSettledToken(bytes32)"):
			return packReturnValueForTest(t, "address", common.Address{}), nil
		case selectorHexForTest("paymentSettledAmount(bytes32)"):
			return packReturnValueForTest(t, "uint256", big.NewInt(0)), nil
		default:
			t.Fatalf("unexpected selector: %s", selector)
			return nil, nil
		}
	}))

	privacy, err := uc.GetPaymentPrivacyStatus(context.Background(), paymentID)
	assert.NoError(t, err)
	assert.Equal(t, entities.PrivacyLifecycleRefundable, privacy.Stage)
	assert.True(t, privacy.IsPrivacyCandidate)
	assert.Contains(t, privacy.Signals, "onchain_privacy_forward_retry_count_present")
}

func selectorHexForTest(sig string) string {
	return common.Bytes2Hex(crypto.Keccak256([]byte(sig))[:4])
}

func packReturnValueForTest(t *testing.T, typeName string, value interface{}) []byte {
	t.Helper()
	typ, err := abi.NewType(typeName, "", nil)
	if err != nil {
		t.Fatalf("abi.NewType(%s): %v", typeName, err)
	}
	out, err := abi.Arguments{{Type: typ}}.Pack(value)
	if err != nil {
		t.Fatalf("pack %s return: %v", typeName, err)
	}
	return out
}

func uuidToBytes32ForTest(id uuid.UUID) [32]byte {
	var out [32]byte
	copy(out[:16], id[:])
	return out
}

package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/usecases"
)

func TestResolveABI_LoadsFromDB(t *testing.T) {
	mockRepo := new(MockSmartContractRepository)
	resolver := usecases.NewABIResolverMixin(mockRepo)

	ctx := context.Background()
	chainID := uuid.New()
	contractType := entities.ContractTypeGateway

	// Valid ABI JSON
	validABI := `[{"constant":true,"inputs":[],"name":"test","outputs":[],"payable":false,"stateMutability":"view","type":"function"}]`

	// Create map[string]interface{} for JSONB unmarshal simulation
	var abiObj interface{}
	_ = json.Unmarshal([]byte(validABI), &abiObj)

	contract := &entities.SmartContract{
		ContractAddress: "0x123",
		ABI:             abiObj,
	}

	mockRepo.On("GetActiveContract", ctx, chainID, contractType).Return(contract, nil)

	resolved, addr, err := resolver.ResolveABI(ctx, chainID, contractType)
	assert.NoError(t, err)
	assert.Equal(t, "0x123", addr)
	assert.NotNil(t, resolved)
	// Check if methods exist
	method, exist := resolved.Methods["test"]
	assert.True(t, exist)
	assert.Equal(t, "test", method.Name)
}

func TestResolveABI_FallbackOnNilABI(t *testing.T) {
	mockRepo := new(MockSmartContractRepository)
	resolver := usecases.NewABIResolverMixin(mockRepo)

	ctx := context.Background()
	chainID := uuid.New()
	contractType := entities.ContractTypeGateway

	// Contract exists but ABI is nil
	contract := &entities.SmartContract{
		ContractAddress: "0x123",
		ABI:             nil,
	}

	mockRepo.On("GetActiveContract", ctx, chainID, contractType).Return(contract, nil)

	// ResolveABI should return nil, address, nil
	resolved, addr, err := resolver.ResolveABI(ctx, chainID, contractType)
	assert.NoError(t, err)
	assert.Equal(t, "0x123", addr)
	assert.Nil(t, resolved)

	// ResolveABIWithFallback should return fallback
	fallback, err := resolver.ResolveABIWithFallback(ctx, chainID, contractType)
	assert.NoError(t, err)
	assert.NotNil(t, fallback)
	_, exists := fallback.Methods["quoteTotalAmount"]
	assert.True(t, exists)
}

func TestResolveABI_ErrorPropagation(t *testing.T) {
	mockRepo := new(MockSmartContractRepository)
	resolver := usecases.NewABIResolverMixin(mockRepo)

	ctx := context.Background()
	chainID := uuid.New()
	contractType := entities.ContractTypeGateway

	expectedErr := errors.New("db error")
	mockRepo.On("GetActiveContract", ctx, chainID, contractType).Return(nil, expectedErr)

	_, _, err := resolver.ResolveABI(ctx, chainID, contractType)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr) || strings.Contains(err.Error(), "db error"))
}

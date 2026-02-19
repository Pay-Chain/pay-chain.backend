package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	derrs "pay-chain.backend/internal/domain/errors"
)

func TestEVMAdminOpsService_RegisterAndDefault(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	resolved := &evmAdminContext{
		sourceChainID:  sourceID,
		destCAIP2:      "eip155:42161",
		routerAddress:  "0x1111111111111111111111111111111111111111",
		gatewayAddress: "0x2222222222222222222222222222222222222222",
	}

	sendTx := func(_ context.Context, chainID uuid.UUID, contractAddress string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
		require.Equal(t, sourceID, chainID)
		switch method {
		case "registerAdapter":
			require.Equal(t, resolved.routerAddress, contractAddress)
		case "setDefaultBridgeType":
			require.Equal(t, resolved.gatewayAddress, contractAddress)
		}
		return "0xtxhash", nil
	}

	mockResolveABI := func(_ context.Context, _ uuid.UUID, _ entities.SmartContractType) (abi.ABI, error) {
		return abi.ABI{}, nil
	}

	svc := newEVMAdminOpsService(
		func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
		func(context.Context, uuid.UUID, string, string, uint8) (string, error) { return "", nil },
		sendTx,
		mockResolveABI,
	)

	_, err := svc.RegisterAdapter(ctx, "eip155:8453", "eip155:42161", 0, "not-hex")
	require.Error(t, err)

	_, err = svc.RegisterAdapter(ctx, "eip155:8453", "eip155:42161", 0, "0x0000000000000000000000000000000000000000")
	require.Error(t, err)

	tx, err := svc.RegisterAdapter(ctx, "eip155:8453", "eip155:42161", 0, "0x3333333333333333333333333333333333333333")
	require.NoError(t, err)
	require.Equal(t, "0xtxhash", tx)

	tx, err = svc.SetDefaultBridgeType(ctx, "eip155:8453", "eip155:42161", 1)
	require.NoError(t, err)
	require.Equal(t, "0xtxhash", tx)

	svcResolveErr := newEVMAdminOpsService(
		func(context.Context, string, string) (*evmAdminContext, error) {
			return nil, errors.New("resolve failed")
		},
		nil,
		sendTx,
		mockResolveABI,
	)
	_, err = svcResolveErr.SetDefaultBridgeType(ctx, "eip155:8453", "eip155:42161", 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve failed")

	svcRegisterResolveErr := newEVMAdminOpsService(
		func(context.Context, string, string) (*evmAdminContext, error) {
			return nil, errors.New("register resolve failed")
		},
		nil,
		sendTx,
		mockResolveABI,
	)
	_, err = svcRegisterResolveErr.RegisterAdapter(ctx, "eip155:8453", "eip155:42161", 1, "0x3333333333333333333333333333333333333333")
	require.Error(t, err)
	require.Contains(t, err.Error(), "register resolve failed")

	svcRegisterTxErr := newEVMAdminOpsService(
		func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
		nil,
		func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
			return "", errors.New("tx failed")
		},
		mockResolveABI,
	)
	_, err = svcRegisterTxErr.RegisterAdapter(ctx, "eip155:8453", "eip155:42161", 1, "0x3333333333333333333333333333333333333333")
	require.Error(t, err)
	require.Contains(t, err.Error(), "tx failed")
}

func TestEVMAdminOpsService_SetHyperbridgeConfig(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	resolved := &evmAdminContext{
		sourceChainID: sourceID,
		destCAIP2:     "eip155:42161",
		routerAddress: "0x1111111111111111111111111111111111111111",
	}

	mockResolveABI := func(_ context.Context, _ uuid.UUID, _ entities.SmartContractType) (abi.ABI, error) {
		return abi.ABI{}, nil
	}

	t.Run("adapter not registered", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x0000000000000000000000000000000000000000", nil
			},
			nil,
			mockResolveABI,
		)
		_, _, err := svc.SetHyperbridgeConfig(ctx, "eip155:8453", "eip155:42161", "0x01", "")
		require.Error(t, err)
	})

	t.Run("payload required", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x3333333333333333333333333333333333333333", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetHyperbridgeConfig(ctx, "eip155:8453", "eip155:42161", "", "")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, "stateMachineId or destinationContract is required", appErr.Message)
	})

	t.Run("success and tx failure", func(t *testing.T) {
		calls := 0
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x3333333333333333333333333333333333333333", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				calls++
				if method == "setDestinationContract" {
					return "", errors.New("tx failed")
				}
				return "0xtx1", nil
			},
			mockResolveABI,
		)

		_, txs, err := svc.SetHyperbridgeConfig(ctx, "eip155:8453", "eip155:42161", "0x01", "0x02")
		require.Error(t, err)
		require.Equal(t, 1, len(txs))
		require.Equal(t, 2, calls)
	})
}

func TestEVMAdminOpsService_SetCCIPAndLayerZeroConfig(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	resolved := &evmAdminContext{
		sourceChainID: sourceID,
		destCAIP2:     "eip155:42161",
		routerAddress: "0x1111111111111111111111111111111111111111",
	}

	ccipSelector := uint64(123)
	lzEid := uint32(101)

	mockResolveABI := func(_ context.Context, _ uuid.UUID, _ entities.SmartContractType) (abi.ABI, error) {
		return abi.ABI{}, nil
	}

	t.Run("ccip success", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x4444444444444444444444444444444444444444", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "0xtx", nil
			},
			mockResolveABI,
		)
		adapter, txs, err := svc.SetCCIPConfig(ctx, "eip155:8453", "eip155:42161", &ccipSelector, "0xabcd")
		require.NoError(t, err)
		require.Equal(t, "0x4444444444444444444444444444444444444444", adapter)
		require.Len(t, txs, 2)
	})

	t.Run("layerzero invalid peer and success", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "0xtx", nil
			},
			mockResolveABI,
		)

		_, _, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", &lzEid, "0x12", "")
		require.Error(t, err)

		adapter, txs, err := svc.SetLayerZeroConfig(
			ctx,
			"eip155:8453",
			"eip155:42161",
			&lzEid,
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0102",
		)
		require.NoError(t, err)
		require.Equal(t, "0x5555555555555555555555555555555555555555", adapter)
		require.Len(t, txs, 2)
	})

	t.Run("ccip adapter missing and payload required", func(t *testing.T) {
		svcMissing := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x0000000000000000000000000000000000000000", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err := svcMissing.SetCCIPConfig(ctx, "eip155:8453", "eip155:42161", &ccipSelector, "")
		require.Error(t, err)

		svcRequired := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x4444444444444444444444444444444444444444", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err = svcRequired.SetCCIPConfig(ctx, "eip155:8453", "eip155:42161", nil, "")
		require.Error(t, err)
	})

	t.Run("layerzero adapter lookup error and partial payload error", func(t *testing.T) {
		svcLookupErr := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "", errors.New("adapter lookup failed")
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err := svcLookupErr.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", &lzEid, "0x"+strings.Repeat("1", 64), "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "adapter lookup failed")

		svcPartial := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err = svcPartial.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", &lzEid, "", "")
		require.Error(t, err)
		_, _, err = svcPartial.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", nil, "0x"+strings.Repeat("1", 64), "")
		require.Error(t, err)
	})

	t.Run("layerzero options only success", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				require.Equal(t, "setEnforcedOptions", method)
				return "0xtx-options", nil
			},
			mockResolveABI,
		)
		adapter, txs, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", nil, "", "0102")
		require.NoError(t, err)
		require.Equal(t, "0x5555555555555555555555555555555555555555", adapter)
		require.Equal(t, []string{"0xtx-options"}, txs)
	})

	t.Run("ccip selector tx error", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x4444444444444444444444444444444444444444", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				if method == "setChainSelector" {
					return "", errors.New("set selector failed")
				}
				return "0xok", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetCCIPConfig(ctx, "eip155:8453", "eip155:42161", &ccipSelector, "")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Contains(t, appErr.Message, "setChainSelector failed")
		require.Contains(t, appErr.Message, "set selector failed")
	})

	t.Run("ccip destination tx error", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x4444444444444444444444444444444444444444", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				if method == "setDestinationAdapter" {
					return "", errors.New("set destination failed")
				}
				return "0xok", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetCCIPConfig(ctx, "eip155:8453", "eip155:42161", nil, "0xabcd")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Contains(t, appErr.Message, "setDestinationAdapter failed")
		require.Contains(t, appErr.Message, "set destination failed")
	})

	t.Run("layerzero adapter missing", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x0000000000000000000000000000000000000000", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", &lzEid, "0x"+strings.Repeat("1", 64), "")
		require.Error(t, err)
	})

	t.Run("layerzero setRoute tx error", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				if method == "setRoute" {
					return "", errors.New("set route failed")
				}
				return "0xok", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", &lzEid, "0x"+strings.Repeat("1", 64), "")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Contains(t, appErr.Message, "setRoute failed")
		require.Contains(t, appErr.Message, "set route failed")
	})

	t.Run("layerzero options tx error", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				if method == "setEnforcedOptions" {
					return "", errors.New("set options failed")
				}
				return "0xok", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", nil, "", "0x0102")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Contains(t, appErr.Message, "setEnforcedOptions failed")
		require.Contains(t, appErr.Message, "set options failed")
	})

	t.Run("layerzero payload required", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x5555555555555555555555555555555555555555", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "unused", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetLayerZeroConfig(ctx, "eip155:8453", "eip155:42161", nil, "", "")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, "dstEid+peerHex or optionsHex is required", appErr.Message)
	})
}

func TestEVMAdminOpsService_SetHyperbridgeConfig_MoreBranches(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	resolved := &evmAdminContext{
		sourceChainID: sourceID,
		destCAIP2:     "eip155:42161",
		routerAddress: "0x1111111111111111111111111111111111111111",
	}

	mockResolveABI := func(_ context.Context, _ uuid.UUID, _ entities.SmartContractType) (abi.ABI, error) {
		return abi.ABI{}, nil
	}

	t.Run("setStateMachine tx error", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x3333333333333333333333333333333333333333", nil
			},
			func(_ context.Context, _ uuid.UUID, _ string, _ abi.ABI, method string, _ ...interface{}) (string, error) {
				if method == "setStateMachineId" {
					return "", errors.New("set state machine failed")
				}
				return "0xok", nil
			},
			mockResolveABI,
		)
		_, _, err := svc.SetHyperbridgeConfig(ctx, "eip155:8453", "eip155:42161", "0x01", "")
		require.Error(t, err)
		var appErr *derrs.AppError
		require.ErrorAs(t, err, &appErr)
		require.Contains(t, appErr.Message, "setStateMachineId failed")
		require.Contains(t, appErr.Message, "set state machine failed")
	})

	t.Run("destination only success", func(t *testing.T) {
		svc := newEVMAdminOpsService(
			func(context.Context, string, string) (*evmAdminContext, error) { return resolved, nil },
			func(context.Context, uuid.UUID, string, string, uint8) (string, error) {
				return "0x3333333333333333333333333333333333333333", nil
			},
			func(context.Context, uuid.UUID, string, abi.ABI, string, ...interface{}) (string, error) {
				return "0xtx-dest", nil
			},
			mockResolveABI,
		)
		adapter, txs, err := svc.SetHyperbridgeConfig(ctx, "eip155:8453", "eip155:42161", "", "0x02")
		require.NoError(t, err)
		require.Equal(t, "0x3333333333333333333333333333333333333333", adapter)
		require.Equal(t, []string{"0xtx-dest"}, txs)
	})
}

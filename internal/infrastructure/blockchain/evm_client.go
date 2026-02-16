package blockchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	dialEVMClient = ethclient.Dial
	getClientChainID = func(client *ethclient.Client, ctx context.Context) (*big.Int, error) {
		return client.ChainID(ctx)
	}
)

// EVMClient provides EVM blockchain interaction
type EVMClient struct {
	client  *ethclient.Client
	chainID *big.Int
	rpcURL  string
	// testCallView allows deterministic unit tests without network sockets.
	testCallView func(ctx context.Context, to string, data []byte) ([]byte, error)
}

// NewEVMClient creates a new EVM client
func NewEVMClient(rpcURL string) (*EVMClient, error) {
	client, err := dialEVMClient(rpcURL)
	if err != nil {
		return nil, err
	}

	chainID, err := getClientChainID(client, context.Background())
	if err != nil {
		return nil, err
	}

	return &EVMClient{
		client:  client,
		chainID: chainID,
		rpcURL:  rpcURL,
	}, nil
}

// NewEVMClientWithCallView creates an EVM client that uses an injected CallView implementation.
// This is intended for unit tests where RPC sockets are unavailable.
func NewEVMClientWithCallView(chainID *big.Int, callViewFn func(ctx context.Context, to string, data []byte) ([]byte, error)) *EVMClient {
	if chainID == nil {
		chainID = big.NewInt(1)
	}
	return &EVMClient{
		chainID:      chainID,
		testCallView: callViewFn,
	}
}

// ChainID returns the chain ID
func (c *EVMClient) ChainID() *big.Int {
	return c.chainID
}

// GetBalance gets the native token balance of an address
func (c *EVMClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	addr := common.HexToAddress(address)
	return c.client.BalanceAt(ctx, addr, nil)
}

// GetTokenBalance gets the ERC20 token balance of an address
func (c *EVMClient) GetTokenBalance(ctx context.Context, tokenAddress, ownerAddress string) (*big.Int, error) {
	token := common.HexToAddress(tokenAddress)
	owner := common.HexToAddress(ownerAddress)

	// balanceOf(address) selector: 0x70a08231
	data := append(common.Hex2Bytes("70a08231"), common.LeftPadBytes(owner.Bytes(), 32)...)

	msg := ethereum.CallMsg{
		To:   &token,
		Data: data,
	}

	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(result), nil
}

// GetTransaction gets transaction details
func (c *EVMClient) GetTransaction(ctx context.Context, txHash string) (*types.Transaction, bool, error) {
	hash := common.HexToHash(txHash)
	return c.client.TransactionByHash(ctx, hash)
}

// GetTransactionReceipt gets transaction receipt
func (c *EVMClient) GetTransactionReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	return c.client.TransactionReceipt(ctx, hash)
}

// GetBlockNumber gets the latest block number
func (c *EVMClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	return c.client.BlockNumber(ctx)
}

// EstimateGas estimates gas for a transaction
func (c *EVMClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return c.client.EstimateGas(ctx, msg)
}

// CallView executes a read-only contract call
func (c *EVMClient) CallView(ctx context.Context, to string, data []byte) ([]byte, error) {
	if c.testCallView != nil {
		return c.testCallView(ctx, to, data)
	}
	addr := common.HexToAddress(to)
	msg := ethereum.CallMsg{
		To:   &addr,
		Data: data,
	}
	return c.client.CallContract(ctx, msg, nil)
}

// Close closes the client connection
func (c *EVMClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

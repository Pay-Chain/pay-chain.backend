package blockchain

import (
	"fmt"
	"sync"
)

// ClientFactory manages blockchain clients
type ClientFactory struct {
	evmClients    map[string]*EVMClient
	solanaClients map[string]interface{}
	mu            sync.RWMutex
}

// NewClientFactory creates a new client factory
func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		evmClients:    make(map[string]*EVMClient),
		solanaClients: make(map[string]interface{}),
	}
}

// GetEVMClient returns an EVM client for the given RPC URL
// If a client already exists for the URL, it returns the cached client
func (f *ClientFactory) GetEVMClient(rpcURL string) (*EVMClient, error) {
	f.mu.RLock()
	client, ok := f.evmClients[rpcURL]
	f.mu.RUnlock()
	if ok {
		return client, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double check
	if client, ok := f.evmClients[rpcURL]; ok {
		return client, nil
	}

	newClient, err := NewEVMClient(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create EVM client: %w", err)
	}

	f.evmClients[rpcURL] = newClient
	return newClient, nil
}

// RegisterEVMClient injects/overrides cached client for a specific rpcURL.
// Useful for deterministic unit tests.
func (f *ClientFactory) RegisterEVMClient(rpcURL string, client *EVMClient) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.evmClients[rpcURL] = client
}

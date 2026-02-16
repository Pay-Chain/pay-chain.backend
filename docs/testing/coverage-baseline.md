# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-16

## Summary
- Global total (all packages): **69.0%**
- `make build`: **PASS**
- Major blockers to higher coverage in this sandbox:
`httptest.NewServer` and `miniredis` often cannot bind local ports, so several integration-like tests are skipped.

## Package Snapshot
- `cmd/admin-apikey`: **14.3%**
- `cmd/apikey-gen`: **81.5%**
- `cmd/genhash`: **85.7%**
- `cmd/hash-gen`: **90.9%**
- `cmd/server`: **62.2%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **87.5%**
- `internal/domain/errors`: **91.7%**
- `internal/infrastructure/blockchain`: **34.9%**
- `internal/infrastructure/datasources/postgres`: **81.8%**
- `internal/infrastructure/jobs`: **92.3%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **82.8%**
- `internal/interfaces/http/handlers`: **72.7%**
- `internal/interfaces/http/middleware`: **70.6%**
- `internal/interfaces/http/response`: **100.0%**
- `internal/usecases`: **59.4%**
- `pkg/crypto`: **81.8%**
- `pkg/jwt`: **79.2%**
- `pkg/logger`: **89.3%**
- `pkg/redis`: **64.6%**
- `pkg/utils`: **83.3%**

## High-Priority Gaps
1. `internal/usecases/crosschain_config_usecase.go`
- `Overview`, `RecheckRoute`, `Preflight`, `AutoFix`, `checkFeeQuoteHealth`
2. `internal/usecases/onchain_adapter_usecase.go`
- `GetStatus`, `sendTx`, and `call*` helpers
3. `internal/usecases/contract_config_audit_usecase.go`
- `runEVMOnchainChecks`, `call*View` helpers
4. `internal/infrastructure/blockchain/evm_client.go`
- most methods depend on live-like JSON-RPC interactions
5. `internal/interfaces/http/handlers`
- `wallet`, `chain`, `token`, `crosschain_policy`, `crosschain_config` branches

## Work Completed in This Iteration
- Added/expanded tests:
`payment_app_handler`, `webhook_handler`, `wallet_handler` extra error paths, command mains (`apikey-gen`, `genhash`, `hash-gen`).
- Stabilized test suite in restricted environment:
added safe-skip behavior when local socket bind fails for `httptest`/`miniredis`.
- Verified:
`go test ./... -coverprofile=coverage.out` and `make build`.


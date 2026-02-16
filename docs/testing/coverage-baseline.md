# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-16

## Summary
- Global total (all packages): **88.3%**
- `make build`: **PASS**
- Remaining major blockers:
`cmd/*` entrypoint mains and deep EVM admin/usecase branches that still require richer transaction/RPC simulation.

## Package Snapshot
- `cmd/admin-apikey`: **88.7%**
- `cmd/apikey-gen`: **88.9%**
- `cmd/genhash`: **85.7%**
- `cmd/hash-gen`: **90.9%**
- `cmd/server`: **98.0%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **87.5%**
- `internal/domain/errors`: **100.0%**
- `internal/infrastructure/blockchain`: **86.8%**
- `internal/infrastructure/datasources/postgres`: **81.8%**
- `internal/infrastructure/jobs`: **96.2%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **86.2%**
- `internal/interfaces/http/handlers`: **88.6%**
- `internal/interfaces/http/middleware`: **88.7%**
- `internal/interfaces/http/response`: **100.0%**
- `internal/usecases`: **86.6%**
- `pkg/crypto`: **81.8%**
- `pkg/jwt`: **87.5%**
- `pkg/logger`: **89.3%**
- `pkg/redis`: **87.9%**
- `pkg/utils`: **88.9%**

## High-Priority Gaps
1. `internal/usecases/crosschain_config_usecase.go`
- `Overview`, `RecheckRoute`, `Preflight`, `AutoFix`, `checkFeeQuoteHealth`
2. `internal/usecases/onchain_adapter_usecase.go`
- `GetStatus`, `sendTx`, and `call*` helpers
3. `internal/usecases/payment_usecase.go`
- `CreatePayment`, `buildTransactionData`, approval/fee quote helper branches
4. `internal/infrastructure/blockchain/evm_client.go`
- remaining branches around low-level client lifecycle and token calls
5. `internal/interfaces/http/handlers`
- `wallet`, `chain`, `token`, `crosschain_policy`, `crosschain_config` branches

## Work Completed in This Iteration
- Refactored `cmd/server/main.go` into `runMainProcess()` with injectable hooks and added `cmd/server/main_unit_test.go` to cover redis/db/session/server error + success paths.
- Added hookable branches and tests for redis middleware/session/client and EVM/onchain quote flows.
- Expanded repository create error-path tests and admin apikey command dependency tests.
- Verified:
`go test ./... -coverprofile=coverage.out` and `make build`.

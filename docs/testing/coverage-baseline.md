# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-16

## Summary
- Global total (all packages): **91.3%**
- `make build`: **PASS**
- Remaining major blockers:
`cmd/*` entrypoint mains and deep EVM admin/usecase branches that still require richer transaction/RPC simulation.

## Package Snapshot
- `cmd/admin-apikey`: **88.7%**
- `cmd/apikey-gen`: **96.3%**
- `cmd/genhash`: **100.0%**
- `cmd/hash-gen`: **90.9%**
- `cmd/server`: **98.4%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **87.5%**
- `internal/domain/errors`: **100.0%**
- `internal/infrastructure/blockchain`: **96.2%**
- `internal/infrastructure/datasources/postgres`: **81.8%**
- `internal/infrastructure/jobs`: **96.2%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **87.6%**
- `internal/interfaces/http/handlers`: **91.3%**
- `internal/interfaces/http/middleware`: **91.0%**
- `internal/interfaces/http/response`: **100.0%**
- `internal/usecases`: **91.7%**
- `pkg/crypto`: **100.0%**
- `pkg/jwt`: **96.0%**
- `pkg/logger`: **89.3%**
- `pkg/redis`: **92.4%**
- `pkg/utils`: **94.4%**

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
- Added internal branch tests for `OnchainAdapterUsecase` admin adapter resolver closures.
- Added EVM client nil-close branch test.
- Added repository branch tests for `TokenRepository.toEntity` (preloaded chain) and `RoutePolicyRepository.Create` defaults/generate-id path.
- Added dedicated `UpdateFeeConfig` error-branch handler tests (existing lookup fail, invalid JSON, invalid chain, invalid token format, token not found).
- Added repository branch tests for `ChainRepo` entity/rpc mapping, `PaymentRequest.GetByID` not-found path, `Team.toEntity` deleted-at path, and `User.List` search branch.
- Added handler gap tests for merchant apply/status, wallet set-primary unauthorized/internal error, and bridge-config update validation paths.
- Added deterministic hook-based error-branch tests for `pkg/crypto`, `pkg/jwt`, `pkg/redis`, and UUID fallback path in `pkg/utils`.
- Added internal helper tests for `ApiKeyUsecase` random/nonce/decrypt malformed branches.
- Added deeper call-view helper tests for pack error, RPC error, decode error, and type assertion mismatch paths.
- Added `buildTransactionData` approval success-path coverage (`approve` + `createPayment`) and unknown chain-type fallback.
- Added `ChainRepo.GetByCAIP2` branch tests (invalid input, reference fallback, malformed/not found).
- Added `SmartContractHandler` update full mutable-field branch test.
- Added `SmartContractHandler` create optional-fields test using UUID chain-id path.
- Added `CrosschainPolicyHandler` create test for default fallback derivation.
- Added `DualAuthMiddleware` tests for expired JWT and invalid JWT signature.
- Added `cmd/apikey-gen` RNG failure tests for first and second random-read branches.
- Verified:
`go test ./... -coverprofile=coverage.out` and `make build`.

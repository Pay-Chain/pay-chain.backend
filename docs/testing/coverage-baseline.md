# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-16

## Summary
- Global total (all packages): **99.1%**
- `make build`: **PASS**
- Remaining major blockers:
`cmd/*` entrypoint mains and deep EVM admin/usecase branches that still require richer transaction/RPC simulation.

## Package Snapshot
- `cmd/admin-apikey`: **100.0%**
- `cmd/apikey-gen`: **100.0%**
- `cmd/genhash`: **100.0%**
- `cmd/hash-gen`: **100.0%**
- `cmd/server`: **99.6%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **100.0%**
- `internal/domain/errors`: **100.0%**
- `internal/infrastructure/blockchain`: **100.0%**
- `internal/infrastructure/datasources/postgres`: **100.0%**
- `internal/infrastructure/jobs`: **100.0%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **98.7%**
- `internal/interfaces/http/handlers`: **99.7%**
- `internal/interfaces/http/middleware`: **99.4%**
- `internal/interfaces/http/response`: **100.0%**
- `internal/usecases`: **97.8%**
- `pkg/crypto`: **100.0%**
- `pkg/jwt`: **100.0%**
- `pkg/logger`: **100.0%**
- `pkg/redis`: **98.5%**
- `pkg/utils`: **100.0%**

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
- Added targeted handler branch tests for payment-config update/list/create gaps and payment list pagination normalization.
- Added auth usecase gap tests for register error matrix, verify-email mark-failed path, and refresh token user-lookup failure.
- Added internal `ApiKeyUsecase` tests for random source failure in key generation and decrypt path with invalid cipher key length.
- Added `ChainRepository.GetByCAIP2` direct/malformed branch tests.
- Added `AuthUsecase.ChangePassword` long-password hash-failure test.
- Added merchant repository status/db-error coverage branches.
- Added session-load hook (`loadSessionFromStore`) in auth middleware path to keep runtime behavior intact while enabling deterministic branch coverage without live Redis sockets.
- Added middleware internal coverage tests for trusted session auth path, optional signature validation branch, and expired bearer branch (`auth.go` / `dual_auth.go`).
- Added additional payment usecase branch tests for route-policy fallback defaults and `quoteBridgeFeeByType` RPC error responses.
- Added additional crosschain autofix branch tests for missing CCIP/LayerZero adapter contracts and Hyperbridge config failure path.
- Added ticker-driven expiry job test to cover `PaymentRequestExpiryJob.Start` processing path; package jobs now reaches 100%.
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

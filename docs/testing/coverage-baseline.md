# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-15

## Summary
- Global total (all packages): **53.6%**
- Kenaikan terbesar datang dari test repository + middleware.
- Bottleneck saat ini: `cmd/server`, `handlers`, `usecases`, `blockchain`.

## Package Coverage Snapshot
- `internal/interfaces/http/middleware`: **59.3%**
- `internal/usecases`: **48.3%**
- `internal/interfaces/http/handlers`: **47.0%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **87.5%**
- `internal/domain/errors`: **58.3%**
- `pkg/crypto`: **81.8%**
- `pkg/jwt`: **79.2%**
- `pkg/utils`: **83.3%**
- `pkg/logger`: **89.3%**
- `pkg/redis`: **64.6%**
- `internal/interfaces/http/response`: **100.0%**
- `cmd/apikey-gen`: **51.9%**
- `cmd/genhash`: **42.9%**
- `cmd/hash-gen`: **45.5%**
- `cmd/*`: **38.6%** (except `cmd/server`)
- `internal/infrastructure/blockchain`: **34.9%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **78.8%**
- `internal/infrastructure/datasources/postgres`: **81.8%**
- `internal/infrastructure/jobs`: **92.3%**
- `cmd/admin-apikey`: **14.3%**
- `cmd/server`: **5.6%**

## Priority Backlog (Highest Impact First)
1. `internal/usecases`
   - Business logic utama pembayaran/crosschain/onchain adapter.
   - Target awal: >60%, lalu naikkan ke 90%+, lalu 100%.
2. `internal/interfaces/http/handlers`
   - Endpoint matrix success/error/validation.
   - Target awal: >50%.
3. `internal/infrastructure/repositories`
   - Integration test DB (CRUD/filter/pagination/soft delete).
4. `internal/config` + `pkg/*`
   - Fast wins (pure function/utility tests).
5. `cmd/*`
   - Smoke/bootstrap test agar total `./...` bisa mencapai 100%.

## Immediate Action Executed
- Repository test matrix baru:
  - `bridge_config`, `fee_config`, `route_policy`, `layerzero_config`
  - `payment_event`, `payment_repo`, `payment_request_repo`, `background_job_repo`
  - `smart_contract_repo`
- Middleware test matrix baru:
  - auth helpers + role guard
  - auth middleware bearer/strict-mode branches
  - idempotency passthrough/error branch
- Handler tambahan:
  - `rpc_handler` full validation+success/error
  - validation branches `onchain_adapter`, `payment_app`, `webhook`
  - smart contract handler: CRUD/list/lookup success + error/validation branches
- Usecase helper tambahan:
  - `crosschain_config` helper guards (`deriveDestinationContractHex`, `checkFeeQuoteHealth`)
  - `contract_config_audit` helper parser/view-call invalid ABI branches
- Crosschain config handler:
  - `overview` success/error
  - validation branches `preflight`, `recheck`, `autofix`, `bulk` endpoints
- Onchain adapter usecase:
  - invalid source/destination branches for `SetDefaultBridgeType`, `SetHyperbridgeConfig`, `SetCCIPConfig`, `SetLayerZeroConfig`
  - internal call helpers (`callDefaultBridgeType`, `callHasAdapter`, `callGetAdapter`, `callHyperbridge*`, `callCCIP*`, `callLayerZero*`) covered via ABI pack-error path
- Crosschain config usecase:
  - `Preflight` invalid input branch
  - `AutoFix` invalid source input branch
- Contract config audit usecase:
  - `runEVMOnchainChecks` branch `RPC_MISSING`
  - `runEVMOnchainChecks` branch `RPC_CONNECT_FAILED`
- Auth handler:
  - `GetSessionExpiry` branch `No session`
  - `GetSessionExpiry` branch `Invalid proxy request`
  - validation branches:
    - `Register` invalid JSON
    - `Login` invalid JSON
    - `VerifyEmail` invalid payload
    - `RefreshToken` missing token
    - `GetMe` unauthorized context
    - `ChangePassword` unauthorized/internal/same-password

## Next Target
- Tambah test matrix untuk:
  - `crosschain_config_usecase`
  - `onchain_adapter_usecase`
  - `contract_config_audit_usecase`
  - branch error/fallback di `payment_usecase`
  - handler endpoint matrix (`internal/interfaces/http/handlers`) sampai >50%
  - `cmd/server` bootstrap path (env and startup failure branches)

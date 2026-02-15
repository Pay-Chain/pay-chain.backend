# Coverage Baseline (Current)

Generated from:
- `go test ./... -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-15

## Summary
- Global total (all packages): **23.5%**
- Reason utama: coverage masih tertahan di package tanpa test (`cmd/server`, `internal/infrastructure/*`) dan sebagian handler/usecase yang belum penuh.

## Package Coverage Snapshot
- `internal/interfaces/http/middleware`: **22.6%**
- `internal/usecases`: **40.5%**
- `internal/interfaces/http/handlers`: **16.8%**
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
- `internal/infrastructure/*`: **0.0%**
- `cmd/admin-apikey`: **14.3%**
- `cmd/server`: **0.0%**

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
- Batch awal test ditambahkan di `internal/usecases`:
  - `chain_resolver`
  - helper functions (`utils`)
  - `merchant_usecase`
  - `wallet_usecase`
  - `webhook_usecase`
  - `auth_usecase` (register/login path)
  - `payment_app_usecase` (chain validation + auto-create user/wallet error path)
  - `auth_usecase` (verify email, refresh token, token expiry, change password)
  - `payment_request_usecase` (validation + success + expiry + list/mark completed)
- Batch quick-win package test:
  - `internal/config`
  - `internal/domain/entities`
  - `internal/domain/errors`
  - `pkg/crypto`
  - `pkg/jwt`
  - `pkg/utils`

## Next Target
- Tambah test matrix untuk:
  - `crosschain_config_usecase`
  - `onchain_adapter_usecase`
  - `contract_config_audit_usecase`
  - branch error/fallback di `payment_usecase`
  - handler endpoint matrix (`internal/interfaces/http/handlers`)
  - repository integration tests (`internal/infrastructure/repositories`)

# Coverage Baseline (Current)

Generated from:
- `go test ./... -count=1 -coverprofile=coverage.out`
- `go tool cover -func=coverage.out`

Date:
- 2026-02-17

## Summary
- Global total (all statements): **100.0%**
- `make build`: **PASS**
- Status: **Coverage target achieved (100%)**

## Package Snapshot
- `cmd/admin-apikey`: **100.0%**
- `cmd/apikey-gen`: **100.0%**
- `cmd/genhash`: **100.0%**
- `cmd/hash-gen`: **100.0%**
- `cmd/server`: **100.0%**
- `internal/config`: **100.0%**
- `internal/domain/entities`: **100.0%**
- `internal/domain/errors`: **100.0%**
- `internal/infrastructure/blockchain`: **100.0%**
- `internal/infrastructure/datasources/postgres`: **100.0%**
- `internal/infrastructure/jobs`: **100.0%**
- `internal/infrastructure/models`: **100.0%**
- `internal/infrastructure/repositories`: **100.0%**
- `internal/interfaces/http/handlers`: **100.0%**
- `internal/interfaces/http/middleware`: **100.0%**
- `internal/interfaces/http/response`: **100.0%**
- `internal/usecases`: **100.0%** (based on `go tool cover -func` aggregate)
- `pkg/crypto`: **100.0%**
- `pkg/jwt`: **100.0%**
- `pkg/logger`: **100.0%**
- `pkg/redis`: **100.0%**
- `pkg/utils`: **100.0%**

## Verification Notes
- `go tool cover -func=coverage.out | grep -v "100.0%"` returns no uncovered function lines.
- Baseline updated to represent final full-coverage state.

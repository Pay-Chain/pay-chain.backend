# Backend Development Rules & Standards

## 1. Project Structure & Architecture
The project strictly follows **Clean Architecture** principles to ensure separation of concerns, testability, and maintainability.

### Layer Structure & Dependency Rules
*   **Dependency Rule**: Dependencies MUST flow **inwards** (Infrastructure -> Usecase -> Domain). The Domain layer MUST NOT depend on any other layer.
*   **`internal/domain`**: The core of the application. Contains business logic and interfaces. **NO external dependencies** (except standard library and basic types like UUID).
    *   `entities`: Pure Go structs representing database tables or core business objects.
    *   `repositories`: Interfaces defining database operations.
    *   `errors`: Domain-specific errors.
*   **`internal/usecases`**: Application business logic. Orchestrates data flow between entities and repositories.
    *   Depends **ONLY** on `domain`.
*   **`internal/infrastructure`**: Concrete implementations of external systems.
    *   `repositories`: Implementation of domain repository interfaces (e.g., GORM implementations).
    *   `blockchain`: Blockchain clients and interactions.
*   **`internal/interfaces`**: Entry points (Delivery Layer).
    *   `http`: REST API handlers, routers, and middleware.
*   **`internal/config`**: Configuration management.

### Detailed Project Tree
To clarify the structure, here is a detailed example of the file organization:

```
├── cmd/
│   └── app/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration loading (Env vars)
│   ├── domain/               # PURE GO, NO FRAMEWORKS
│   │   ├── entities/
│   │   │   ├── user.go       # User entity struct
│   │   │   └── payment.go    # Payment entity struct
│   │   ├── repositories/
│   │   │   ├── user_repo.go  # User repository interface
│   │   │   └── payment_repo.go
│   │   └── errors/
│   │       └── errors.go     # Domain-specific error definitions
│   ├── usecases/             # BUSINESS LOGIC
│   │   ├── payment/
│   │   │   ├── create.go     # CreatePaymentUsecase struct & logic
│   │   │   └── refund.go
│   │   └── user/
│   │       └── register.go
│   ├── infrastructure/       # EXTERNAL ADAPTERS
│   │   ├── database/
│   │   │   └── postgres.go   # DB Connection setup
│   │   ├── repositories/
│   │   │   ├── user_repo.go  # GORM implementation of UserRepo
│   │   │   └── payment_repo.go
│   │   └── blockchain/
│   │       ├── ethereum/
│   │       │   └── client.go # EthClient implementation
│   │       └── solana/
│   │           └── client.go
│   └── interfaces/           # DELIVERY LAYER
│       └── http/
│           ├── router.go     # Route definitions
│           ├── middleware/
│           │   └── auth.go
│           └── handlers/
│               ├── user.go   # User HTTP handlers
│               └── payment.go
└── pkg/                      # Shared library code (Utils)
    └── logger/
        └── logger.go
```

### Components per Endpoint
Each API endpoint **MUST** define explicit structs for its operation:
*   **Request Struct**: Defines expected input payload. MUST include `validate` tags for all fields.
*   **Response Struct**: Defines output payload. MUST match the API contract exactly.
*   **Model Struct**: Database entity representation (if different from Domain Entity).

## 2. Database Schema Standards
Every table in the database **MUST** include the following audit timestamp fields. This is non-negotiable.

```sql
created_at TIMESTAMP DEFAULT NOW()
updated_at TIMESTAMP DEFAULT NOW()
deleted_at TIMESTAMP -- For Soft Delete support
```

### Data Type Rules
*   **Primary Keys**: MUST use **UUID v7** (time-sortable) for better indexing performance compared to random UUID v4.
*   **Financial Values**:
    *   **Database**: MUST use `DECIMAL(precision, scale)` or `NUMERIC` for storage to ensure absolute precision. Floating point types (`FLOAT`, `REAL`) are strictly prohibited for monetary values.
    *   **Go Code**: MUST handle as `big.Int` or `string` to prevent float precision loss during calculations or serialization.
*   **Enums**: Prefer Database Enums (PostgreSQL `TYPE ... AS ENUM`) for fixed states (e.g., `payment_status_enum`) to enforce data integrity at the database level.
*   **Struct Tags**: Every struct field in Models, Requests, and Responses **MUST** have explicit tags defining its type and behavior (e.g., `json:"camelCase" validate:"required"`).

### Relation Loading
*   **Mandatory Preloading**: When fetching entities (e.g., in Repositories), if a Foreign Key field is not null (e.g., `chain_id`), the related entity (e.g., `Chain`) **MUST** be preloaded (e.g., `Preload("Chain")`).
*   **Prohibition**: Do not rely on manual subsequent fetches for related data to avoid N+1 query performance issues and potential data inconsistency.

## 3. Transactions & Data Consistency (MANDATORY)
**Strict Rule**: All write operations across tables must be atomic.
*   **Use Case Ownership**: The Usecase layer is the **ONLY** layer allowed to manage database transactions. Infrastructure layers must not start transactions implicitly.
*   **Transaction Scope**: No database write (INSERT, UPDATE, DELETE) is allowed outside a transaction scope.
*   **Repository Interface**: Methods modifying data MUST accept a transaction context (usually `context.Context` or a specific interface).

### Outbox Pattern (For Async Events)
To prevent inconsistent states between the Database and Message Queue (RabbitMQ):
*   **Atomic Write**: The Usecase **MUST** write the event payload to an `outbox` database table within the same database transaction as the business logic.
*   **Worker**: A separate worker process reads from the `outbox` table and publishes messages to RabbitMQ.
*   **Prohibition**: No RabbitMQ publish is allowed inside the HTTP request lifecycle directly.

## 4. Reliability, Concurrency & Locking
### Idempotency
*   **Mandatory Support**: All mutation endpoints (Payment, Transfer, Settlement, Webhooks) **MUST** support idempotency keys.
*   **Behavior**: Duplicate requests with the same key within a validity window MUST return the same response without re-executing business logic.
*   **Implementation**: Use Redis or DB with TTL unique constraints on the idempotency key.

### Concurrency Controls
*   **Locking**: All concurrent write access to critical data (e.g., account balances, ledgers) **MUST** be protected by:
    *   **Database Locking**: `SELECT ... FOR UPDATE` within a transaction.
    *   **Distributed Locking**: Redis Distributed Lock (Redlock).
*   **No Optimistic Assumptions**: Never assume a balance is valid without a lock during update.

### RabbitMQ Consumers
*   **Idempotent**: All consumers must handle duplicate messages gracefully.
*   **Ack Strategy**: Acknowledge message (`ack`) ONLY after successful processing.
*   **Retries**: Use localized retry logic with backoff before giving up.
*   **DLQ**: Poison messages (permanently failing) MUST be moved to a Dead Letter Queue for manual inspection.
*   **Stateless**: Consumers must not rely on local memory state between messages.

## 5. Performance, Observability & Error Handling
### Redis Usage Rules
*   **Not Source of Truth**: Redis is for caching only. Persistent data must reside in the Database.
*   **Keys**:
    *   MUST be namespaced: `paychain:{env}:{feature}:{id}`.
    *   MUST have a TTL (Time To Live).
*   **Access**: Wrapped in a dedicated Cache Service, never raw commands in handlers.

### Observability & Logging (MANDATORY)
*   **Comprehensive Logging**: **EVERY** Request and Response, whether Internal (HTTP Inbound) or External (Third-Party API Outbound), **MUST** be logged.
    *   **Inbound**: Log Method, URL, Status, Latency, Request Body (sanitized), Response Body (sanitized).
    *   **Outbound**: Log Target URL, Method, Status, Latency, Request Payload, Response Payload.
*   **Correlation ID**: All requests (HTTP & MQ) MUST carry a unique ID (`request_id` or `trace_id`) propagated across all internal services and logs.
*   **Structured Logging**: Logs must be structured (JSON) and include context (`user_id`, `tx_id`, `correlation_id`).
*   **Error Tracing**: Errors must be traceable from entry point to the root cause (database error/panic).

### Error Handling
*   **No Raw Errors**: Internal database or logic errors must **NEVER** leave the Usecase layer as raw strings or generic errors.
*   **Domain Errors**: Map all failures to Domain Errors with:
    *   **Code**: Unique error code string (e.g., `ERR_INSUFFICIENT_FUNDS`).
    *   **Message**: Public-safe message (e.g., "Insufficient funds").
    *   **HTTP Status**: Appropriate status code map (e.g., 400).

## 6. Blockchain Integration Rules
*   **Database-Driven Configuration**: All interaction data (Contracts, Chains, Tokens, RPCs) **MUST** be stored in the database. Hardcoding addresses or RPC URLs is strictly prohibited.
*   **Authorization**: Private keys are injected via environment variables/vault, NEVER stored in code/DB.
*   **Frontend Data Format**:
    *   **EVM Transactions**: Must return raw transaction data as **Hex String**.
    *   **SVM Transactions**: Must return raw transaction data as **Base64** serialized instruction.

## 7. Security Rules
*   **Webhooks**: All webhook endpoints MUST verify provider signatures.
*   **Replay Attacks**: Prevent replay attacks on public endpoints (using nonces or timestamps).
*   **Rate Limiting**: MUST be applied to all public endpoints.
*   **Logging**: Sensitive fields (PII, Credentials, Private Keys) MUST NEVER be logged.

## 8. Code Standards
*   **Naming**: Struct properties use **camelCase**.
*   **Formatting**: Standard `gofmt`.

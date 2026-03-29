# Development Guide: PaymentKita Backend

This document contains instructions for setting up, running, and developing the PaymentKita backend. For a high-level product overview and requirements, please refer to the [Main README](../README.md).

## Tech Stack

- **Framework**: [Gin](https://gin-gonic.com/)
- **ORM**: [SQLBoiler](https://github.com/volatiletech/sqlboiler)
- **Database**: PostgreSQL (via Supabase in production)
- **Cache**: Redis
- **Message Broker**: RabbitMQ
- **Blockchain Interface**: [go-ethereum](https://github.com/ethereum/go-ethereum)

## Getting Started

### Prerequisites
- Go 1.21+
- Docker & Docker Compose (for local infra)
- Python 3 (for some scripts)

### Installation
```bash
# Install dependencies
go mod download

# Run local infrastructure (Postgres, Redis, RabbitMQ)
docker-compose up -d
```

### Database Management
```bash
# Run database migrations
make migrate-up

# Generate SQLBoiler models (requires sqlboiler bin)
make generate-models
```

### Running the Application
```bash
# Start development server
make run

# Run tests
make test
```

## Project Structure

```
├── cmd/server/          # Application entry point
├── internal/
│   ├── domain/          # Entities, repository interfaces, errors
│   ├── usecases/        # Core business logic (Clean Architecture)
│   ├── infrastructure/  # Repositories, Redis, Blockchain clients
│   └── interfaces/      # HTTP handlers, Middleware, API responses
├── models/              # SQLBoiler generated models (do not edit)
├── migrations/          # Database migration files
├── pkg/                 # Common utilities (logger, jwt, etc.)
└── scripts/             # Administrative and helper scripts
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

```env
# Infrastructure
DATABASE_URL=postgres://user:pass@localhost:5432/paymentkita
REDIS_URL=redis://localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Security
JWT_SECRET=your-secret-key
ADMIN_API_KEY=your-admin-key
ENCRYPTION_KEY=32-byte-hex-key

# Blockchain
EVM_OWNER_PRIVATE_KEY=your-private-key
```

## Testing Strategy
We use standard Go testing with `testify`. Most use cases are tested with mocks located in `internal/usecases/mocks_test.go`.
- **Unit Tests**: `make test`
- **Coverage**: `make test-coverage`

# Pay-Chain Backend

Cross-chain stablecoin payment gateway API built with Go, Gin, and SQLBoiler.

## Tech Stack

- **Framework**: Gin
- **ORM**: SQLBoiler
- **Database**: PostgreSQL
- **Cache**: Redis
- **Message Broker**: RabbitMQ

## Getting Started

```bash
# Install dependencies
go mod download

# Run database migrations
make migrate-up

# Generate SQLBoiler models
make generate-models

# Start development server
make run

# Run tests
make test
```

## Project Structure

```
├── cmd/server/          # Application entry point
├── internal/
│   ├── domain/          # Domain entities and interfaces
│   ├── usecases/        # Business logic
│   ├── infrastructure/  # External services implementation
│   └── interfaces/      # HTTP handlers and middleware
├── models/              # SQLBoiler generated models
├── migrations/          # Database migrations
└── pkg/                 # Shared packages
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

```env
DATABASE_URL=postgres://user:pass@localhost:5432/paychain
REDIS_URL=redis://localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
JWT_SECRET=your-secret-key
```

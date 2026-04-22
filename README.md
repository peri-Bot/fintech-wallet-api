# Fintech Wallet API

Production-grade Go REST API for digital wallet operations. PostgreSQL streaming replication for read/write splitting. Redis rate limiting. ACID-safe transactions with row-level locking.

## Architecture

```
┌─────────────┐       ┌──────────────────┐       ┌──────────────────┐
│  wallet-api  │──W───▶│  postgres-primary │──rep──▶│  postgres-replica │
│  (Gin+GORM)  │──R───▶│     (writes)      │       │     (reads)       │
│   :8080      │       │     :5432         │       │     :5433         │
└──────┬───────┘       └──────────────────┘       └──────────────────┘
       │
       │ rate limit
       ▼
┌─────────────┐
│    redis     │
│    :6379     │
└─────────────┘
```

## Tech Stack

| Layer            | Tech                                  |
| ---------------- | ------------------------------------- |
| Language         | Go 1.21                               |
| Router           | Gin                                   |
| ORM              | GORM + DBResolver                     |
| Primary DB       | PostgreSQL 16 (Bitnami)               |
| Read Replica     | PostgreSQL 16 (streaming replication) |
| Rate Limiter     | Redis 7 + fixed-window counter        |
| Containerization | Docker + Docker Compose               |
| Dev Environment  | Nix Flakes                            |

## API Endpoints

| Method | Endpoint                   | Description                 |
| ------ | -------------------------- | --------------------------- |
| `GET`  | `/health`                  | Health check                |
| `POST` | `/api/v1/deposit`          | Add funds to wallet         |
| `POST` | `/api/v1/withdraw`         | Deduct funds (row-locked)   |
| `GET`  | `/api/v1/balance/:user_id` | Read balance (from replica) |

### Request Body (deposit/withdraw)

```json
{
  "user_id": "user1",
  "amount": "100.00",
  "currency": "USD"
}
```

## Rate Limiting

5 requests/minute per user. Fixed-window counter via Redis `INCR` + `EXPIRE`.

Response headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`.

Exceeding limit returns `429 Too Many Requests`.

## Race Condition Prevention

Withdrawals use `SELECT ... FOR UPDATE` — acquires exclusive row lock. Concurrent withdrawals block until lock releases. Prevents double-spending.

## Quick Start

```bash
# Nix users
direnv allow

# Start all services
docker compose up -d

# Deposit
curl -X POST http://localhost:8080/api/v1/deposit \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user1", "amount": "100.00"}'

# Balance
curl http://localhost:8080/api/v1/balance/user1

# Withdraw
curl -X POST http://localhost:8080/api/v1/withdraw \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user1", "amount": "50.00"}'

# Verify replica is in recovery mode
docker exec wallet-pg-replica psql -U wallet_user -d wallet_db \
  -c "SELECT pg_is_in_recovery();"
```

## Project Structure

```
├── main.go                   # Entry point, router setup
├── config/config.go          # Env-based configuration
├── database/database.go      # GORM + DBResolver (read/write routing)
├── models/wallet.go          # UserWallet model (decimal precision)
├── handlers/wallet.go        # Deposit, Withdraw, Balance handlers
├── middleware/ratelimit.go   # Redis rate limiter middleware
├── docker-compose.yml        # PG primary + replica + Redis + API
├── Dockerfile                # Multi-stage build (Alpine)
├── flake.nix                 # Nix dev environment
└── .envrc                    # direnv activation
```

## Environment Variables

| Variable          | Default          | Description             |
| ----------------- | ---------------- | ----------------------- |
| `PRIMARY_DB_HOST` | `localhost`      | Primary PostgreSQL host |
| `PRIMARY_DB_PORT` | `5432`           | Primary PostgreSQL port |
| `REPLICA_DB_HOST` | `localhost`      | Replica PostgreSQL host |
| `REPLICA_DB_PORT` | `5433`           | Replica PostgreSQL port |
| `DB_USER`         | `wallet_user`    | Database username       |
| `DB_PASSWORD`     | `wallet_secret`  | Database password       |
| `DB_NAME`         | `wallet_db`      | Database name           |
| `REDIS_ADDR`      | `localhost:6379` | Redis address           |
| `SERVER_PORT`     | `8080`           | API server port         |

## License

MIT

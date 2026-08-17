# Scenic Ticket

Scenic Ticket is a small ticket-sales and gate-checking application. The Vue frontend calls a Gin API, and production data is stored in PostgreSQL through GORM.

## Core lifecycle

The ticket lifecycle is implemented in `backend/ticket`:

1. Selling a ticket validates the ticket type, phone quota, daily capacity, and time-slot capacity in one transaction.
2. Refunding an unused ticket restores the phone quota and both capacity counters in the same transaction.
3. Check-in atomically changes the ticket state, records the gate event, and updates admission counters.
4. Check-out atomically records the exit on the ticket and in the gate-event history.

HTTP handlers map service errors consistently:

- invalid input: `400 Bad Request`
- missing resource: `404 Not Found`
- business-state or capacity conflict: `409 Conflict`
- request cancellation or timeout: `408 Request Timeout`
- storage failure: `500 Internal Server Error`

The service receives a transaction repository, clock, and ticket-number generator. Tests use SQLite and deterministic dependencies; the running application uses PostgreSQL, the system clock, and cryptographically random ticket numbers.

Calendar dates are evaluated in `Asia/Shanghai`. Date-only database keys retain the existing UTC-midnight representation, while gate events and hourly statistics use Shanghai-local day boundaries.

## Run locally

Requirements:

- Docker with Compose
- or Go 1.21+, Node.js 18+, and PostgreSQL 15+

Start the complete stack:

```bash
docker compose up --build
```

The API listens on `http://localhost:8741`, and the frontend listens on `http://localhost:5173`.

Default development users are created only in an empty database:

| Username | Password | Role |
| --- | --- | --- |
| `admin` | `scenic2024` | administrator |
| `seller1` | `sell123` | seller |
| `seller2` | `sell123` | seller |

## Backend configuration

The backend accepts these environment variables:

| Variable | Default |
| --- | --- |
| `DB_HOST` | `localhost` |
| `DB_PORT` | `5432` |
| `DB_USER` | `scenic` |
| `DB_PASSWORD` | `scenic2024` |
| `DB_NAME` | `scenic_db` |
| `JWT_SECRET` | `scenic-ticket-jwt-secret-2024` |
| `PORT` | `8741` |

Run only the API after PostgreSQL is available:

```bash
cd backend
go run .
```

## Verification

Run the focused lifecycle tests:

```bash
cd backend
go test ./ticket ./handlers -count=1
```

Run the complete backend verification:

```bash
cd backend
go test ./...
go build ./...
go vet ./...
```

Run the PostgreSQL migration and concurrency integration tests against the Compose database:

```bash
docker compose up -d postgres
cd backend
go test -race -tags=integration ./database -count=1
```

Set `TEST_POSTGRES_DSN` to use another PostgreSQL 15 database. Each integration test creates and removes its own schema.

Validate deployment configuration and build the frontend:

```bash
docker compose config --quiet
cd frontend
npm ci
npm run build
```

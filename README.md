# Sentinel

Sentinel is a small, modular authentication and authorization microservice written in Go. It's designed to provide a central auth layer for applications (web, mobile, desktop) and to be easy to extend for production use.

Core features

- User registration (username/email + bcrypt-hashed password)
- Login and JWT issuance with configurable expiration
- JWT middleware for protecting routes
- Pluggable persistence (SQLite for dev; Postgres/MySQL supported in production)
- Environment-based configuration (PORT, DATABASE_URL, JWT_SECRET)

Repository layout

- `cmd/server/` — service entrypoint
- `internal/config/` — configuration loading
- `internal/models/` — domain models (User, etc.)
- `internal/store/` — persistence interfaces and implementations
- `internal/auth/` — password hashing, JWT helpers
- `internal/handlers/` — HTTP handlers
- `internal/middleware/` — auth middleware
- `internal/server/` — HTTP server wiring
- `migrations/` — DB migrations
- `scripts/` — helper scripts (bootstrap, migrations)
- `docs/` — architecture and usage docs

Quickstart (development)

1. Copy the example env file and edit values if needed:

```powershell
copy .env.example .env
```

2. Build and run locally:

```powershell
go build ./...
go run ./cmd/server
```

3. The server currently contains placeholders and stubs. See `docs/USAGE.md` for the next commands for running migrations and interacting with the API when implemented.

Getting involved

Check `CONTRIBUTING.md` for contribution guidelines and run the tests locally. The project favors small, well-tested PRs and clear commit messages.

Notes about configuration

- `internal/config.Load()` now reads values directly from environment variables and does not provide defaults or write files. This enforces explicit configuration in CI and production.
- Use `.env` for local development only; never commit `.env` to the repository.
- `.env.example` contains placeholders; copy it to `.env` and fill values before running locally.
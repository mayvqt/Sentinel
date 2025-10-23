# Sentinel Architecture

Overview
 
Sentinel is split into a small set of well-scoped packages under `internal/` with a thin `cmd/server` entrypoint. The service is designed to be embedded into larger systems or run as a standalone auth microservice.

Design goals

- Security: passwords are hashed with bcrypt and tokens are signed with a strong secret.
- Modularity: storage and auth mechanisms are abstracted by interfaces so implementations can be swapped (SQLite for local dev, Postgres for production).
- Observability: HTTP handlers should emit structured logs and metrics (not implemented in scaffold).

Package responsibilities

- `internal/config` — read environment variables or configuration files and provide typed access across the app.
- `internal/models` — domain models such as `User` and any DTOs used by handlers.
- `internal/store` — persistence layer interface plus optional implementations (e.g., SQLite). Keep queries and migrations here.
- `internal/auth` — password hashing and JWT generation/verification. Use bcrypt for hashing and a secure HMAC (or RSA) signing key for tokens.
- `internal/handlers` — HTTP route handlers for registration, login, profile, and admin operations.
- `internal/middleware` — middleware such as JWT verification and role-based access checks.
- `internal/server` — small orchestration code to wire router, middlewares, and dependency injection.

Data flow (user login example)

1. Client POSTs credentials to `/login`.
2. Handler uses `store` to find user record by username/email.
3. `auth` compares bcrypt hash with provided password.
4. On success, `auth` creates a JWT (user id, role claims) and returns it to the client.
5. Client includes JWT in `Authorization: Bearer <token>` header for protected endpoints.

Security considerations

- Secrets (JWT_SECRET, DB credentials) must not be checked into the repository. Use environment variables or a secrets manager.
- Always validate token expiry and consider token revocation strategies (refresh tokens or sessions table).

Configuration and security notes

- The config loader intentionally reads from environment variables only and does not provide defaults. This helps avoid accidental insecure defaults in production.
- Keep `.env` local and private. Use `.env.example` as a template for developers.


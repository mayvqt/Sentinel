# Sentinel

Sentinel is a compact, production-minded authentication and authorization microservice written in Go. This single README consolidates quickstart, configuration, API reference, development notes, contributing guidelines, and security considerations.

---

## Quickstart (development)

1. Copy `.env.example` to `.env` and set values for local development:

```powershell
copy .env.example .env
# Edit .env and set a secure JWT_SECRET and (optional) DATABASE_URL
```

2. Build and run the service:

```powershell
go build .
.\Sentinel.exe
```

3. Verify the service is running:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET
```

---
## API reference

- `POST /api/auth/register` — Register a new user. JSON body: `{ "username": "alice", "email": "alice@example.com", "password": "Secur3P@ss!" }`. Returns 201 on success.
- `POST /api/auth/login` — Login and return a JWT. JSON body: `{ "username": "alice", "password": "Secur3P@ss!" }`. Returns `{ "token": "..." }` on success.
- `GET /api/auth/profile` — Protected endpoint. Requires header `Authorization: Bearer <token>`; returns the user's profile.
- `GET /health` — Health check; returns basic status and version.

### Example (PowerShell)

```powershell
# Register
Invoke-RestMethod -Uri "http://localhost:8080/api/auth/register" -Method POST -ContentType "application/json" -Body '{"username":"alice","email":"alice@example.com","password":"Secur3P@ss!"}'

# Login
$body = @{ username = "alice"; password = "Secur3P@ss!" } | ConvertTo-Json
$resp = Invoke-RestMethod -Uri "http://localhost:8080/api/auth/login" -Method POST -ContentType "application/json" -Body $body
$token = $resp.token

# Profile
Invoke-RestMethod -Uri "http://localhost:8080/api/auth/profile" -Method GET -Headers @{ Authorization = "Bearer $token" }
```

---

## Configuration

Sentinel reads configuration from environment variables. Use `.env` only for local development.

- `PORT` — server port (default: `8080`)
- `DATABASE_URL` — connection string (e.g. `sqlite://./sentinel.db`)
- `JWT_SECRET` — required HMAC secret used to sign JWTs (must be kept secret)

Ensure `JWT_SECRET` is set before running the service in development or production.

---

## Development & testing

- Run unit tests with `go test ./...`.
- The project includes a SQLite implementation for development; for CI and production use a managed SQL database.
- Avoid committing `.env` or any secrets; use your CI/CD secret management instead.

---

## Contributing

We welcome contributions. Keep PRs small and focused. A suggested workflow:

1. Fork and create a feature branch.
2. Run `go fmt` and `go test` locally.
3. Open a PR with a clear description and any necessary migration steps.

Security: If you discover a security issue, please contact the maintainers privately.

---

## License

MIT. See `LICENSE` for full text.
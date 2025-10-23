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
# Sentinel

Sentinel is a compact, production-minded authentication and authorization microservice written in Go. This README is the canonical project document: quickstart, configuration, API reference, development notes, and contribution guidance.

---

## Quickstart (development)

1. Copy the example environment file and set values (local development only):

```powershell
copy .env.example .env
# Edit `.env` and set a secure JWT_SECRET and (optional) DATABASE_URL
```

Minimal example `.env` (do not commit):

```env
PORT=8080
DATABASE_URL=sqlite://./sentinel.db
JWT_SECRET=replace-with-a-random-secret-or-base64
```

2. Build and run the service (either):

```powershell
go build .
.\Sentinel.exe
```

or for rapid iteration:

```powershell
go run .
```

3. Verify the service is running:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET
```

---

## API reference

All endpoints are under `/api/auth` unless otherwise noted.

- POST /api/auth/register — Create a new user.
	- Request: JSON `{ "username": "alice", "email": "alice@example.com", "password": "Secur3P@ss!" }`
	- Response: 201 Created on success; 4xx on validation errors.

- POST /api/auth/login — Authenticate and receive a JWT.
	- Request: JSON `{ "username": "alice", "password": "Secur3P@ss!" }`
	- Response: 200 OK `{ "token": "<jwt>" }` on success; 401 on credentials failure.

- GET /api/auth/profile — Protected; returns authenticated user's profile.
	- Request: Header `Authorization: Bearer <token>`
	- Response: 200 OK with user profile; 401 if token is missing/invalid.

- GET /health — Health check (non-authenticated).

### Examples

PowerShell (Windows):

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

curl (POSIX):

```bash
# Register
curl -sS -X POST http://localhost:8080/api/auth/register \
	-H "Content-Type: application/json" \
	-d '{"username":"alice","email":"alice@example.com","password":"Secur3P@ss!"}'

# Login (capture token)
token=$(curl -sS -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d '{"username":"alice","password":"Secur3P@ss!"}' | jq -r .token)

# Profile
curl -sS -H "Authorization: Bearer $token" http://localhost:8080/api/auth/profile
```

---

## Configuration

Sentinel reads configuration from environment variables. Use `.env` only for local development; do not commit secrets.

- `PORT` — server port (default: `8080`).
- `DATABASE_URL` — data store connection string (e.g. `sqlite://./sentinel.db`).
- `JWT_SECRET` — HMAC secret used to sign tokens (required). Use a high-entropy random value (32+ bytes) or base64-encoded secret.

Recommended: generate a secure JWT secret locally, e.g. using OpenSSL or a secure random generator.

---

## Development & testing

- Run unit tests with:

```bash
go test ./...
```

- Use the provided SQLite backend for local development. For CI and production, prefer a managed SQL database and appropriate connection pooling.
- Keep `.env` in `.gitignore` and provide secrets via your CI/CD or secrets manager.

---

## Contributing

Contributions are welcome. Keep pull requests small and focused. Suggested workflow:

1. Fork, create a branch, and implement your change.
2. Run `go fmt` and `go test` locally.
3. Open a PR with a clear description and tests for new behavior.

If you discover a security issue, please contact the maintainers privately rather than opening a public issue.

---

## License

MIT. See `LICENSE` for full text.

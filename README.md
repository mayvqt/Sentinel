# Sentinel

Sentinel is a small, secure Go-based authentication microservice. This repository is configured to run locally and in containers with optional TLS support.

## Quick Docker (local) — build and run

1. Ensure you have Docker and Docker Compose installed.

2. Create a strong JWT secret and (optionally) TLS certs. For local testing you can use the included PowerShell script to generate self-signed certs on Windows:

```powershell
.
scripts\generate-cert.ps1 -OutDir .\certs -CommonName localhost
```

3. Build and run with Docker Compose (example using TLS disabled):

```powershell
$env:JWT_SECRET = 'a-very-strong-32-char-or-longer-secret'
# Sentinel

Sentinel is a focused, secure Go-based authentication microservice with practical examples for running locally, in containers, or behind TLS-terminating reverse proxies.

## Quickstart

Run locally (development):

```powershell
setx JWT_SECRET "replace-with-strong-secret"
go run ./cmd/server
```

Run with Docker Compose (simple local):

```powershell
# Sentinel

Sentinel is a focused, secure Go-based authentication microservice with practical examples for running locally, in containers, or behind TLS-terminating reverse proxies.

## Features

- **JWT-based authentication** with access and refresh tokens
- **Secure password hashing** with bcrypt (cost 12)
- **Rate limiting** (configurable per-endpoint)
- **CORS support** with allowlist (no wildcard defaults)
- **Request body size limits** to prevent DoS
- **Security headers** (CSP, X-Frame-Options, HSTS)
- **SQLite or in-memory storage**
- **TLS support** (optional, HTTP default for reverse proxy setups)
- **Structured logging** with request IDs

## Quickstart (works out of the box)

Run locally (development):

```powershell
# Set JWT secret (required)
$env:JWT_SECRET = 'your-very-strong-32-char-or-longer-secret'

# Run the server
go run ./cmd/server
```

The service listens on `http://localhost:8080` by default and uses an in-memory store.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | ✅ Yes | - | JWT signing secret (recommend ≥32 characters) |
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | No | in-memory | SQLite path (e.g., `sqlite://./data.db`) |
| `CORS_ALLOWED_ORIGINS` | No | `http://localhost:3000,http://localhost:8080` | Comma-separated allowed origins |
| `TLS_ENABLED` | No | `false` | Enable HTTPS/TLS (use only if not behind reverse proxy) |
| `TLS_CERT_FILE` | No | - | Path to TLS certificate (required if TLS enabled) |
| `TLS_KEY_FILE` | No | - | Path to TLS private key (required if TLS enabled) |

## API Endpoints & Usage

Base URL: `http://localhost:8080`

### 1. Register a New User

**Endpoint:** `POST /api/auth/register`

**Request:**
```powershell
curl -X POST http://localhost:8080/api/auth/register `
  -H "Content-Type: application/json" `
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "SecureP@ss123"
  }'
```

**Response (Success):**
```json
{
  "message": "User registered successfully",
  "user_id": 1
}
```

**Requirements:**
- Username: 3-32 characters, alphanumeric/underscore/hyphen only
- Email: valid email format
- Password: ≥8 characters, must include uppercase, lowercase, number, and special character

---

### 2. Login

**Endpoint:** `POST /api/auth/login`

**Request:**
```powershell
curl -X POST http://localhost:8080/api/auth/login `
  -H "Content-Type: application/json" `
  -d '{
    "username": "alice",
    "password": "SecureP@ss123"
  }'
```

**Response (Success):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600
}
```

- `access_token`: Use for authenticated requests (valid 1 hour)
- `refresh_token`: Use to obtain new access tokens (valid 7 days)

---

### 3. Get User Profile (Protected)

**Endpoint:** `GET /api/auth/profile`

**Request:**
```powershell
# Replace YOUR_ACCESS_TOKEN with the token from login
curl http://localhost:8080/api/auth/profile `
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Response (Success):**
```json
{
  "id": 1,
  "username": "alice",
  "email": "alice@example.com",
  "role": "user",
  "created_at": "2025-10-23T12:00:00Z"
}
```

---

### 4. Refresh Access Token

**Endpoint:** `POST /api/auth/refresh`

**Request:**
```powershell
curl -X POST http://localhost:8080/api/auth/refresh `
  -H "Content-Type: application/json" `
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

**Response (Success):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600
}
```

---

### 5. Health Check

**Endpoint:** `GET /health`

**Request:**
```powershell
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "ok"
}
```

## Complete Example Workflow

```powershell
# 1. Start the server
$env:JWT_SECRET = 'my-super-secret-jwt-key-32chars'
go run ./cmd/server

# 2. Register a user
$registerResponse = curl -X POST http://localhost:8080/api/auth/register `
  -H "Content-Type: application/json" `
  -d '{"username":"bob","email":"bob@example.com","password":"MyP@ssw0rd123"}' | ConvertFrom-Json

# 3. Login
$loginResponse = curl -X POST http://localhost:8080/api/auth/login `
  -H "Content-Type: application/json" `
  -d '{"username":"bob","password":"MyP@ssw0rd123"}' | ConvertFrom-Json

# 4. Get profile using access token
$accessToken = $loginResponse.access_token
curl http://localhost:8080/api/auth/profile `
  -H "Authorization: Bearer $accessToken"

# 5. Refresh token when access token expires
curl -X POST http://localhost:8080/api/auth/refresh `
  -H "Content-Type: application/json" `
  -d "{`"refresh_token`":`"$($loginResponse.refresh_token)`"}"
```

## Docker

Run with Docker Compose:

```powershell
$env:JWT_SECRET = 'your-strong-secret'
docker-compose up --build
```

The `Dockerfile` is multi-stage and produces a minimal runtime image. For reverse proxy setups, run with `TLS_ENABLED=false`.

## Reverse Proxy Examples

Included configurations for production TLS termination:

- **Caddy**: `docker-compose.caddy.yml` — automatic HTTPS via Let's Encrypt
- **Traefik**: `docker-compose.traefik.yml` — Docker-aware ACME integration
- **Nginx**: `docker-compose.nginx.yml` — traditional setup with certbot
- **HAProxy**: `docker-compose.haproxy.yml` — TLS termination with PEM files

Example (Caddy):
```powershell
$env:JWT_SECRET = 'your-strong-secret'
docker-compose -f docker-compose.caddy.yml up --build
```

## Security

- **CORS**: Set `CORS_ALLOWED_ORIGINS` in production (defaults to localhost)
- **Rate Limiting**: 5 requests per 2 seconds for auth endpoints
- **Request Size Limits**: 1MB max body size on auth endpoints
- **Token Validation**: Explicit expiry and clock skew checks
- **Password Requirements**: Strong password validation enforced
- **Bcrypt**: Cost factor 12 for password hashing

## Troubleshooting

**"JWT_SECRET is required"**
- Set the `JWT_SECRET` environment variable before starting

**"Username already exists"**
- Try a different username or login with existing credentials

**"Token expired"**
- Use the refresh token to obtain a new access token

**CORS errors in browser**
- Set `CORS_ALLOWED_ORIGINS` to include your frontend origin

## License

MIT
docker-compose up --build
```

By default the service listens on port 8080.

## Docker notes

- The `Dockerfile` is multi-stage and produces a minimal runtime image.
- If a reverse proxy terminates TLS, run Sentinel with `TLS_ENABLED=false` and let the proxy handle certificates.
- If Sentinel should serve TLS itself, mount certificates into the container and set `TLS_CERT_FILE` and `TLS_KEY_FILE`.

## Reverse proxy options

Included examples (in the repo):

- Caddy: `Caddyfile` + `docker-compose.caddy.yml` — automatic HTTPS via ACME and simple reverse-proxying.
- Traefik: `docker-compose.traefik.yml` — Docker-friendly ACME integration.
- Nginx: `nginx/sentinel.conf` + `docker-compose.nginx.yml` — traditional proxy with certs from `/etc/letsencrypt`.
- HAProxy: `haproxy.cfg` + `docker-compose.haproxy.yml` — TLS termination with PEM files.

Prefer proxy termination for certificates; it simplifies renewal and scaling.

## Certbot (host or container)

Use `certbot certonly --webroot` or standalone mode to obtain certificates and mount them into your proxy or container. Use `--staging` for testing.

Example (container, staging):

```powershell
docker run --rm -p 80:80 -v "%cd%\letsencrypt:/etc/letsencrypt" certbot/certbot certonly --standalone --non-interactive --agree-tos --staging -m you@example.com -d yourdomain.example
```

## API examples

Register:

```powershell
curl -X POST http://localhost:8080/v1/register -H "Content-Type: application/json" -d '{"username":"alice","password":"P@ssw0rd","email":"alice@example.com"}'
```

Login:

```powershell
curl -X POST http://localhost:8080/v1/login -H "Content-Type: application/json" -d '{"username":"alice","password":"P@ssw0rd"}'
```

Protected endpoint (replace ACCESS_TOKEN):

```powershell
curl -H "Authorization: Bearer ACCESS_TOKEN" http://localhost:8080/v1/profile
```

## Environment variables

- `JWT_SECRET` (required) — a strong secret (recommend >=32 bytes).
- `PORT` (optional) — default 8080.
- `DATABASE_URL` (optional) — e.g. `sqlite://./data.db`. Omit to use in-memory store for development.
- `TLS_ENABLED`, `TLS_CERT_FILE`, `TLS_KEY_FILE` — configure if the app should serve TLS directly.

## Security checklist

- Always use HTTPS in production (prefer proxy termination with ACME-capable proxies).
- Protect `JWT_SECRET` with a secrets manager.
- Use CA-signed certs in production and monitor renewals.

## Troubleshooting

- If ACME/Let's Encrypt challenges fail, check that ports 80/443 are reachable and DNS is correct.
- Use staging endpoints during testing to avoid rate limits.

## License

MIT

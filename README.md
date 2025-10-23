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
$env:TLS_ENABLED = 'false'
docker-compose up --build
```

4. To enable TLS, mount certs and set TLS_ENABLED to true. Example (PowerShell):

```powershell
$env:JWT_SECRET = 'a-very-strong-32-char-or-longer-secret'
$env:TLS_ENABLED = 'true'
docker-compose up --build
```

By default the container exposes port 8080.

## Dockerfile notes

- Multi-stage build: compiles the Go binary in an Alpine-based builder and produces a minimal distroless image.
- TLS certificates should be mounted into the container and referenced by `TLS_CERT_FILE` and `TLS_KEY_FILE` environment variables.

## Environment variables

- JWT_SECRET (required) — Use a strong secret of at least 32 characters.
# Sentinel

Sentinel is a small, secure Go-based authentication microservice focused on simplicity and secure defaults. This repository contains examples to run the service directly, in Docker, or behind a TLS-terminating reverse proxy.

## Quickstart (local)

Prerequisites: Go 1.20+, Docker (optional), Docker Compose (optional).

1. Build and run locally (development):

```powershell
setx JWT_SECRET "a-very-strong-32-char-or-longer-secret"
go run ./cmd/server
```

2. Run with Docker Compose (simple local):

```powershell
$env:JWT_SECRET = 'a-very-strong-32-char-or-longer-secret'
docker-compose up --build
```

By default the service listens on port 8080.

## Docker (images and compose)

- The provided `Dockerfile` is multi-stage: it compiles a static Linux binary and produces a minimal runtime image.
- Use `TLS_ENABLED=false` when the container is behind a TLS-terminating proxy.
- Mount certificate files into the container and set `TLS_CERT_FILE` and `TLS_KEY_FILE` only if you want the app itself to serve TLS.

Example (run container directly):

```powershell
docker build -t sentinel:local .
docker run -e JWT_SECRET=$env:JWT_SECRET -p 8080:8080 sentinel:local
```

## Reverse proxy & TLS patterns

Recommended: terminate TLS at the proxy. This centralizes certificate automation and renewal and keeps the app simple.

- Caddy: included `Caddyfile` + `docker-compose.caddy.yml` — Caddy will automatically provision certificates via Let's Encrypt and reverse-proxy to the app.
- Traefik: see `docker-compose.traefik.yml` — Traefik integrates with Docker and ACME to automate cert management.
- Nginx: see `nginx/sentinel.conf` and `docker-compose.nginx.yml` — classic setup; pair with Certbot (host or container) for certificate management.
- HAProxy: see `haproxy.cfg` and `docker-compose.haproxy.yml` — show how to provide PEM files for TLS termination.

Example: run Caddy with compose (Caddy terminates TLS, Sentinel runs HTTP):

```powershell
$env:JWT_SECRET = 'a-very-strong-32-char-or-longer-secret'
docker-compose -f docker-compose.caddy.yml up --build
```

## Certbot (obtaining certificates)

If you prefer to run certbot directly (host or container):

- Use `certbot certonly --webroot` or the standalone mode to obtain certificates.
- Store certificates under `/etc/letsencrypt` and mount them into your reverse proxy or the Sentinel container.
- Use staging for testing (`--staging`) to avoid LetsEncrypt rate limits.

Example: certbot in a container (first-run example):

```powershell
# Run once to obtain certs (use --staging to test)
docker run --rm -p 80:80 -v "%cd%\letsencrypt:/etc/letsencrypt" certbot/certbot certonly --standalone --non-interactive --agree-tos -m you@example.com -d yourdomain.example
```

Then mount `letsencrypt` into your proxy or sentinel container.

## API (quick examples)

The service exposes typical auth endpoints (see `internal/handlers`): register, login, refresh, and a protected profile endpoint.

Example curl (register):

```powershell
curl -X POST http://localhost:8080/v1/register -H "Content-Type: application/json" -d '{"username":"alice","password":"P@ssw0rd","email":"alice@example.com"}'
```

Login example:

```powershell
curl -X POST http://localhost:8080/v1/login -H "Content-Type: application/json" -d '{"username":"alice","password":"P@ssw0rd"}'
```

Protected request (replace ACCESS_TOKEN with the token returned by login):

```powershell
curl -H "Authorization: Bearer ACCESS_TOKEN" http://localhost:8080/v1/profile
```

## Security checklist

- Always run in HTTPS in production; prefer proxy termination (Caddy/Traefik) or use managed TLS.
- Keep `JWT_SECRET` strong and store it in a secrets manager. Do not commit it or store in plaintext in repos.
- Use CA-signed certificates for production.
- Monitor certificate renewal and ensure services reload or use proxies that auto-reload.

## Troubleshooting

- If ACME challenges fail, ensure ports 80/443 are not blocked and DNS points to the server.
- Use Let's Encrypt staging to test flows without hitting rate limits.

## License

MIT

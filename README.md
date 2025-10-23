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
$env:JWT_SECRET = 'replace-with-strong-secret'
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
